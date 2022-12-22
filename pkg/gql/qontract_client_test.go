package gql

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/app-sre/go-qontract-reconcile/pkg"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var testContext = context.WithValue(context.TODO(), pkg.ContextIngetrationNameKey, "user-validator")

func qontractSetupViper() {
	v := viper.GetViper()

	qontractCfg := make(map[string]interface{})
	qontractCfg["serverurl"] = "http://conf.example"

	v.Set("qontract", qontractCfg)
}

func TestNewQontractClient(t *testing.T) {
	qontractSetupViper()

	client, err := NewQontractClient(context.TODO())
	assert.Nil(t, err)
	assert.Equal(t, "http://conf.example", client.config.ServerURL)
	assert.NotNil(t, client)
}

func TestNewQontractClientEnv(t *testing.T) {
	qontractSetupViper()
	os.Setenv("QONTRACT_SERVER_URL", "http://env.example")

	client, err := NewQontractClient(context.TODO())
	assert.Nil(t, err)
	assert.Equal(t, "http://env.example", client.config.ServerURL)
	assert.NotNil(t, client)
}

func TestClientTimeout(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
		}))
	qontractSetupViper()
	os.Setenv("QONTRACT_SERVER_URL", mock.URL)
	os.Setenv("QONTRACT_TIMEOUT", "1")
	os.Setenv("QONTRACT_RETRIES", "0")

	client, err := NewQontractClient(testContext)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	err = client.MakeRequest(testContext, &graphql.Request{}, &graphql.Response{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "giving up after 1 attempt(s)")
}

func TestClientRetry(t *testing.T) {
	reqCount := 0
	mock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			reqCount++
			if reqCount == 1 {
				w.WriteHeader(500)
			} else {
				w.Write([]byte(`{"data":{}, "extensions": {"schemas":[]}}`))
			}
		}))
	qontractSetupViper()
	os.Setenv("QONTRACT_SERVER_URL", mock.URL)
	os.Setenv("QONTRACT_RETRIES", "1")

	client, err := NewQontractClient(testContext)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	err = client.MakeRequest(testContext, &graphql.Request{}, &graphql.Response{})
	assert.Nil(t, err)
	// first request fails, then schema + query
	assert.Equal(t, reqCount, 3)
}

func TestClientAuth(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "basic foobar", r.Header.Get("Authorization"))
			w.Write([]byte(`{"data":{}, "extensions": {"schemas":[]}}`))
		}))
	qontractSetupViper()
	os.Setenv("QONTRACT_SERVER_URL", mock.URL)
	os.Setenv("QONTRACT_TOKEN", "basic foobar")

	client, err := NewQontractClient(testContext)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	err = client.MakeRequest(testContext, &graphql.Request{}, &graphql.Response{})
	assert.Nil(t, err)
}

func TestBrokenExtensions(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data":{}, "extensions": {}}`))
		}))
	qontractSetupViper()
	os.Setenv("QONTRACT_SERVER_URL", mock.URL)

	client, err := NewQontractClient(testContext)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	err = client.MakeRequest(testContext, &graphql.Request{}, &graphql.Response{})
	assert.NotNil(t, err)
}

func TestIntegrationsCalled(t *testing.T) {
	var expected_queries = []string{
		`{"query":"","operationName":""}`,
		`{"query":"\nquery integrations {\n\tintegrations: integrations_v1 {\n\t\tname\n\t\tdescription\n\t\tschemas\n\t}\n}\n","operationName":"integrations"}`,
	}
	var extensionsQueried bool
	mock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			if pkg.Contains(expected_queries, string(b)) {
				extensionsQueried = true
			}
			w.Write([]byte(`{"data":{}, "extensions": {"schemas": []}}`))
		}))
	qontractSetupViper()
	os.Setenv("QONTRACT_SERVER_URL", mock.URL)

	client, err := NewQontractClient(testContext)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	err = client.MakeRequest(testContext, &graphql.Request{}, &graphql.Response{})
	assert.Nil(t, err)
	assert.True(t, extensionsQueried)
}

func TestSchemaMissing(t *testing.T) {
	client, _ := NewQontractClient(testContext)
	assert.NotNil(t, client)

	var schemas []interface{}
	schemas = append(schemas, "/dummy.json")
	err := client.ensureSchema("foo",
		&graphql.Response{Extensions: map[string]interface{}{"schemas": schemas}},
		&integrationsResponse{Integrations: []integrationsIntegrationsIntegration_v1{{
			Name:    "foo",
			Schemas: []string{"/other.json"},
		}}},
	)
	assert.NotNil(t, err)
	assert.ErrorContains(t, err, "usage of schema /dummy.json not allowed for integration foo")
}

func TestSchemaOkay(t *testing.T) {
	client, _ := NewQontractClient(testContext)
	assert.NotNil(t, client)

	var schemas []interface{}
	schemas = append(schemas, "/dummy.json")
	err := client.ensureSchema("foo",
		&graphql.Response{Extensions: map[string]interface{}{"schemas": schemas}},
		&integrationsResponse{Integrations: []integrationsIntegrationsIntegration_v1{{
			Name:    "foo",
			Schemas: []string{"/dummy.json"},
		}}},
	)
	assert.Nil(t, err)
}
