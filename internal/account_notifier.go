package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/app-sre/go-qontract-reconcile/internal/queries"
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

var ACCOUNT_NOTIFIER_NAME = "account-notifier"

type status int

const (
	REENCRYPT status = iota
	SKIP
	NOTIFY_EXPIRED
)

type GetUsers func(context.Context) (*queries.UsersResponse, error)
type GetPgpReencryptSettings func(context.Context) (*queries.PgpReencryptSettingsResponse, error)
type GetSmtpSettings func(context.Context) (*queries.SmtpSettingsResponse, error)
type SendEmail func(context.Context, *notify.Notify, string) error
type SetFailedState func(context.Context, state.Persistence, string, notification) error
type RmFailedState func(context.Context, state.Persistence, string) error

type AccountNotifier struct {
	state            state.Persistence
	vault            *vault.VaultClient
	appSrePGPKeyPath string
	vaultImportPath  string
	vaultExportPath  string

	smtpauth            smtpAuth
	getuserFunc         GetUsers
	getReencryptFunc    GetPgpReencryptSettings
	getSmtpSettingsFunc GetSmtpSettings
	sendEmailFunc       SendEmail
	setFailedStateFunc  SetFailedState
	rmFailedStateFunc   RmFailedState
}

type smtpAuth struct {
	mailAddress string
	username    string
	password    string
	server      string
	port        string
}

func NewAccountNotifier() *AccountNotifier {
	notifier := AccountNotifier{
		getuserFunc: func(ctx context.Context) (*queries.UsersResponse, error) {
			return queries.Users(ctx)
		},
		getReencryptFunc: func(ctx context.Context) (*queries.PgpReencryptSettingsResponse, error) {
			return queries.PgpReencryptSettings(ctx)
		},
		getSmtpSettingsFunc: func(ctx context.Context) (*queries.SmtpSettingsResponse, error) {
			return queries.SmtpSettings(ctx)
		},
		sendEmailFunc: func(ctx context.Context, notifier *notify.Notify, body string) error {
			return notifier.Send(ctx, "AWS Access provisioned", body)
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

type Secret struct {
	EncyptedPassword string
	ConsoleURL       string
	Username         string
}

func (n *AccountNotifier) LogDiff(ri *reconcile.ResourceInventory) {
	for target := range ri.State {
		current := ri.State[target].Current.(notification)
		if current.Status == SKIP {
			util.Log().Debugw("Skipping notification for", "account", current.Secret.Account, "username", current.Secret.Username)
		} else if current.Status == REENCRYPT {
			util.Log().Debugw("Reencrypting", "account", current.Secret.Account, "username", current.Secret.Username)
		} else if current.Status == NOTIFY_EXPIRED {
			util.Log().Debugw("PGP Key expired, notifying", "account", current.Secret.Account, "username", current.Secret.Username)
		}
	}
}

func (n *AccountNotifier) CurrentState(ctx context.Context, ri *reconcile.ResourceInventory) error {
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
				Status:     REENCRYPT,
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

func (n *AccountNotifier) DesiredState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	users, err := n.getuserFunc(ctx)
	if err != nil {
		return errors.Wrap(err, "Error while getting users from graphql")
	}

	userMap := make(map[string]queries.UsersUsers_v1User_v1)
	for _, user := range users.GetUsers_v1() {
		userMap[user.Org_username] = user
	}

	for target, state := range ri.State {
		user, ok := userMap[target]
		if !ok {
			util.Log().Errorf("User %s was delete, got stale password. Manual fix required", user.GetName())
		}
		err, exists := n.state.Exists(ctx, target)
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
					state.Desired = *c.newNotificationFromCurrent(NOTIFY_EXPIRED, user.GetPublic_gpg_key(), n.getEmailAddress(user.GetOrg_username()))
				} else {
					util.Log().Debug("Key is stale, but was notified recently")
					c := state.Current.(notification)
					state.Desired = *c.newNotificationFromCurrent(SKIP, user.GetPublic_gpg_key(), n.getEmailAddress(user.GetOrg_username()))
				}
			} else {
				c := state.Current.(notification)
				state.Desired = *c.newNotificationFromCurrent(REENCRYPT, user.GetPublic_gpg_key(), n.getEmailAddress(user.GetOrg_username()))
			}
		} else {
			c := state.Current.(notification)
			state.Desired = *c.newNotificationFromCurrent(REENCRYPT, user.GetPublic_gpg_key(), n.getEmailAddress(user.GetOrg_username()))
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

func generateEmail(consoleUrl, username, password string) string {
	return fmt.Sprintf(
		`
You have been invited to join an AWS account!\n
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
`, consoleUrl, username, password)
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

// Sent notifications and add them to the state
func (n *AccountNotifier) Reconcile(ctx context.Context, ri *reconcile.ResourceInventory) error {
	for _, state := range ri.State {
		desired := state.Desired.(notification)
		if desired.Status == REENCRYPT {
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
			secretMap := make(map[string]interface{})
			secretMap["console_url"] = desired.Secret.ConsoleURL
			secretMap["encrypted_password"] = encodedReencryptedPassword
			secretMap["account"] = desired.Secret.Account
			secretMap["user_name"] = desired.Secret.Username

			_, err = n.vault.WriteSecret(fmt.Sprintf("%s/%s_%s", n.vaultExportPath, desired.Secret.Account, desired.Secret.Username), secretMap)
			if err != nil {
				return errors.Wrap(err, "Error while writing encrypted password to vault")
			}

			_, err = n.vault.DeleteSecret(desired.SecretPath)
			if err != nil {
				return errors.Wrap(err, "Error while deleting initial password from vault")
			}

			err, exists := n.state.Exists(ctx, desired.Secret.Username)
			if err != nil {
				return errors.Wrap(err, "Error while checking state for stale PGP Key existence")
			}
			if exists {
				err = n.rmFailedStateFunc(ctx, n.state, desired.Secret.Username)
				if err != nil {
					return errors.Wrap(err, "Error while deleting statel PGP Key reference from state")
				}
			}

			err = n.sendEmailFunc(ctx, n.newNotifier(desired.Email), generateEmail(desired.Secret.ConsoleURL, desired.Secret.Username, encodedReencryptedPassword))
			if err != nil {
				return errors.Wrap(err, "Error while sending user notification")
			}

		}
		if desired.Status == NOTIFY_EXPIRED {
			util.Log().Info("Notification of expired keys to be done")
			err := n.sendEmailFunc(ctx, n.newNotifier(desired.Email), generateEmailExpired(desired.Secret.Username))
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

func (n *AccountNotifier) Setup(ctx context.Context) error {
	var err error

	n.vault, err = vault.NewVaultClient()
	if err != nil {
		return errors.Wrapf(err, "Error setting up vault client")
	}

	n.state = state.NewS3State(ctx, "state", ACCOUNT_NOTIFIER_NAME, *n.vault)

	settings, err := n.getReencryptFunc(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error getting reencrypt settings")
	}

	n.appSrePGPKeyPath = settings.GetPgp_reencrypt_settings_v1()[0].GetPrivate_pgp_key_vault_path()
	n.vaultImportPath = settings.GetPgp_reencrypt_settings_v1()[0].GetReencrypt_vault_path()
	n.vaultExportPath = settings.GetPgp_reencrypt_settings_v1()[0].GetAws_account_output_vault_path()

	allSmtpSettings, err := n.getSmtpSettingsFunc(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error gettoing smtp settings")
	}

	smtpSettings := allSmtpSettings.GetSettings()[0].GetSmtp()
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
