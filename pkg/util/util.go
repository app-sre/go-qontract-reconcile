// Package util has code that does not fit anywhere else
package util

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Log returns the SuggardLoggar that can be used accross integrations
func Log() *zap.SugaredLogger {
	return zap.L().Sugar()
}

// NoopLog returns a no-op logger, that can be used to supress logging
func NoopLog() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}

// EnsureViperSub will return a viper sub if available or create one
func EnsureViperSub(viper *viper.Viper, key string) *viper.Viper {
	sub := viper.Sub(key)
	if sub != nil {
		return sub
	}
	fakeSub := make(map[string]interface{})
	viper.Set(key, fakeSub)
	return viper.Sub(key)
}

// Contains checks if a string is in a slice
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// StrPointer returns a pointer to a string
func StrPointer(s string) *string {
	return &s
}

// ReadKeyFile reads a key file and returns the content
func ReadKeyFile(t *testing.T, fileName string) []byte {
	key, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("Could not read public key test data %s, error: %s", fileName, err.Error())
	}
	return key
}

// AuthedTransport is a http.RoundTripper that adds an Authorization header
type AuthedTransport struct {
	Key     string
	Wrapped http.RoundTripper
}

// RoundTrip implements http.RoundTripper and sets Authorization header
func (t *AuthedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.Key)
	return t.Wrapped.RoundTrip(req)
}

// NewHTTPTestServer returns a new httptest.Server with a given handlerFunc
func NewHTTPTestServer(handlerFunc func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handlerFunc))
}
