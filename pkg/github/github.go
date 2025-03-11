// Package github provides a client to interact with Github API
package github

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/google/go-github/v69/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// AuthenticatedGithubClient is an oauth2 using Github API client
type AuthenticatedGithubClient struct {
	GithubClient *github.Client
	config       *clientConfig
}

// clientConfig holds configuration GithubClient
type clientConfig struct {
	Timeout int
	BaseURL string
}

func newGithubClientConfig() *clientConfig {
	var qc clientConfig
	sub := util.EnsureViperSub(viper.GetViper(), "github")
	sub.SetDefault("timeout", 60)
	sub.BindEnv("baseurl", "GITHUB_API")
	sub.BindEnv("timeout", "GITHUB_API_TIMEOUT")
	if err := sub.Unmarshal(&qc); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &qc
}

// NewAuthenticatedGithubClient creates a Github client with custom oauth2 client
func NewAuthenticatedGithubClient(ctx context.Context, token string) (*AuthenticatedGithubClient, error) {
	config := newGithubClientConfig()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	tc.Timeout = time.Duration(config.Timeout) * time.Second

	client := github.NewClient(tc)
	if strings.Compare(config.BaseURL, "") != 0 {
		actualBaseURL := config.BaseURL
		if !strings.HasSuffix(config.BaseURL, "/") {
			util.Log().Debugw("Github Base Url has no / suffix, addding it", "url", config.BaseURL)
			actualBaseURL = config.BaseURL + "/"
		}
		baseURL, err := url.Parse(actualBaseURL)
		if err != nil {
			return nil, err
		}
		client.BaseURL = baseURL
	}

	return &AuthenticatedGithubClient{
		GithubClient: client,
		config:       config,
	}, nil
}

// GetUsers uses authenticated client to get user information
func (c *AuthenticatedGithubClient) GetUsers(ctx context.Context, user string) (*github.User, error) {
	ghUser, _, err := c.GithubClient.Users.Get(ctx, user)
	return ghUser, err
}
