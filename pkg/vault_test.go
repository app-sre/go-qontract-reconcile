package pkg

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setupViperAll() {
	vaultCfg := make(map[string]interface{})
	vaultCfg["addr"] = "fooAddr"
	vaultCfg["token"] = "fooToken"
	vaultCfg["roleid"] = "fooRoleID"
	vaultCfg["authType"] = "fooAuthType"
	vaultCfg["secretid"] = "fooSecretID"

	viper.GetViper().Set("vault", vaultCfg)
}

func setupViperEnv() {
	os.Setenv("VAULT_TOKEN", "fooToken")
	os.Setenv("VAULT_ROLE_ID", "fooRoleID")
	os.Setenv("VAULT_SECRET_ID", "fooSecretID")

	vaultCfg := make(map[string]interface{})
	viper.GetViper().Set("vault", vaultCfg)
}

func setupViperToken() {
	os.Setenv("VAULT_TOKEN", "token")
	os.Setenv("VAULT_ADDR", "http://foo.example")
	os.Setenv("VAULT_AUTHTYPE", "token")

	vaultCfg := make(map[string]interface{})
	viper.GetViper().Set("vault", vaultCfg)
}

func setupViperAppRole() {
	os.Setenv("VAULT_ROLE_ID", "bar")
	os.Setenv("VAULT_SECRET_ID", "foo")
	os.Setenv("VAULT_AUTHTYPE", "approle")

	vaultCfg := make(map[string]interface{})
	viper.GetViper().Set("vault", vaultCfg)
}

func TestNewVaultConfigAll(t *testing.T) {
	setupViperAll()
	vc := newVaultConfig()

	assert.Equal(t, vc.Addr, "fooAddr")
	assert.Equal(t, vc.Token, "fooToken")
	assert.Equal(t, vc.RoleID, "fooRoleID")
	assert.Equal(t, vc.AuthType, "fooAuthType")
	assert.Equal(t, vc.SecretID, "fooSecretID")
}

func TestNewVaultConfigEnv(t *testing.T) {
	setupViperEnv()
	vc := newVaultConfig()

	assert.Equal(t, vc.Token, "fooToken")
	assert.Equal(t, vc.RoleID, "fooRoleID")
	assert.Equal(t, vc.SecretID, "fooSecretID")
}

func TestNewVaultClientToken(t *testing.T) {
	setupViperToken()
	v, err := NewVaultClient()

	assert.Nil(t, err)
	assert.Equal(t, "token", v.client.Token())
}

func TestNewVaultClientAppRole(t *testing.T) {
	setupViperAppRole()
	mockedToken := "65b74ffd-842c-fd43-1386-f7d7006e520a"
	vaultMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "auth/approle/login")
		sentBody, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Equal(t, `{"role_id":"bar","secret_id":"foo"}`, string(sentBody))

		fmt.Fprintf(w, `{"auth": {"client_token": "%s"}}`, mockedToken)

	}))
	defer vaultMock.Close()

	os.Setenv("VAULT_ADDR", vaultMock.URL)

	v, err := NewVaultClient()
	assert.Nil(t, err)

	assert.Equal(t, mockedToken, v.client.Token())
}

func TestNewVaultClientUnsuportedAuthType(t *testing.T) {
	os.Setenv("VAULT_AUTHTYPE", "jkjisdf")

	_, err := NewVaultClient()
	assert.NotNil(t, err)
	assert.EqualError(t, err, "unsupported auth type \"jkjisdf\"")
}

func TestVaultClientTimeout(t *testing.T) {
	vaultMock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
		}))
	setupViperToken()
	os.Setenv("VAULT_ADDR", vaultMock.URL)
	os.Setenv("VAULT_TIMEOUT", "1")

	client, err := NewVaultClient()
	assert.NotNil(t, client)
	assert.Nil(t, err)
	secret, err := client.ReadSecret("foo")
	assert.NotNil(t, err)
	assert.Nil(t, secret)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
