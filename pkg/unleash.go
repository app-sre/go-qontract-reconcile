package pkg

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Not using github.com/Unleash/unleash-client-go/v3
// As of 2022/04 it is only tested with go 1.13
// Also encountered couple of issues like weak error handling or not working metrics

type unleashConfig struct {
	Timeout           int
	ApiUrl            string
	ClientAccessToken string
}

type UnleashClient struct {
	Client        *http.Client
	unleashConfig *unleashConfig
}

func newUnleasConfig() *unleashConfig {
	sub := EnsureViperSub(viper.GetViper(), "unleash")
	var c unleashConfig

	sub.SetDefault("timeout", 60)

	sub.BindEnv("timeout", "UNLEASH_TIMEOUT")
	sub.BindEnv("apiurl", "UNLEASH_API_URL")
	sub.BindEnv("clientaccesstoken", "UNLEASH_CLIENT_ACCESS_TOKEN")

	if err := sub.Unmarshal(&c); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}

	return &c
}

type authedTransport struct {
	key     string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.key))
	return t.wrapped.RoundTrip(req)
}

func NewUnleashClient() (*UnleashClient, error) {
	c := newUnleasConfig()

	return &UnleashClient{
		Client: &http.Client{
			Timeout: time.Duration(c.Timeout) * time.Second,
			Transport: &authedTransport{
				key:     c.ClientAccessToken,
				wrapped: http.DefaultTransport,
			},
		},
		unleashConfig: c,
	}, nil
}

// Dept: split up this method if you add new URLs, do not just copy and paste it!
func (c *UnleashClient) GetFeature(ctx context.Context, name string) (*Feature, error) {
	Log().Debugw("Checking if feature is enabled", "feature", name)
	path := fmt.Sprintf("%s/api/client/features/%s", c.unleashConfig.ApiUrl, name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feature Feature
	err = yaml.Unmarshal(body, &feature)
	if err != nil {
		return nil, err
	}
	return &feature, nil
}
