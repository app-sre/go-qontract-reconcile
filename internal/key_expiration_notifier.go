package internal

import (
	"context"

	. "github.com/app-sre/user-validator/pkg"
)

type KeyExpirationNotifier struct {
}

// CurrentState in our sense is every expired key
func (n *KeyExpirationNotifier) CurrentState(context.Context, *ResourceInventory) error {
	return nil
}

// DesiredState checks if notifications have been sent before
func (n *KeyExpirationNotifier) DesiredState(context.Context, *ResourceInventory) error {
	return nil
}

// Sent notifications and add them to the state
func (n *KeyExpirationNotifier) Reconcile(context.Context, *ResourceInventory) error {
	return nil
}

func (n *KeyExpirationNotifier) Setup() error {
	return nil
}
