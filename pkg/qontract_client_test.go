package pkg

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func qontractSetupViper() *viper.Viper {
	v := viper.New()

	qontractCfg := make(map[string]interface{})
	qontractCfg["serverurl"] = "http://conf.example"

	v.Set("qontract", qontractCfg)
	return v
}

func qontractSetupViperEnv() *viper.Viper {
	v := viper.New()
	os.Setenv("QONTRACT_SERVER_URL", "http://env.example")

	qontractCfg := make(map[string]interface{})
	v.Set("qontract", qontractCfg)
	return v
}

func TestNewQontractClient(t *testing.T) {
	qc := NewQontractConfig(qontractSetupViper())
	assert.Equal(t, "http://conf.example", qc.ServerURL)

	client := NewQontractClient(qc)
	assert.NotNil(t, client)
}

func TestNewQontractClientEnv(t *testing.T) {
	qc := NewQontractConfig(qontractSetupViperEnv())
	assert.Equal(t, "http://env.example", qc.ServerURL)

	client := NewQontractClient(qc)
	assert.NotNil(t, client)
}
