package cmd

import (
	"github.com/app-sre/user-validator/internal"
	. "github.com/app-sre/user-validator/pkg"
)

func userValidator() {
	validator := internal.NewValidateUser()
	runner := NewValidationRunner(validator, "user-validator")
	err := runner.Run()
	if err != nil {
		Log().Errorw("Error during integration", "error", err.Error())
	}
}
