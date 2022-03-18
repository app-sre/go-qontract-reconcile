package pkg

import (
	"context"

	"github.com/google/go-github/v42/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// AuthenticatedGithubClient is an oauth2 using Github API client
type AuthenticatedGithubClient struct {
	GithubClient *github.Client
}

// GithubClientConfig holds configuration GithubClient
type GithubClientConfig struct {
	Timeout int
	BaseURL string
}

// NewGithubClientConfig creates a new GithubClientConfig from viper
func NewGithubClientConfig(v *viper.Viper) *GithubClientConfig {
	var qc GithubClientConfig
	sub := EnsureViperSub(v, "github")
	sub.SetDefault("timeout", 60)
	sub.BindEnv("baseurl", "GITHUB_API")
	sub.BindEnv("timeout", "GITHUB_API_TIMEOUT")
	if err := sub.Unmarshal(&qc); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &qc
}

// NewAuthenticatedGithubClient creates a Github client with custom oauth2 client
func NewAuthenticatedGithubClient(ctx context.Context, token string) *AuthenticatedGithubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &AuthenticatedGithubClient{
		GithubClient: github.NewClient(tc),
	}
}

func (c *AuthenticatedGithubClient) GetUsers(ctx context.Context, user string) *github.User {
	ghUser, _, _ := c.GithubClient.Users.Get(ctx, user)
	return ghUser
}
