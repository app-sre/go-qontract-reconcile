package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal/uservalidator"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
)

func userValidator() {
	validator := uservalidator.NewValidateUser()
	runner := reconcile.NewValidationRunner(validator, "user-validator")
	runner.Run()
}
