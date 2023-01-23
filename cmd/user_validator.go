package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
)

func userValidator() {
	validator := internal.NewValidateUser()
	runner := reconcile.NewValidationRunner(validator, "user-validator")
	runner.Run()
}
