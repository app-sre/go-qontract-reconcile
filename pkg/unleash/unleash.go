// Package unleash contains a client for integrating with Unleash
package unleash

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Not using github.com/Unleash/unleash-client-go/v3
// As of 2022/04 it is only tested with go 1.13
// Also encountered couple of issues like weak error handling or not working metrics

type unleashConfig struct {
	Timeout           int
	APIURL            string
	ClientAccessToken string
}

// Client is a simple abstraction for the Unleash API
type Client struct {
	Client        *http.Client
	unleashConfig *unleashConfig
}

func newUnleasConfig() *unleashConfig {
	sub := util.EnsureViperSub(viper.GetViper(), "unleash")
	var c unleashConfig

	sub.SetDefault("timeout", 60)

	sub.BindEnv("timeout", "UNLEASH_TIMEOUT")
	sub.BindEnv("apiurl", "UNLEASH_API_URL")
	sub.BindEnv("clientaccesstoken", "UNLEASH_CLIENT_ACCESS_TOKEN")

	if err := sub.Unmarshal(&c); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}

	return &c
}

// NewUnleashClient creates a new UnleashClient
func NewUnleashClient() (*Client, error) {
	c := newUnleasConfig()

	return &Client{
		Client: &http.Client{
			Timeout: time.Duration(c.Timeout) * time.Second,
			Transport: &util.AuthedTransport{
				Key:     c.ClientAccessToken,
				Wrapped: http.DefaultTransport,
			},
		},
		unleashConfig: c,
	}, nil
}

// GetFeature returns a feature from Unleash
// Dept: split up this method if you add new URLs, do not just copy and paste it!
func (c *Client) GetFeature(ctx context.Context, name string) (*Feature, error) {
	util.Log().Debugw("Checking if feature is enabled", "feature", name)
	path := fmt.Sprintf("%s/client/features/%s", c.unleashConfig.APIURL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
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
