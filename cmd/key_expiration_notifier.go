package cmd

import (
	"github.com/app-sre/user-validator/internal"
	. "github.com/app-sre/user-validator/pkg"
)

func keyExpirationNotifier() {
	notifier := internal.NewKeyExpirationNotifier()
	runner := NewIntegrationRunner(notifier, internal.KEY_EXPIRATION_NOTIFIER_NAME)
	runner.Run()
}
