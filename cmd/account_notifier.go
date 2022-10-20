package cmd

import (
	"github.com/app-sre/user-validator/internal"
	. "github.com/app-sre/user-validator/pkg"
)

func accountNotifier() {
	notifier := internal.NewAccountNotifier()
	runner := NewIntegrationRunner(notifier, internal.ACCOUNT_NOTIFIER_NAME)
	runner.Run()
}
