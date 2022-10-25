package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/app-sre/user-validator/internal/queries"
	. "github.com/app-sre/user-validator/pkg"
)

var KEY_EXPIRATION_NOTIFIER_NAME = "key-expiration-notifier"

type KeyExpirationNotifier struct {
	state Persistence
}

type KeyExpirationNotifierConfig struct {
}

func NewKeyExpirationNotifier() *KeyExpirationNotifier {
	notifier := KeyExpirationNotifier{}
	return &notifier
}

type notificationStatus struct {
	status string
	sentAt time.Time
}

func (n *KeyExpirationNotifier) CurrentState(ctx context.Context, ri *ResourceInventory) error {
	users, err := queries.Users(ctx)
	if err != nil {
		return err
	}
	for _, user := range users.GetUsers_v1() {
		err, state := n.state.Exists(ctx, user.GetOrg_username())
		if err != nil {
			return err
		}

		if state {
			var ns notificationStatus

			n.state.Get(ctx, user.GetOrg_username(), &ns)

			ri.AddResourceState(user.GetOrg_username(), &ResourceState{
				Current: ns,
			})
		}
	}
	return nil
}

func (n *KeyExpirationNotifier) DesiredState(ctx context.Context, ri *ResourceInventory) error {
	users, err := queries.Users(ctx)
	if err != nil {
		return err
	}

	for _, user := range users.GetUsers_v1() {
		pgpKey := user.GetPublic_gpg_key()

		state := ri.GetResourceState(user.GetOrg_username())

		if len(pgpKey) > 0 {
			Log().Debugw("Decoding key for", "user", user.GetName())
			entity, err := DecodePgpKey(pgpKey, user.GetPath())
			if err != nil {
				Log().Debug("Error reading key")
				state.Desired = notificationStatus{
					status: "send",
				}
				continue
			}

			Log().Debugw("Testing encryption for", "user", user.GetName())
			err = TestEncrypt(entity)
			if err != nil {
				Log().Debug("Error testing encryption")
				state.Desired = notificationStatus{
					status: "send",
				}
			}
		}
	}

	return nil
}

// Sent notifications and add them to the state
func (n *KeyExpirationNotifier) Reconcile(ctx context.Context, ri *ResourceInventory) error {
	for _, state := range ri.State {
		fmt.Println(state)
	}
	return nil
}

func (n *KeyExpirationNotifier) Setup(ctx context.Context) error {
	n.state = NewS3State("integrations", KEY_EXPIRATION_NOTIFIER_NAME, NewClient(ctx))

	return nil
}
