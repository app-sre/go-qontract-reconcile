package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal/keyvalidator"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
)

func validateKey() {
	validator := keyvalidator.NewKeyValidator()
	runner := reconcile.NewValidationRunner(validator, "validate-key")
	runner.Run()
}
