package vault

import (
	"fmt"
	"io"
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
	vaultCfg["server"] = "fooAddr"
	vaultCfg["token"] = "fooToken"
	vaultCfg["role_id"] = "fooRoleID"
	vaultCfg["authType"] = "fooAuthType"
	vaultCfg["secret_id"] = "fooSecretID"
	vaultCfg["kube_auth_role"] = "fooKubeRole"
	vaultCfg["kube_auth_mount"] = "fooKubeMount"
	vaultCfg["kube_sa_token_path"] = "fooKubeTokenPath"

	viper.GetViper().Set("vault", vaultCfg)
}

func setupViperEnv() {
	os.Setenv("VAULT_TOKEN", "fooToken")
	os.Setenv("VAULT_ROLE_ID", "fooRoleID")
	os.Setenv("VAULT_SECRET_ID", "fooSecretID")
	os.Setenv("VAULT_KUBE_AUTH_ROLE", "fooKubeRole")
	os.Setenv("VAULT_KUBE_AUTH_MOUNT", "fooKubeMount")
	os.Setenv("VAULT_KUBE_SA_TOKEN_PATH", "fooKubeTokenPath")

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

func setupViperKube() string {
	os.Setenv("VAULT_AUTHTYPE", "kubernetes")
	os.Setenv("VAULT_KUBE_AUTH_ROLE", "foo")
	os.Setenv("VAULT_KUBE_AUTH_MOUNT", "kubernetes")

	path := "./k8s-test-token"
	os.Setenv("VAULT_KUBE_SA_TOKEN_PATH", path)
	os.WriteFile(path, []byte("base64jwt"), 0644)

	vaultCfg := make(map[string]interface{})
	viper.GetViper().Set("vault", vaultCfg)

	return path
}

func TestNewVaultConfigAll(t *testing.T) {
	setupViperAll()
	vc := newVaultConfig()

	assert.Equal(t, vc.Server, "fooAddr")
	assert.Equal(t, vc.Token, "fooToken")
	assert.Equal(t, vc.Role_ID, "fooRoleID")
	assert.Equal(t, vc.AuthType, "fooAuthType")
	assert.Equal(t, vc.Secret_ID, "fooSecretID")
	assert.Equal(t, vc.Kube_Auth_Role, "fooKubeRole")
	assert.Equal(t, vc.Kube_Auth_Mount, "fooKubeMount")
	assert.Equal(t, vc.Kube_SA_Token_Path, "fooKubeTokenPath")
}

func TestNewVaultConfigEnv(t *testing.T) {
	setupViperEnv()
	vc := newVaultConfig()

	assert.Equal(t, vc.Token, "fooToken")
	assert.Equal(t, vc.Role_ID, "fooRoleID")
	assert.Equal(t, vc.Secret_ID, "fooSecretID")
	assert.Equal(t, vc.Kube_Auth_Role, "fooKubeRole")
	assert.Equal(t, vc.Kube_Auth_Mount, "fooKubeMount")
	assert.Equal(t, vc.Kube_SA_Token_Path, "fooKubeTokenPath")
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
		sentBody, err := io.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Equal(t, `{"role_id":"bar","secret_id":"foo"}`, string(sentBody))

		fmt.Fprintf(w, `{"auth": {"client_token": "%s"}}`, mockedToken)

	}))
	defer vaultMock.Close()

	os.Setenv("VAULT_SERVER", vaultMock.URL)

	v, err := NewVaultClient()
	assert.Nil(t, err)

	assert.Equal(t, mockedToken, v.client.Token())
}

func TestNewVaultClientKube(t *testing.T) {
	path := setupViperKube()
	defer os.Remove(path)

	mockedToken := "65b74ffd-842c-fd43-1386-f7d7006e520a"
	vaultMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "auth/kubernetes/login")
		sentBody, err := io.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Equal(t, `{"jwt":"base64jwt","role":"foo"}`, string(sentBody))

		fmt.Fprintf(w, `{"auth": {"client_token": "%s"}}`, mockedToken)
	}))
	defer vaultMock.Close()

	os.Setenv("VAULT_SERVER", vaultMock.URL)

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
	os.Setenv("VAULT_SERVER", vaultMock.URL)
	os.Setenv("VAULT_TIMEOUT", "1")

	client, err := NewVaultClient()
	assert.NotNil(t, client)
	assert.Nil(t, err)
	secret, err := client.ReadSecret("foo")
	assert.NotNil(t, err)
	assert.Nil(t, secret)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
