package pkg

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func unleashSetupViper() {
	v := viper.GetViper()

	unleashCfg := make(map[string]interface{})
	unleashCfg["apiurl"] = "http://conf.example"

	v.Set("unleash", unleashCfg)
}

func TestNewUnleashClient(t *testing.T) {
	unleashSetupViper()

	client, err := NewUnleashClient()
	assert.Nil(t, err)
	assert.Equal(t, "http://conf.example", client.unleashConfig.ApiUrl)
	assert.NotNil(t, client.Client)
	assert.NotNil(t, client)
}

func TestGetFeature(t *testing.T) {
	token := "foobar"
	ctx := context.Background()
	mock := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, fmt.Sprintf("Bearer %s", token), r.Header.Get("Authorization"))
			assert.Equal(t, r.URL.Path, "/api/client/features/test")
			w.Write([]byte(`{"enabled":true,"name":"test","project":"default","type":"release"}`))
		}))

	os.Setenv("UNLEASH_API_URL", mock.URL)
	os.Setenv("UNLEASH_CLIENT_ACCESS_TOKEN", token)
	unleashSetupViper()
	client, err := NewUnleashClient()
	assert.Nil(t, err)
	f, err := client.GetFeature(ctx, "test")
	assert.Nil(t, err)
	assert.True(t, f.Enabled)
	assert.Equal(t, f.Name, "test")
}
