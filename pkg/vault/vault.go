package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
	"github.com/hashicorp/vault/api/auth/kubernetes"

	"github.com/spf13/viper"
)

// VaultClient is an abstraction to github.com/hashicorp/vault/api
type VaultClient struct {
	client *api.Client
	config *vaultConfig
}

type vaultConfig struct {
	Server        string
	AuthType      string
	Token         string
	Role_ID       string
	Secret_ID     string
	KubeRole      string
	KubeMount     string
	KubeTokenPath string
	Timeout       int
}

func newVaultConfig() *vaultConfig {
	var vc vaultConfig
	sub := util.EnsureViperSub(viper.GetViper(), "vault")
	sub.SetDefault("timeout", 60)
	sub.SetDefault("authtype", "approle")
	sub.SetDefault("kube_sa_token_path", "/var/run/secrets/kubernetes.io/serviceaccount/token")
	sub.BindEnv("server", "VAULT_SERVER")
	sub.BindEnv("authtype", "VAULT_AUTHTYPE")
	sub.BindEnv("token", "VAULT_TOKEN")
	sub.BindEnv("role_id", "VAULT_ROLE_ID")
	sub.BindEnv("secret_id", "VAULT_SECRET_ID")
	sub.BindEnv("kube_auth_role", "VAULT_KUBE_ROLE")
	sub.BindEnv("kube_auth_mount", "VAULT_KUBE_MOUNT")
	sub.BindEnv("timeout", "VAULT_TIMEOUT")
	if err := sub.Unmarshal(&vc); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &vc
}

// NewVaultClient creates a new VaultClient from a VaultConfig
func NewVaultClient() (*VaultClient, error) {
	vc := newVaultConfig()
	vaultClient := &VaultClient{
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

	switch vc.AuthType {
	case "approle":
		appRoleAuth, err := approle.NewAppRoleAuth(
			vc.Role_ID,
			&approle.SecretID{FromString: vc.Secret_ID})

		if err != nil {
			return nil, err
		}
		_, err = vaultClient.client.Auth().Login(context.Background(), appRoleAuth)
		if err != nil {
			return nil, err
		}

	case "token":
		vaultClient.client.SetToken(vc.Token)

	case "kubernetes":
		kubeAuth, err := kubernetes.NewKubernetesAuth(
			vc.KubeRole,
			kubernetes.WithServiceAccountTokenPath(vc.KubeTokenPath),
			kubernetes.WithMountPath(vc.KubeMount),
		)

		if err != nil {
			return nil, err
		}
		_, err = vaultClient.client.Auth().Login(context.Background(), kubeAuth)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported auth type \"%s\"", vc.AuthType)
	}

	return vaultClient, nil
}

// ReadSecret do a logical read on a given Secret Path
func (v *VaultClient) ReadSecret(secretPath string) (*api.Secret, error) {
	return v.client.Logical().Read(secretPath)
}

type SecretList struct {
	Keys []string
}

// ListSecrets list secrets on a given Secret Path
func (v *VaultClient) ListSecrets(secretPath string) (*SecretList, error) {
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
				return nil, fmt.Errorf("unexpected return type for secret %s, this is a bug", secretPath)
			}
		}
	}

	return &SecretList{
		Keys: keyList,
	}, nil
}

func (v *VaultClient) WriteSecret(secretPath string, secret map[string]interface{}) (*api.Secret, error) {
	return v.client.Logical().Write(secretPath, secret)
}

func (v *VaultClient) DeleteSecret(secretPath string) (*api.Secret, error) {
	return v.client.Logical().Delete(secretPath)
}
