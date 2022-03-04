package pkg

import (
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
