// Package vault adds a vault client implementation
package vault

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
	"github.com/hashicorp/vault/api/auth/kubernetes"

	"github.com/spf13/viper"
)

const (
	// How long before client requests to Vault are timed out.
	defaultClientTimeout = 60 // Seconds.

	// How long before a client login attempt to Vault is timed out.
	defaultClientLoginTimeout = 5 * time.Second

	// How many times attempt to retry when failing
	// to retrieve a valid client token.
	defaultTokenRetryAttempts = 5

	// How long to sleep in between each retry attempt.
	defaultTokenRetrySleep = 250 * time.Millisecond
)

// Client is an abstraction to github.com/hashicorp/vault/api
type Client struct {
	client *api.Client
	config *vaultConfig
}

// Disable lint, cause names should match Qontract Reconcile
//
//revive:disable:var-naming
type vaultConfig struct {
	Server             string
	AuthType           string
	Token              string
	Role_ID            string
	Secret_ID          string
	Kube_Auth_Role     string
	Kube_Auth_Mount    string
	Kube_SA_Token_Path string
	Timeout            int
}

//revive:enable:var-naming

func newVaultConfig() *vaultConfig {
	var vc vaultConfig
	sub := util.EnsureViperSub(viper.GetViper(), "vault")
	sub.SetDefault("timeout", defaultClientTimeout)
	sub.SetDefault("authtype", "approle")
	sub.SetDefault("kube_sa_token_path", "/var/run/secrets/kubernetes.io/serviceaccount/token")
	sub.BindEnv("server", "VAULT_SERVER")
	sub.BindEnv("authtype", "VAULT_AUTHTYPE")
	sub.BindEnv("token", "VAULT_TOKEN")
	sub.BindEnv("role_id", "VAULT_ROLE_ID")
	sub.BindEnv("secret_id", "VAULT_SECRET_ID")
	sub.BindEnv("kube_auth_role", "VAULT_KUBE_AUTH_ROLE")
	sub.BindEnv("kube_auth_mount", "VAULT_KUBE_AUTH_MOUNT")
	sub.BindEnv("kube_sa_token_path", "VAULT_KUBE_SA_TOKEN_PATH")
	sub.BindEnv("timeout", "VAULT_TIMEOUT")
	if err := sub.Unmarshal(&vc); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration: %s", err.Error())
	}
	return &vc
}

// NewVaultClient creates a new VaultClient from a VaultConfig
func NewVaultClient() (*Client, error) {
	vc := newVaultConfig()
	vaultClient := &Client{
		config: vc,
	}
	vaultCFG := api.DefaultConfig()
	vaultCFG.Address = vc.Server
	vaultCFG.Timeout = time.Duration(vc.Timeout) * time.Second

	tmpClient, err := api.NewClient(vaultCFG)
	if err != nil {
		return nil, err
	}
	vaultClient.client = tmpClient

	// This timeout will override the default one set for all client requests.
	ctxTimeout, cancel := context.WithTimeout(context.Background(), defaultClientLoginTimeout)
	defer cancel()

	switch vc.AuthType {
	case "approle":
		if err := approleAuthLogin(ctxTimeout, vaultClient); err != nil {
			return nil, fmt.Errorf("unable to login with AppRole credentials: %w", err)
		}
	case "kubernetes":
		if err := kubernetesAuthLogin(ctxTimeout, vaultClient); err != nil {
			return nil, fmt.Errorf("unable to login with Kubernetes credentials: %w", err)
		}
	case "token":
		vaultClient.client.SetToken(vc.Token)
	default:
		return nil, fmt.Errorf("unsupported authentication type %q", vc.AuthType)
	}

	return vaultClient, nil
}

// ReadSecret do a logical read on a given Secret Path
func (v *Client) ReadSecret(secretPath string) (*api.Secret, error) {
	parts := strings.SplitN(secretPath, "/", 2)
	kv2Path := fmt.Sprintf("%s/data/%s", parts[0], parts[1])
	return v.client.Logical().Read(kv2Path)
}

// SecretList is a list of secrets
type SecretList struct {
	Keys []string
}

// ListSecrets list secrets on a given Secret Path
func (v *Client) ListSecrets(secretPath string) (*SecretList, error) {
	secret, err := v.client.Logical().List(secretPath)
	if err != nil {
		return nil, err
	}

	keyList := make([]string, 0)
	if secret != nil {
		for _, key := range secret.Data["keys"].([]interface{}) {
			switch key := key.(type) {
			case string:
				keyList = append(keyList, key)
			default:
				return nil, fmt.Errorf("unexpected type for secret %q: %T", secretPath, key)
			}
		}
	}

	return &SecretList{
		Keys: keyList,
	}, nil
}

// WriteSecret do a logical write on a given Secret Path
func (v *Client) WriteSecret(secretPath string, secret map[string]interface{}) (*api.Secret, error) {
	return v.client.Logical().Write(secretPath, secret)
}

// DeleteSecret do a logical delete on a given Secret Path
func (v *Client) DeleteSecret(secretPath string) (*api.Secret, error) {
	return v.client.Logical().Delete(secretPath)
}

func approleAuthLogin(ctx context.Context, client *Client) error {
	auth, err := approle.NewAppRoleAuth(
		client.config.Role_ID,
		&approle.SecretID{FromString: client.config.Secret_ID},
	)
	if err != nil {
		return err
	}

	return login(ctx, client.client, auth)
}

func kubernetesAuthLogin(ctx context.Context, client *Client) error {
	auth, err := kubernetes.NewKubernetesAuth(
		client.config.Kube_Auth_Role,
		kubernetes.WithServiceAccountTokenPath(client.config.Kube_SA_Token_Path),
		kubernetes.WithMountPath(client.config.Kube_Auth_Mount),
	)
	if err != nil {
		return err
	}

	return login(ctx, client.client, auth)
}

func login(ctx context.Context, client *api.Client, auth api.AuthMethod) error {
	err := util.Retry(defaultTokenRetryAttempts, defaultTokenRetrySleep, func() error {
		_, err := client.Auth().Login(ctx, auth)
		if err != nil {
			const clientTokenError = `client token not set`
			// The high-level client API also issues a write to the AppRole
			// mount endpoint to "login" to obtain a new token. The request
			// might return an empty response without necessarily failing.
			// As such, the high-level API checks for the presence of the
			// client token and returns an error if there is none. We then
			// attempt to retry the login attempt.
			if strings.Contains(err.Error(), clientTokenError) {
				util.Log().Warn("Received empty authentication information. Retrying...")
				return err
			}
			return util.RetryStop(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
