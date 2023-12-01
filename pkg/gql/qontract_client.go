// Package gql adds a client to integration with Qontract-Server
package gql

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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

type useCompareClient string

// UseCompareClientKey is the key used to store the useCompareClient value in the context
var UseCompareClientKey useCompareClient = "useCompareClient"

// QontractClient abstraction for generated GraphQL client
//
//go:generate go run github.com/Khan/genqlient
type QontractClient struct {
	Client        graphql.Client
	CompareClinet *graphql.Client
	config        *qontractConfig
}

type qontractConfig struct {
	Server     string
	Timeout    int
	Token      string
	Retries    int
	CompareSha string
}

func newQontractConfig() *qontractConfig {
	var qc qontractConfig
	sub := util.EnsureViperSub(viper.GetViper(), "graphql")
	sub.SetDefault("timeout", 60)
	sub.SetDefault("retries", 5)
	sub.SetDefault("CompareSha", "")
	sub.BindEnv("server", "GRAPHQL_SERVER")
	sub.BindEnv("timeout", "GRAPHQL_TIMEOUT")
	sub.BindEnv("token", "GRAPHQL_TOKEN")
	sub.BindEnv("retries", "GRAPHQL_RETRIES")
	sub.BindEnv("comparesha", "COMPARE_SHA")
	if err := sub.Unmarshal(&qc); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &qc
}

// NewQontractClient creates a new QontractClient
func NewQontractClient(ctx context.Context) (*QontractClient, error) {
	config := newQontractConfig()
	retryClient := newRetryableHTTPWrapper()

	retryClient.SetTimeout(config.Timeout)
	retryClient.SetRetries(config.Retries)

	if strings.Compare(config.Token, "") != 0 {
		retryClient.SetAuthTransport(&util.AuthedTransport{
			Key:     config.Token,
			Wrapped: http.DefaultTransport,
		})
	}
	client := &QontractClient{
		Client: graphql.NewClient(config.Server, retryClient),
		config: config,
	}

	fmt.Println(config.CompareSha)
	if len(config.CompareSha) > 0 {
		path := fmt.Sprintf("/graphqlsha/%s", config.CompareSha)
		compareClient := graphql.NewClient(strings.ReplaceAll(config.Server, "/graphql", path), retryClient)
		client.CompareClinet = &compareClient
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
		if !util.Contains(allowedIntegrations, schemaUsed.(string)) {
			return fmt.Errorf("usage of schema %s not allowed for integration %s", schemaUsed, integrationName)
		}
	}
	return nil
}

// MakeRequest makes a request to graphql server, ensuring schema usage is allowed
func (c *QontractClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	var client graphql.Client
	useCompare := ctx.Value(UseCompareClientKey)
	if useCompare != nil && useCompare.(bool) == true {
		client = *c.CompareClinet
		if client == nil {
			return fmt.Errorf("compare client not initialized")
		}
	} else {
		client = c.Client
	}
	err := client.MakeRequest(ctx, req, resp)
	if err != nil {
		return err
	}
	integrationName := ctx.Value(reconcile.ContextIngetrationNameKey).(string)
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

type zapLog struct {
	z *zap.SugaredLogger
}

func (z zapLog) Error(msg string, param ...interface{}) {
	z.z.Errorw(msg, param...)
}
func (z zapLog) Info(msg string, param ...interface{}) {
	z.z.Infow(msg, param...)
}
func (z zapLog) Debug(msg string, param ...interface{}) {
	z.z.Debugw(msg, param...)
}
func (z zapLog) Warn(msg string, param ...interface{}) {
	z.z.Warnw(msg, param...)
}

type retryableHTTPWrapper struct {
	Client *retryablehttp.Client
}

// newRetryableHTTPWrapper creates a new retryableHttpWrapper
func newRetryableHTTPWrapper() *retryableHTTPWrapper {
	r := &retryableHTTPWrapper{
		Client: retryablehttp.NewClient(),
	}
	var zapLog retryablehttp.LeveledLogger = zapLog{
		z: util.NoopLog(),
	}
	r.Client.Logger = zapLog
	return r

}

func (r *retryableHTTPWrapper) Do(req *http.Request) (*http.Response, error) {
	reqRetryable, err := retryablehttp.NewRequest(req.Method, req.URL.String(), req.Body)
	reqRetryable.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	return r.Client.Do(reqRetryable)
}

func (r *retryableHTTPWrapper) SetAuthTransport(transport *util.AuthedTransport) {
	r.Client.HTTPClient.Transport = transport
}

func (r *retryableHTTPWrapper) SetTimeout(timeout int) {
	r.Client.HTTPClient.Timeout = time.Duration(timeout) * time.Second
}

func (r *retryableHTTPWrapper) SetRetries(retries int) {
	r.Client.RetryMax = retries
}
