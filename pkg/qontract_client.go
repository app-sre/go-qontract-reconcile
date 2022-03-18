package pkg

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/viper"
)

// QontractClient abstraction for generated GraphQL client
type QontractClient struct {
	Client graphql.Client
	config *qontractConfig
}

type qontractConfig struct {
	ServerURL string
}

func newQontractConfig() *qontractConfig {
	var qc qontractConfig
	sub := EnsureViperSub(viper.GetViper(), "qontract")
	sub.BindEnv("serverurl", "QONTRACT_SERVER_URL")
	if err := sub.Unmarshal(&qc); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &qc
}

// NewQontractClient creates a new QontractClient
func NewQontractClient() *QontractClient {
	config := newQontractConfig()
	return &QontractClient{
		Client: graphql.NewClient(config.ServerURL, http.DefaultClient),
		config: config,
	}
}
