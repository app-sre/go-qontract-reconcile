package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal"
	. "github.com/app-sre/go-qontract-reconcile/pkg"
)

func accountNotifier() {
	notifier := internal.NewAccountNotifier()
	runner := NewIntegrationRunner(notifier, internal.ACCOUNT_NOTIFIER_NAME)
	runner.Run()
}
