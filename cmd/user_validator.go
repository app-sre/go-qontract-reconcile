package cmd

import (
	"github.com/janboll/user-validator/internal"
	. "github.com/janboll/user-validator/pkg"
	"github.com/spf13/viper"
)

func userValidator() {
	validator := internal.NewValidateUser(internal.NewValidateUserConfig(viper.GetViper()), NewRunnerConfig(nil))
	runner := NewValidationRunner(validator)
	err := runner.Run()
	if err != nil {
		Log().Errorw("Error during integration", "error", err.Error())
	}
}
