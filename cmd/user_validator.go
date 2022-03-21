package cmd

import (
	"github.com/janboll/user-validator/internal"
	. "github.com/janboll/user-validator/pkg"
)

func userValidator() {
	validator := internal.NewValidateUser()
	runner := NewValidationRunner(validator)
	err := runner.Run()
	if err != nil {
		Log().Errorw("Error during integration", "error", err.Error())
	}
}
