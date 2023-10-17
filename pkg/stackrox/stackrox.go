package stackrox

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/spf13/viper"
)

type StackroxClient struct {
	httpClient *http.Client
	endpoint   string
}

type bearerTokenTransport struct {
	Base  http.RoundTripper
	Token string
}

// RoundTrip is defined within http.RoundTripper and implemented here for bearer token auth usage
func (t *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedReq := req.Clone(req.Context())
	clonedReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.Token))
	return t.Base.RoundTrip(clonedReq)
}

type clientConfig struct {
	Endpoint string
	Timeout  string
	Token    string
}

func newClientConfig() *clientConfig {
	var cfg clientConfig
	sub := util.EnsureViperSub(viper.GetViper(), "advanced_cluster_security")
	sub.SetDefault("timeout", 5)
	sub.BindEnv("timeout", "ACS_API_TIMEOUT")
	sub.BindEnv("endpoint", "ACS_API_ENDPOINT")
	sub.BindEnv("token", "ACS_API_TOKEN")
	if err := sub.Unmarshal(&cfg); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &cfg
}

// NewClient creates a StackRox client with custom transport auth
func NewClient() (*StackroxClient, error) {
	cfg := newClientConfig()
	transport := &bearerTokenTransport{
		Base: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
		},
		Token: cfg.Token,
	}
	t, err := strconv.Atoi(cfg.Timeout)
	if err != nil {
		return nil, err
	}
	return &StackroxClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   time.Second * time.Duration(t),
		},
		endpoint: cfg.Endpoint,
	}, nil

}
