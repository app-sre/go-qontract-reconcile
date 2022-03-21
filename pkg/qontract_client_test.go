package pkg

import (
	"os"
	"testing"

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
