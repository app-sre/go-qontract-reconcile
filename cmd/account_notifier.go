package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal/accountnotifier"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
)

func accountNotifier() {
	notifier := accountnotifier.NewAccountNotifier()
	runner := reconcile.NewIntegrationRunner(notifier, accountnotifier.ACCOUNT_NOTIFIER_NAME)
	runner.Run()
}
