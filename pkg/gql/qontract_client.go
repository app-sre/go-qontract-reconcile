package gql

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/app-sre/user-validator/pkg"
	"github.com/spf13/viper"
)

var _ graphql.Client = &QontractClient{}

var _ = `# @genqlient 
	query integrations {
		integrations: integrations_v1 {
        name
        description
        schemas
    }
}`

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
	sub := pkg.EnsureViperSub(viper.GetViper(), "qontract")
	sub.SetDefault("timeout", 60)
	sub.BindEnv("serverurl", "QONTRACT_SERVER_URL")
	sub.BindEnv("timeout", "QONTRACT_TIMEOUT")
	sub.BindEnv("token", "QONTRACT_TOKEN")
	if err := sub.Unmarshal(&qc); err != nil {
		pkg.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
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
func NewQontractClient(ctx context.Context) (*QontractClient, error) {
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
	return client, nil
}

func (c *QontractClient) ensureSchema(integrationName string, resp *graphql.Response, integrationsResponse *integrationsResponse) error {
	var allowedIntegrations []string
	for _, integration := range integrationsResponse.GetIntegrations() {
		if integration.Name == integrationName {
			allowedIntegrations = integration.GetSchemas()
		}
	}

	if resp.Extensions == nil || resp.Extensions["schemas"] == nil {
		return fmt.Errorf("missing correct extensions from graphql response")
	}
	extensions := resp.Extensions["schemas"]
	for _, schemaUsed := range extensions.([]interface{}) {
		if !pkg.Contains(allowedIntegrations, schemaUsed.(string)) {
			return fmt.Errorf("usage of schema %s not allowed for integration %s", schemaUsed, integrationName)
		}
	}
	return nil
}

func (c *QontractClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	err := c.Client.MakeRequest(ctx, req, resp)
	if err != nil {
		return err
	}
	integrationName := ctx.Value(pkg.ContextIngetrationNameKey).(string)
	integrationsResponse, err := integrations(ctx, c.Client)
	if err != nil {
		return err
	}
	err = c.ensureSchema(integrationName, resp, integrationsResponse)
	if err != nil {
		return err
	}
	return nil
}
