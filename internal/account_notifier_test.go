package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/app-sre/go-qontract-reconcile/internal/queries"
	"github.com/app-sre/go-qontract-reconcile/pkg"
	"github.com/app-sre/go-qontract-reconcile/pkg/mock"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/mock/gomock"
	"github.com/nikoksr/notify"
	"github.com/stretchr/testify/assert"
)

var (
	privateKey = "../test/data/notifier_private_key.b64"
	publicKey  = "../test/data/notifier_public_key.b64"
	testData   = "../test/data/notifier_test_data.b64"
)

func NewHttpTestServer(handlerFunc func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handlerFunc))
}

func SetupVaultEnv(url string) {
	os.Setenv("VAULT_TOKEN", "token")
	os.Setenv("VAULT_ADDR", "http://foo.example")
	os.Setenv("VAULT_AUTHTYPE", "token")
	os.Setenv("VAULT_ADDR", url)
}

func SetupGqlEnv(url string) {
	os.Setenv("QONTRACT_SERVER_URL", url)
}

func TestANCurrentState(t *testing.T) {
	vaultMock := NewHttpTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/v1?list=true" {
			fmt.Fprintf(w, `{"data": {"keys": ["%s"]}}`, "pgpKey")
		}
		if r.URL.String() == "/v1/pgpKey" {
			fmt.Fprintf(w, `{"data": {"user_name":"foobar","console_url": "http://a", "encrypted_password": "a", "account": "foobar" }}`)
		}
	})
	defer vaultMock.Close()
	SetupVaultEnv(vaultMock.URL)

	v, err := pkg.NewVaultClient()

	assert.NoError(t, err)

	a := AccountNotifier{
		vault: v,
	}

	ri := pkg.NewResourceInventory()
	a.CurrentState(context.TODO(), ri)

	assert.NotNil(t, ri.State["foobar"])

	cs := ri.State["foobar"].Current.(notification)

	assert.Equal(t, "foobar", cs.Secret.Username)
	assert.Equal(t, "http://a", cs.Secret.ConsoleURL)
	assert.Equal(t, "a", cs.Secret.EncyptedPassword)
}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	s := string(b)
	return s[1 : len(s)-1]
}

func setupVaultMock(t *testing.T) *httptest.Server {
	return NewHttpTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/v1/import?list=true" {
			fmt.Fprintf(w, `{"data": {"keys": ["%s"]}}`, "pgpKey")
		} else if r.URL.String() == "/v1/import/pgpKey" {
			fmt.Fprintf(w, `{"data": {"user_name":"foobar","console_url": "http://a", "encrypted_password": "%s", "account": "foobar" }}`,
				jsonEscape(string(pkg.ReadKeyFile(t, testData))))
		}
		if r.URL.String() == "/v1/privatekey" {
			fmt.Fprintf(w, `{"data": {"private_key": "%s", "passphrase": "abc123" }}`, jsonEscape(string(pkg.ReadKeyFile(t, privateKey))))
		}
	})
}

func createUserMock(pgpKey string) *queries.UsersResponse {
	return &queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Path:               "testing.yml",
			Name:               "foobar",
			Org_username:       "foobar",
			Github_username:    "foobar",
			Slack_username:     "foobar",
			Pagerduty_username: "foobar",
			Public_gpg_key:     pgpKey,
		},
		},
	}
}

func createTestNotifier(t *testing.T, vaultMock *pkg.VaultClient, awsClientMock *mock.MockClient, users *queries.UsersResponse) AccountNotifier {
	return AccountNotifier{
		vault: vaultMock,
		state: pkg.NewS3State("state", "test", awsClientMock),
		getuserFunc: func(ctx context.Context) (*queries.UsersResponse, error) {
			return users, nil
		},
		appSrePGPKeyPath: "privatekey",
		vaultImportPath:  "/import",
	}
}

/*

This table should help understanding the need for the various tests. There
are a couple of states the code could be in.

Status        Vault   State   PGP     Notification    Test name
------------- ------- ------- ------- --------------- --------------------
Reencrypt     Import  None    Valid   Yes             TestReencryptOkay
              Export
Reencrypt     Import  Update  Invalid No              TestReencryptInvalid
Reencrypt     Import  Delete  Updated Yes             TestReencryptUpdated
              Export
NotifyExpired Import  Updated Invalid Yes             TestNotifyExpired
Skip          Import  Read    Invalid No              TestSKip

*/

func TestReencryptOkay(t *testing.T) {
	users := createUserMock(string(pkg.ReadKeyFile(t, publicKey)))

	vaultMock := setupVaultMock(t)
	defer vaultMock.Close()
	SetupVaultEnv(vaultMock.URL)

	v, err := pkg.NewVaultClient()

	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.WithValue(context.Background(), pkg.ContextIngetrationNameKey, ACCOUNT_NOTIFIER_NAME)

	mockClient := mock.NewMockClient(ctrl)

	mockClient.EXPECT().HeadObject(ctx, gomock.Any()).Return(nil, fmt.Errorf("api error NotFound: Not Found")).MaxTimes(2)
	a := createTestNotifier(t, v, mockClient, users)
	mailSent := false
	a.sendEmailFunc = func(ctx context.Context, n *notify.Notify, body string) error {
		assert.Contains(t, body, "You have been invited to join an AWS account")
		assert.NotNil(t, n)
		mailSent = true
		return nil
	}
	ri := pkg.NewResourceInventory()

	err = a.CurrentState(ctx, ri)
	assert.NoError(t, err)

	err = a.DesiredState(ctx, ri)
	assert.NoError(t, err)

	err = a.Reconcile(ctx, ri)
	assert.NoError(t, err)
	assert.True(t, mailSent)
}

func TestReencryptInvalid(t *testing.T) {
	users := createUserMock("Invalid key")

	vaultMock := setupVaultMock(t)
	defer vaultMock.Close()
	SetupVaultEnv(vaultMock.URL)

	v, err := pkg.NewVaultClient()

	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.WithValue(context.Background(), pkg.ContextIngetrationNameKey, ACCOUNT_NOTIFIER_NAME)

	mockClient := mock.NewMockClient(ctrl)

	mockClient.EXPECT().HeadObject(ctx, gomock.Any()).Return(nil, fmt.Errorf("api error NotFound: Not Found")).MaxTimes(2)

	a := createTestNotifier(t, v, mockClient, users)
	a.setFailedStateFunc = func(ctx context.Context, p pkg.Persistence, s string, n notification) error {
		assert.Equal(t, "foobar", s)
		assert.Equal(t, "Invalid key", n.PublicPgpKey)
		return nil
	}
	ri := pkg.NewResourceInventory()

	err = a.CurrentState(ctx, ri)
	assert.NoError(t, err)

	err = a.DesiredState(ctx, ri)
	assert.NoError(t, err)

	err = a.Reconcile(ctx, ri)
	assert.ErrorContains(t, err, "Error while decoding and armoring User Public PGP Key, setting state entry")
}

func TestReencryptUpdated(t *testing.T) {
	users := createUserMock(string(pkg.ReadKeyFile(t, publicKey)))

	vaultMock := setupVaultMock(t)
	defer vaultMock.Close()
	SetupVaultEnv(vaultMock.URL)

	v, err := pkg.NewVaultClient()

	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.WithValue(context.Background(), pkg.ContextIngetrationNameKey, ACCOUNT_NOTIFIER_NAME)

	mockClient := mock.NewMockClient(ctrl)

	mockClient.EXPECT().HeadObject(ctx, gomock.Any()).Return(nil, nil).MaxTimes(2)
	mockClient.EXPECT().GetObject(ctx, gomock.Any()).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader([]byte(`
publicpgpkey: oldone
`))),
	}, nil)

	a := createTestNotifier(t, v, mockClient, users)
	stateRemoved := false
	mailSent := false
	a.sendEmailFunc = func(ctx context.Context, n *notify.Notify, body string) error {
		assert.Contains(t, body, "You have been invited to join an AWS account")
		assert.NotNil(t, n)
		mailSent = true
		return nil
	}
	a.rmFailedStateFunc = func(ctx context.Context, p pkg.Persistence, s string) error {
		assert.Equal(t, "foobar", s)
		stateRemoved = true
		return nil
	}

	ri := pkg.NewResourceInventory()

	err = a.CurrentState(ctx, ri)
	assert.NoError(t, err)

	err = a.DesiredState(ctx, ri)
	assert.NoError(t, err)

	err = a.Reconcile(ctx, ri)
	assert.NoError(t, err)

	assert.True(t, stateRemoved)
	assert.True(t, mailSent)
}

func TestNotifySkip(t *testing.T) {
	users := createUserMock("Invalid key")

	vaultMock := setupVaultMock(t)
	defer vaultMock.Close()
	SetupVaultEnv(vaultMock.URL)

	v, err := pkg.NewVaultClient()

	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.WithValue(context.Background(), pkg.ContextIngetrationNameKey, ACCOUNT_NOTIFIER_NAME)

	mockClient := mock.NewMockClient(ctrl)

	dateByte, err := time.Now().MarshalJSON()
	assert.NoError(t, err)

	mockClient.EXPECT().HeadObject(ctx, gomock.Any()).Return(nil, nil).MaxTimes(2)
	mockClient.EXPECT().GetObject(ctx, gomock.Any()).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`
publicpgpkey: Invalid key
lastnotifiedat: %s
`, string(dateByte))))),
	}, nil)

	a := createTestNotifier(t, v, mockClient, users)
	ri := pkg.NewResourceInventory()

	err = a.CurrentState(ctx, ri)
	assert.NoError(t, err)

	err = a.DesiredState(ctx, ri)
	assert.NoError(t, err)

	desiredState := ri.GetResourceState("foobar").Desired.(notification)
	assert.Equal(t, SKIP, desiredState.Status)
}

func TestNotifyExpired(t *testing.T) {
	users := createUserMock("Invalid key")

	vaultMock := setupVaultMock(t)
	defer vaultMock.Close()
	SetupVaultEnv(vaultMock.URL)

	v, err := pkg.NewVaultClient()

	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.WithValue(context.Background(), pkg.ContextIngetrationNameKey, ACCOUNT_NOTIFIER_NAME)

	mockClient := mock.NewMockClient(ctrl)

	dateByte, err := time.Date(2020, 1, 1, 1, 1, 1, 1, time.Local).MarshalJSON()
	assert.NoError(t, err)

	mockClient.EXPECT().HeadObject(ctx, gomock.Any()).Return(nil, nil).MaxTimes(2)
	mockClient.EXPECT().GetObject(ctx, gomock.Any()).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`
publicpgpkey: Invalid key
lastnotifiedat: %s
`, string(dateByte))))),
	}, nil)

	a := createTestNotifier(t, v, mockClient, users)
	mailSent := false
	statePersisted := false
	a.sendEmailFunc = func(ctx context.Context, n *notify.Notify, body string) error {
		assert.Contains(t, body, "Your PGP key on the record has expired and is not valid anymore.")
		assert.NotNil(t, n)
		mailSent = true
		return nil
	}
	a.setFailedStateFunc = func(ctx context.Context, p pkg.Persistence, s string, n notification) error {
		assert.Equal(t, "foobar", s)
		assert.Equal(t, "Invalid key", n.PublicPgpKey)
		statePersisted = true
		return nil
	}
	ri := pkg.NewResourceInventory()

	err = a.CurrentState(ctx, ri)
	assert.NoError(t, err)

	err = a.DesiredState(ctx, ri)
	assert.NoError(t, err)

	desiredState := ri.GetResourceState("foobar").Desired.(notification)
	assert.Equal(t, NOTIFY_EXPIRED, desiredState.Status)

	err = a.Reconcile(ctx, ri)
	assert.NoError(t, err)

	assert.True(t, mailSent)
	assert.True(t, statePersisted)
}
