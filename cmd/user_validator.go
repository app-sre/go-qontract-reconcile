package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal"
	. "github.com/app-sre/go-qontract-reconcile/pkg"
)

func userValidator() {
	validator := internal.NewValidateUser()
	runner := NewValidationRunner(validator, "user-validator")
	runner.Run()
}
