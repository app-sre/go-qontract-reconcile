package pkg

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/viper"
)

// QontractClient abstraction for generated GraphQL client
type QontractClient struct {
	Client graphql.Client
}

// QontractConfig is used to unmarshal yaml configuration for GQL Clients
type QontractConfig struct {
	ServerURL string
}

// NewQontractConfig creates a new Qontractconfig from viper
func NewQontractConfig(v *viper.Viper) *QontractConfig {
	var qc QontractConfig
	sub := EnsureViperSub(v, "qontract")
	sub.BindEnv("serverurl", "QONTRACT_SERVER_URL")
	if err := sub.Unmarshal(&qc); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &qc
}

// NewQontractClient creates a new QontractClient
func NewQontractClient(config *QontractConfig) *QontractClient {
	return &QontractClient{
		Client: graphql.NewClient(config.ServerURL, http.DefaultClient),
	}
}
