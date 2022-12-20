package pkg

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ConcatValidationErrors can be used to merge two list of ValiudationErros
func ConcatValidationErrors(a, b []ValidationError) []ValidationError {
	allErrors := make([]ValidationError, len(a)+len(b))
	copy(allErrors, a)
	for i, e := range b {
		allErrors[len(a)+i] = e
	}
	return allErrors
}

// Log returns the SuggardLoggar that can be used accross integrations
func Log() *zap.SugaredLogger {
	return zap.L().Sugar()
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

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func StrPointer(s string) *string {
	return &s
}

func ReadKeyFile(t *testing.T, fileName string) []byte {
	key, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("Could not read public key test data %s, error: %s", fileName, err.Error())
	}
	return key
}
