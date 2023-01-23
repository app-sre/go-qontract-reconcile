package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
)

func accountNotifier() {
	notifier := internal.NewAccountNotifier()
	runner := reconcile.NewIntegrationRunner(notifier, internal.ACCOUNT_NOTIFIER_NAME)
	runner.Run()
}
