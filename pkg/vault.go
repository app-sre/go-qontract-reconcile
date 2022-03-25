package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"

	"github.com/spf13/viper"
)

// VaultClient is an abstraction to github.com/hashicorp/vault/api
type VaultClient struct {
	client *api.Client
	config *vaultConfig
}

type vaultConfig struct {
	Addr     string
	AuthType string
	Token    string
	RoleID   string
	SecretID string
	Timeout  int
}

func newVaultConfig() *vaultConfig {
	var vc vaultConfig
	sub := EnsureViperSub(viper.GetViper(), "vault")
	sub.SetDefault("timeout", 60)
	sub.BindEnv("addr", "VAULT_ADDR")
	sub.BindEnv("authtype", "VAULT_AUTHTYPE")
	sub.BindEnv("token", "VAULT_TOKEN")
	sub.BindEnv("roleid", "VAULT_ROLE_ID")
	sub.BindEnv("secretid", "VAULT_SECRET_ID")
	sub.BindEnv("timeout", "VAULT_TIMEOUT")
	if err := sub.Unmarshal(&vc); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
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
	vaultCFG.Address = vc.Addr
	vaultCFG.Timeout = time.Duration(vc.Timeout) * time.Second

	tmpClient, err := api.NewClient(vaultCFG)
	if err != nil {
		return nil, err
	}
	vaultClient.client = tmpClient

	switch vc.AuthType {
	case "approle":
		appRoleAuth, err := approle.NewAppRoleAuth(
			vc.RoleID,
			&approle.SecretID{FromString: vc.SecretID})

		if err != nil {
			return nil, err
		}
		_, err = vaultClient.client.Auth().Login(context.Background(), appRoleAuth)
		if err != nil {
			return nil, err
		}

	case "token":
		vaultClient.client.SetToken(vc.Token)

	default:
		return nil, fmt.Errorf("unsupported auth type \"%s\"", vc.AuthType)
	}

	return vaultClient, nil
}

// ReadSecret do a logical read on a given Secret Path
func (v *VaultClient) ReadSecret(secretPath string) (*api.Secret, error) {
	return v.client.Logical().Read(secretPath)
}
