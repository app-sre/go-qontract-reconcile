package pkg

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v42/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// AuthenticatedGithubClient is an oauth2 using Github API client
type AuthenticatedGithubClient struct {
	GithubClient *github.Client
	config       *GithubClientConfig
}

// GithubClientConfig holds configuration GithubClient
type GithubClientConfig struct {
	Timeout int
	BaseURL string
}

func newGithubClientConfig() *GithubClientConfig {
	var qc GithubClientConfig
	sub := EnsureViperSub(viper.GetViper(), "github")
	sub.SetDefault("timeout", 60)
	sub.BindEnv("baseurl", "GITHUB_API")
	sub.BindEnv("timeout", "GITHUB_API_TIMEOUT")
	if err := sub.Unmarshal(&qc); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
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
		actualBaseUrl := config.BaseURL
		if !strings.HasSuffix(config.BaseURL, "/") {
			Log().Debugw("Github Base Url has no / suffix, addding it", "url", config.BaseURL)
			actualBaseUrl = config.BaseURL + "/"
		}
		baseUrl, err := url.Parse(actualBaseUrl)
		if err != nil {
			return nil, err
		}
		client.BaseURL = baseUrl
	}

	return &AuthenticatedGithubClient{
		GithubClient: client,
		config:       config,
	}, nil
}

func (c *AuthenticatedGithubClient) GetUsers(ctx context.Context, user string) (*github.User, error) {
	ghUser, _, err := c.GithubClient.Users.Get(ctx, user)
	return ghUser, err
}
