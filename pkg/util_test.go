package pkg

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestConcatValidationErrorOkay(t *testing.T) {
	a := []ValidationError{{Path: "/foo/rab"}}
	b := []ValidationError{{Path: "/foo/bar"}}
	c := ConcatValidationErrors(a, b)
	assert.Len(t, c, 2)
	assert.Contains(t, c, a[0])
	assert.Contains(t, c, b[0])
}

func TestEnsureViperSubEmpty(t *testing.T) {
	v := viper.New()
	sub := EnsureViperSub(v, "foo")
	assert.NotNil(t, v.Get("foo"))
	assert.NotNil(t, sub)
}

func TestEnsureViperSub(t *testing.T) {
	v := viper.New()
	values := make(map[string]interface{})
	values["test"] = "bar"
	v.Set("foo", values)
	sub := EnsureViperSub(v, "foo")
	assert.NotNil(t, sub)
	assert.Equal(t, "bar", sub.Get("test"))
}
