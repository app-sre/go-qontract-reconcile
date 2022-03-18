package pkg

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setupViperAll() *viper.Viper {
	v := viper.New()

	vaultCfg := make(map[string]interface{})
	vaultCfg["addr"] = "fooAddr"
	vaultCfg["token"] = "fooToken"
	vaultCfg["roleid"] = "fooRoleID"
	vaultCfg["authType"] = "fooAuthType"
	vaultCfg["secretid"] = "fooSecretID"

	v.Set("vault", vaultCfg)
	return v
}

func setupViperEnv() *viper.Viper {
	v := viper.New()

	vaultCfg := make(map[string]interface{})
	os.Setenv("VAULT_TOKEN", "fooToken")
	os.Setenv("VAULT_ROLE_ID", "fooRoleID")
	os.Setenv("VAULT_SECRET_ID", "fooSecretID")

	v.Set("vault", vaultCfg)
	return v
}

func TestNewVaultConfigAll(t *testing.T) {
	vc := NewVaultConfig(setupViperAll())

	assert.Equal(t, vc.Addr, "fooAddr")
	assert.Equal(t, vc.Token, "fooToken")
	assert.Equal(t, vc.RoleID, "fooRoleID")
	assert.Equal(t, vc.AuthType, "fooAuthType")
	assert.Equal(t, vc.SecretID, "fooSecretID")
}

func TestNewVaultConfigEnv(t *testing.T) {
	vc := NewVaultConfig(setupViperEnv())

	assert.Equal(t, vc.Token, "fooToken")
	assert.Equal(t, vc.RoleID, "fooRoleID")
	assert.Equal(t, vc.SecretID, "fooSecretID")
}

func TestNewVaultClientToken(t *testing.T) {
	vc := VaultConfig{
		Addr:     "http://foo.example",
		AuthType: "token",
		Token:    "xxx",
	}

	v, err := NewVaultClient(&vc)
	assert.Nil(t, err)

	assert.Equal(t, vc.Token, v.client.Token())
}

func TestNewVaultClientAppRole(t *testing.T) {
	vc := VaultConfig{
		AuthType: "approle",
		SecretID: "foo",
		RoleID:   "bar",
	}
	mockedToken := "65b74ffd-842c-fd43-1386-f7d7006e520a"
	vaultMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "auth/approle/login")
		sentBody, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf(`{"role_id":"%s","secret_id":"%s"}`, vc.RoleID, vc.SecretID), string(sentBody))

		fmt.Fprintf(w, `{"auth": {"client_token": "%s"}}`, mockedToken)

	}))
	defer vaultMock.Close()

	vc.Addr = vaultMock.URL

	v, err := NewVaultClient(&vc)
	assert.Nil(t, err)

	assert.Equal(t, mockedToken, v.client.Token())
}

func TestNewVaultClientUnsuportedAuthType(t *testing.T) {
	vc := VaultConfig{
		Addr:     "http://localhost",
		AuthType: "foo",
	}

	_, err := NewVaultClient(&vc)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Unsupported auth type \"foo\"")
}
