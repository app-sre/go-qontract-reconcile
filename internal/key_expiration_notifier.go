package internal

import (
	"context"

	"github.com/app-sre/user-validator/internal/queries"
	. "github.com/app-sre/user-validator/pkg"
)

type KeyExpirationNotifier struct {
	state Persistence
}

type KeyExpirationNotifierConfig struct {
}

func NewKeyExpirationNotifier() *KeyExpirationNotifier {
	notifier := KeyExpirationNotifier{
		state: NewMemoryState(),
	}
	return &notifier
}

// CurrentState in our sense is every expired key
func (n *KeyExpirationNotifier) CurrentState(ctx context.Context, ri *ResourceInventory) error {
	users, err := queries.Users(ctx)
	if err != nil {
		return err
	}
	for _, user := range users.GetUsers_v1() {
		pgpKey := user.GetPublic_gpg_key()

		if len(pgpKey) > 0 {
			Log().Debugw("Decoding key for", "user", user.GetName())
			entity, err := DecodePgpKey(pgpKey, user.GetPath())
			if err != nil {
				Log().Debug("Error reading key")
				ri.AddResourceState(&ResourceState{
					Action:  Create,
					Target:  user,
					Desired: "notified",
				})
				continue
			}

			Log().Debugw("Testing encryption for", "user", user.GetName())
			err = TestEncrypt(entity)
			if err != nil {
				Log().Debug("Error testing encryption")
				ri.AddResourceState(&ResourceState{
					Action:  Create,
					Target:  user,
					Desired: "notified",
				})
			}
		}
	}

	return nil
}

// DesiredState checks if notifications have been sent before
func (n *KeyExpirationNotifier) DesiredState(ctx context.Context, ri *ResourceInventory) error {
	for _, state := range ri.State {
		user := state.Target.(queries.UsersUsers_v1User_v1)
		if err, sent := n.state.Exists(user.GetOrg_username()); err == nil {
			if !sent {
				state.Current = "missing"
			} else {
				state.Current = "exists"
			}
		}
	}
	return nil
}

// Sent notifications and add them to the state
func (n *KeyExpirationNotifier) Reconcile(ctx context.Context, ri *ResourceInventory) error {
	for _, state := range ri.State {
		if state.Current == "missing" {
			user := state.Target.(queries.UsersUsers_v1User_v1)
			Log().Infow("Sending notification to user", "user", user.GetOrg_username())
		}
	}
	return nil
}

func (n *KeyExpirationNotifier) Setup() error {
	return nil
}
