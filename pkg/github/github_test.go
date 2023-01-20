package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAuthedGithubClient(t *testing.T) {
	ctx := context.Background()
	token := "FOOBAR"
	client, err := NewAuthenticatedGithubClient(ctx, token)
	assert.NotNil(t, client)
	assert.Nil(t, err)
}
