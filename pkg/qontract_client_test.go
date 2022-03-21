package pkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func qontractSetupViper() {
	v := viper.GetViper()

	qontractCfg := make(map[string]interface{})
	qontractCfg["serverurl"] = "http://conf.example"

	v.Set("qontract", qontractCfg)
}

func TestNewQontractClient(t *testing.T) {
	qontractSetupViper()

	client := NewQontractClient()
	assert.Equal(t, "http://conf.example", client.config.ServerURL)
	assert.NotNil(t, client)
}

func TestNewQontractClientEnv(t *testing.T) {
	qontractSetupViper()
	os.Setenv("QONTRACT_SERVER_URL", "http://env.example")

	client := NewQontractClient()
	assert.Equal(t, "http://env.example", client.config.ServerURL)
	assert.NotNil(t, client)
}

func TestNewQontractClientTimeout(t *testing.T) {
	var in, out interface{}
	timeoutMock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
		}))
	qontractSetupViper()
	os.Setenv("QONTRACT_SERVER_URL", timeoutMock.URL)
	os.Setenv("QONTRACT_TIMEOUT", "1")

	client := NewQontractClient()
	assert.NotNil(t, client)
	err := client.Client.MakeRequest(context.Background(), "query", "foo", &in, &out)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Client.Timeout exceeded while awaiting headers")
}
