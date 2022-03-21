package pkg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAuthedGithubClient(t *testing.T) {
	ctx := context.Background()
	token := "FOOBAR"
	client := NewAuthenticatedGithubClient(ctx, token)
	assert.NotNil(t, client)
}
