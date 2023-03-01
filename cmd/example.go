package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal/example"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
)

func exampleIntegration() {
	notifier := example.NewExample()
	runner := reconcile.NewIntegrationRunner(notifier, example.EXAMPLE_INTEGRATION_NAME)
	runner.Run()
}
