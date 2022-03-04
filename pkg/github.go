package pkg

import (
	"context"

	"github.com/google/go-github/v42/github"
	"golang.org/x/oauth2"
)

// NewAuthedGithubClient creates a Github client with custom oauth2 client
func NewAuthedGithubClient(ctx *context.Context, token string) *github.Client {

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(*ctx, ts)

	return github.NewClient(tc)
}
