// Package accountnotifier is used for Pgp Reencryption
package accountnotifier

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/aws"
	"github.com/app-sre/go-qontract-reconcile/pkg/pgp"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/state"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/app-sre/go-qontract-reconcile/pkg/vault"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/mail"
	"github.com/pkg/errors"

	parmor "github.com/ProtonMail/gopenpgp/v2/armor"
	"github.com/ProtonMail/gopenpgp/v2/constants"
	phelper "github.com/ProtonMail/gopenpgp/v2/helper"
)

// IntegrationName is the name of the integration
var IntegrationName = "account-notifier"

type status int

const (
	// reencrypt is the status for reencrypting the pgp key
	reencrypt status = iota
	// skip is the status for skipping the pgp key
	skip
	// notifyExpired is the status for notifying the pgp key is expired
	notifyExpired
)

type getUsers func(context.Context) (*UsersResponse, error)
type getPgpReencryptSettings func(context.Context) (*PgpReencryptSettingsResponse, error)
type getSMTPSettings func(context.Context) (*SmtpSettingsResponse, error)
type sendEmail func(context.Context, *notify.Notify, string, string) error
type setFailedState func(context.Context, state.Persistence, string, notification) error
type rmFailedState func(context.Context, state.Persistence, string) error

// AccountNotifier is the account notifier integration used for pgp reencryption
type AccountNotifier struct {
	state            state.Persistence
	vault            *vault.Client
	appSrePGPKeyPath string
	vaultImportPath  string
	vaultExportPath  string

	smtpauth            smtpAuth
	getuserFunc         getUsers
	getReencryptFunc    getPgpReencryptSettings
	getSMTPSettingsFunc getSMTPSettings
	sendEmailFunc       sendEmail
	setFailedStateFunc  setFailedState
	rmFailedStateFunc   rmFailedState
}

type smtpAuth struct {
	mailAddress string
	username    string
	password    string
	server      string
	port        string
}

// NewAccountNotifier create a new account notifier
func NewAccountNotifier() *AccountNotifier {
	notifier := AccountNotifier{
		getuserFunc: func(ctx context.Context) (*UsersResponse, error) {
			return Users(ctx)
		},
		getReencryptFunc: func(ctx context.Context) (*PgpReencryptSettingsResponse, error) {
			return PgpReencryptSettings(ctx)
		},
		getSMTPSettingsFunc: func(ctx context.Context) (*SmtpSettingsResponse, error) {
			return SmtpSettings(ctx)
		},
		sendEmailFunc: func(ctx context.Context, notifier *notify.Notify, subject, body string) error {
			return notifier.Send(ctx, subject, body)
		},
		setFailedStateFunc: func(ctx context.Context, state state.Persistence, path string, desiredState notification) error {
			return state.Add(ctx, path, desiredState)
		},
		rmFailedStateFunc: func(ctx context.Context, state state.Persistence, path string) error {
			return state.Rm(ctx, path)

		},
	}
	return &notifier
}

type notification struct {
	Status         status
	SecretPath     string
	Secret         userSecret
	Email          string
	PublicPgpKey   string
	LastNotifiedAt time.Time
}

func (n *notification) newNotificationFromCurrent(state status, pgpKey, email string) *notification {
	newNotification := &notification{
		Status:         state,
		SecretPath:     n.SecretPath,
		Secret:         n.Secret,
		Email:          email,
		PublicPgpKey:   pgpKey,
		LastNotifiedAt: n.LastNotifiedAt,
	}

	return newNotification
}

type userSecret struct {
	// Fetched from Vault
	EncyptedPassword string
	ConsoleURL       string
	Account          string
	Username         string
}

// LogDiff specifies the logging for the account notifier
func (n *AccountNotifier) LogDiff(ri *reconcile.ResourceInventory) {
	for target := range ri.State {
		desired := ri.State[target].Desired.(notification)
		if desired.Status == skip {
			util.Log().Debugw("Skipping notification for", "account", desired.Secret.Account, "username", desired.Secret.Username)
		} else if desired.Status == reencrypt {
			util.Log().Infow("Reencrypting", "account", desired.Secret.Account, "username", desired.Secret.Username)
		} else if desired.Status == notifyExpired {
			util.Log().Infow("PGP Key expired, notifying", "account", desired.Secret.Account, "username", desired.Secret.Username)
		}
	}
}

// CurrentState lists the secrets from the vault import path and adds them to the resource inventory as current state
func (n *AccountNotifier) CurrentState(_ context.Context, ri *reconcile.ResourceInventory) error {
	s, err := n.vault.ListSecrets(n.vaultImportPath)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error while getting list of secrets from import path %s", n.vaultImportPath))
	}

	for _, secretKey := range s.Keys {
		secretPath := fmt.Sprintf("%s/%s", n.vaultImportPath, secretKey)

		secret, err := n.vault.ReadSecret(secretPath)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error while reading secret %s", secretPath))
		}
		ri.AddResourceState(secret.Data["user_name"].(string), &reconcile.ResourceState{
			Current: notification{
				Status:     reencrypt,
				SecretPath: secretPath,
				Secret: userSecret{
					Username:         secret.Data["user_name"].(string),
					ConsoleURL:       secret.Data["console_url"].(string),
					EncyptedPassword: secret.Data["encrypted_password"].(string),
					Account:          secret.Data["account"].(string),
				},
			},
		})
	}
	return nil
}

func (n *AccountNotifier) getEmailAddress(username string) string {
	return fmt.Sprintf("%s@%s", username, n.smtpauth.mailAddress)
}

// DesiredState lists the users from the QontractServer and adds them to the resource inventory as desired state
func (n *AccountNotifier) DesiredState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	users, err := n.getuserFunc(ctx)
	if err != nil {
		return errors.Wrap(err, "Error while getting users from graphql")
	}

	userMap := make(map[string]UsersUsers_v1User_v1)
	for _, user := range users.GetUsers_v1() {
		userMap[user.Org_username] = user
	}

	for target, state := range ri.State {
		user, ok := userMap[target]
		if !ok {
			util.Log().Errorf("User %s was delete, got stale password. Manual fix required", user.GetName())
		}
		exists, err := n.state.Exists(ctx, target)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error during state.Exists on target %s", target))
		}
		if exists {
			var lastNotification notification
			err = n.state.Get(ctx, target, &lastNotification)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error getting s3 state object %s", target))
			}
			if lastNotification.PublicPgpKey == user.GetPublic_gpg_key() {
				util.Log().Debug("Key is stale")
				if lastNotification.LastNotifiedAt.Add(24 * time.Hour).Before(time.Now()) {
					c := state.Current.(notification)
					state.Desired = *c.newNotificationFromCurrent(notifyExpired, user.GetPublic_gpg_key(), n.getEmailAddress(user.GetOrg_username()))
				} else {
					util.Log().Debug("Key is stale, but was notified recently")
					c := state.Current.(notification)
					state.Desired = *c.newNotificationFromCurrent(skip, user.GetPublic_gpg_key(), n.getEmailAddress(user.GetOrg_username()))
				}
			} else {
				c := state.Current.(notification)
				state.Desired = *c.newNotificationFromCurrent(reencrypt, user.GetPublic_gpg_key(), n.getEmailAddress(user.GetOrg_username()))
			}
		} else {
			c := state.Current.(notification)
			state.Desired = *c.newNotificationFromCurrent(reencrypt, user.GetPublic_gpg_key(), n.getEmailAddress(user.GetOrg_username()))
		}
	}

	return nil
}

func (n *AccountNotifier) newNotifier(receipt string) *notify.Notify {
	notifier := notify.New()
	email := mail.New(n.smtpauth.username, fmt.Sprintf("%s:%s", n.smtpauth.server, n.smtpauth.port))
	util.Log().Debugw("Sending email to", "address", receipt)
	email.AddReceivers(receipt)
	email.AuthenticateSMTP("", n.smtpauth.username, n.smtpauth.password, n.smtpauth.server)
	email.BodyFormat(mail.PlainText)
	notifier.UseServices(email)
	return notifier
}

func generateEmail(consoleURL, username, password string) string {
	return fmt.Sprintf(
		`
You have been invited to join an AWS account!
Below you will find credentials for the first sign in.
You will be requested to change your password.

The password is encrypted with your public PGP key. To decrypt the password:

echo <password> | base64 -d | gpg -d - && echo
(you will be asked to provide your passphrase to unlock the secret)

Once you are logged in, navigate to the "Security credentials" page [1] and enable MFA [2].
Once you have enabled MFA, sign out and sign in again.

Details:

Console URL: %s
Username: %s
Encrypted password: %s

[1] https://console.aws.amazon.com/iam/home#security_credential
[2] https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_mfa.html
`, consoleURL, username, password)
}

func generateEmailExpired(path string) string {
	return fmt.Sprintf(
		`
Your PGP key on the record has expired and is not valid anymore.
Changing passwords or requesting access to new AWS accounts will no longer work.
Please generate a new one following this guide [1]

Link to userfile: https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data%s

[1] https://gitlab.cee.redhat.com/service/app-interface/-/tree/master/#generating-a-gpg-key
`, path)
}

// Reconcile loop over state and reconcile depending on it.
func (n *AccountNotifier) Reconcile(ctx context.Context, ri *reconcile.ResourceInventory) error {
	for _, state := range ri.State {
		desired := state.Desired.(notification)
		if desired.Status == reencrypt {
			appsrekey, err := n.vault.ReadSecret(n.appSrePGPKeyPath)
			if err != nil {
				return errors.Wrap(err, "Error while reading secret from vault")
			}
			if appsrekey == nil {
				return fmt.Errorf("appsre PGP key not found in vault path: %s", n.appSrePGPKeyPath)
			}
			armoredOriginalPassword, err := pgp.DecodeAndArmorBase64Entity(desired.Secret.EncyptedPassword, constants.PGPMessageHeader)
			if err != nil {
				return errors.Wrap(err, "Error decoding and armoring encrypted password")
			}

			theActualPassword, err := phelper.DecryptMessageArmored(appsrekey.Data["private_key"].(string),
				[]byte(appsrekey.Data["passphrase"].(string)), armoredOriginalPassword)
			if err != nil {
				return errors.Wrap(err, "Error while decrypting encrypted password")
			}
			armoredUserPublicPgpKey, err := pgp.DecodeAndArmorBase64Entity(desired.PublicPgpKey, constants.PublicKeyHeader)
			if err != nil {
				errorWrapped := errors.Wrap(err, "Error while decoding and armoring User Public PGP Key, setting state entry")
				err = n.setFailedStateFunc(ctx, n.state, desired.Secret.Username, desired)
				if err != nil {
					return errors.Wrapf(errorWrapped, "Error while setting state entry for broken Public PGP Key")
				}
				return errorWrapped
			}
			armoredReencryptedPassword, err := phelper.EncryptMessageArmored(armoredUserPublicPgpKey, theActualPassword)
			if err != nil {
				errorWrapped := errors.Wrap(err, "Error while encrypting password with User Public PGP Key")
				err = n.setFailedStateFunc(ctx, n.state, desired.Secret.Username, desired)
				if err != nil {
					return errors.Wrapf(errorWrapped, "Error while setting state entry for broken Public PGP Key")
				}
				return errorWrapped
			}

			unarmoredReencryptedPassword, err := parmor.Unarmor(armoredReencryptedPassword)
			if err != nil {
				errors.Wrap(err, "Error while unarmoring encrypted password")
			}

			encodedReencryptedPassword := base64.StdEncoding.EncodeToString(unarmoredReencryptedPassword)

			// Disable linting for var-naming because we need to be consistent with qontract-reconcile
			//
			//revive:disable:var-naming
			type outputSecret struct {
				Console_url        string `json:"console_url"`
				Encrypted_password string `json:"encrypted_password"`
				Acount             string `json:"account"`
				User_name          string `json:"user_name"`
			}
			//revive:enable:var-naming

			output := outputSecret{
				Console_url:        desired.Secret.ConsoleURL,
				Encrypted_password: encodedReencryptedPassword,
				Acount:             desired.Secret.Account,
				User_name:          desired.Secret.Username,
			}

			err = n.state.Add(ctx, fmt.Sprintf("output/%s/%s", desired.Secret.Account, desired.Secret.Username), output)
			if err != nil {
				return errors.Wrap(err, "Error while writing encrypted password to s3")
			}

			_, err = n.vault.DeleteSecret(desired.SecretPath)
			if err != nil {
				return errors.Wrap(err, "Error while deleting initial password from vault")
			}

			exists, err := n.state.Exists(ctx, desired.Secret.Username)
			if err != nil {
				return errors.Wrap(err, "Error while checking state for stale PGP Key existence")
			}
			if exists {
				err = n.rmFailedStateFunc(ctx, n.state, desired.Secret.Username)
				if err != nil {
					return errors.Wrap(err, "Error while deleting statel PGP Key reference from state")
				}
			}

			err = n.sendEmailFunc(ctx, n.newNotifier(desired.Email), "AWS Access provisioned", generateEmail(desired.Secret.ConsoleURL, desired.Secret.Username, encodedReencryptedPassword))
			if err != nil {
				return errors.Wrap(err, "Error while sending user notification")
			}

		}
		if desired.Status == notifyExpired {
			util.Log().Info("Notification of expired keys to be done")
			err := n.sendEmailFunc(ctx, n.newNotifier(desired.Email), "Action required: Update PGP key", generateEmailExpired(desired.Secret.Username))
			if err != nil {
				return errors.Wrapf(err, "Error while sending user notification")
			}
			desired.LastNotifiedAt = time.Now()
			err = n.setFailedStateFunc(ctx, n.state, desired.Secret.Username, desired)
			if err != nil {
				return errors.Wrapf(err, "Error while setting state entry for broken Public PGP Key")
			}
		}
	}
	return nil
}

// Setup the account notifier
func (n *AccountNotifier) Setup(ctx context.Context) error {
	var err error

	n.vault, err = vault.NewVaultClient()
	if err != nil {
		return errors.Wrapf(err, "Error setting up vault client")
	}

	awsSecrets, err := aws.GetAwsCredentials(ctx, n.vault)
	if err != nil {
		return errors.Wrapf(err, "Error getting AWS secrets")
	}

	awsclient, err := aws.NewClient(ctx, awsSecrets)
	if err != nil {
		return errors.Wrapf(err, "Error getting AWS client")
	}

	n.state = state.NewS3State("state", IntegrationName, awsclient)

	settings, err := n.getReencryptFunc(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error getting reencrypt settings")
	}

	n.appSrePGPKeyPath = settings.GetPgp_reencrypt_settings_v1()[0].GetPrivate_pgp_key_vault_path()
	n.vaultImportPath = settings.GetPgp_reencrypt_settings_v1()[0].GetReencrypt_vault_path()
	n.vaultExportPath = settings.GetPgp_reencrypt_settings_v1()[0].GetAws_account_output_vault_path()

	allSMTPSettings, err := n.getSMTPSettingsFunc(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error gettoing smtp settings")
	}

	smtpSettings := allSMTPSettings.GetSettings()[0].GetSmtp()
	smtpSecret, err := n.vault.ReadSecret(smtpSettings.GetCredentials().Path)
	if err != nil {
		return errors.Wrapf(err, "Error while reading smtp credentials from vault")
	}

	n.smtpauth = smtpAuth{
		mailAddress: smtpSettings.GetMailAddress(),
		username:    smtpSecret.Data["username"].(string),
		password:    smtpSecret.Data["password"].(string),
		server:      smtpSecret.Data["server"].(string),
		port:        smtpSecret.Data["port"].(string),
	}

	return nil
}
