package pkg

import (
	"net/http"
	"strings"
	"time"

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
	Timeout   int
	Token     string
}

func newQontractConfig() *qontractConfig {
	var qc qontractConfig
	sub := EnsureViperSub(viper.GetViper(), "qontract")
	sub.SetDefault("timeout", 60)
	sub.BindEnv("serverurl", "QONTRACT_SERVER_URL")
	sub.BindEnv("timeout", "QONTRACT_TIMEOUT")
	sub.BindEnv("token", "QONTRACT_TOKEN")
	if err := sub.Unmarshal(&qc); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &qc
}

type authedTransport struct {
	key     string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.key)
	return t.wrapped.RoundTrip(req)
}

// NewQontractClient creates a new QontractClient
func NewQontractClient() *QontractClient {
	config := newQontractConfig()
	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	if strings.Compare(config.Token, "") != 0 {
		httpClient.Transport = &authedTransport{
			key:     config.Token,
			wrapped: http.DefaultTransport,
		}
	}
	client := &QontractClient{
		Client: graphql.NewClient(config.ServerURL, httpClient),
		config: config,
	}
	return client
}
