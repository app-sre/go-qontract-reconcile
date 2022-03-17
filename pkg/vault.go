package pkg

import (
	"fmt"

	"github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
)

// VaultClient is an abstraction to github.com/hashicorp/vault/api
type VaultClient struct {
	client *api.Client
}

// VaultConfig is used to unmarshal yaml configuration for VaultClients
type VaultConfig struct {
	Addr     string
	AuthType string
	Token    string
	RoleID   string
	SecretID string
}

// NewVaultConfig creates a new VaultConfig from viper configuration
func NewVaultConfig(v *viper.Viper) *VaultConfig {
	var vc VaultConfig
	sub := EnsureViperSub(v, "vault")
	sub.BindEnv("addr", "VAULT_ADDR")
	sub.BindEnv("authtype", "VAULT_AUTHTYPE")
	sub.BindEnv("token", "VAULT_TOKEN")
	sub.BindEnv("roleid", "VAULT_ROLE_ID")
	sub.BindEnv("secretid", "VAULT_SECRET_ID")
	if err := sub.Unmarshal(&vc); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &vc
}

// NewVaultClient creates a new VaultClient from a VaultConfig
func NewVaultClient(vc *VaultConfig) (*VaultClient, error) {
	vaultClient := &VaultClient{}
	vaultCFG := api.DefaultConfig()
	vaultCFG.Address = vc.Addr

	tmpClient, err := api.NewClient(vaultCFG)
	if err != nil {
		return nil, err
	}
	vaultClient.client = tmpClient

	switch vc.AuthType {
	case "approle":
		roleID := vc.RoleID
		secretID := vc.SecretID

		secret, err := vaultClient.client.Logical().Write("auth/approle/login", map[string]interface{}{
			"role_id":   roleID,
			"secret_id": secretID,
		})
		if err != nil {
			return nil, err
		}
		vaultClient.client.SetToken(secret.Auth.ClientToken)

	case "token":
		vaultClient.client.SetToken(vc.Token)

	default:
		return nil, fmt.Errorf("Unsupported auth type \"%s\"", vc.AuthType)
	}

	return vaultClient, nil
}

// ReadSecret do a logical read on a given Secret Path
func (v *VaultClient) ReadSecret(secretPath string) (*api.Secret, error) {
	return v.client.Logical().Read(secretPath)
}
