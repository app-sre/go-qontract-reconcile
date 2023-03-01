package example

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var EXAMPLE_INTEGRATION_NAME = "example"

type ExampleConfig struct {
	Tempdir string
}

func newExampleConfig() *ExampleConfig {
	var ec ExampleConfig
	sub := util.EnsureViperSub(viper.GetViper(), "example")
	sub.SetDefault("tempdir", "/tmp/example")
	sub.BindEnv("tempdir", "EXAMPLE_TEMPDIR")
	if err := sub.Unmarshal(&ec); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &ec
}

type Example struct {
	config *ExampleConfig
}

func NewExample() *Example {
	ec := newExampleConfig()
	return &Example{config: ec}
}

type UserFiles struct {
	FileNames string
	GpgKey    string
}

func (e *Example) CurrentState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	util.Log().Infow("Getting current state")

	files, err := ioutil.ReadDir(e.config.Tempdir)
	if err != nil {
		return errors.Wrap(err, "Error while reading workdir")
	}

	for _, f := range files {
		absolutePath := e.config.Tempdir + "/" + f.Name()
		util.Log().Debugw("Found file", "file", absolutePath)
		content, err := ioutil.ReadFile(absolutePath)
		if err != nil {
			return errors.Wrap(err, "Error while reading file")
		}
		rs := &reconcile.ResourceState{
			Current: &UserFiles{
				FileNames: f.Name(),
				GpgKey:    string(content),
			},
		}
		ri.AddResourceState(f.Name(), rs)
	}

	return nil
}

func (e *Example) DesiredState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	util.Log().Infow("Getting desired state")

	users, err := Users(ctx)

	if err != nil {
		return errors.Wrap(err, "Error while getting users")
	}

	for _, user := range users.GetUsers_v1() {
		state := ri.GetResourceState(user.GetOrg_username())
		if state == nil {
			state = &reconcile.ResourceState{}
		}
		state.Desired = &UserFiles{
			FileNames: user.GetOrg_username(),
			GpgKey:    user.GetPublic_gpg_key(),
		}
	}

	return nil
}

func (e *Example) Reconcile(ctx context.Context, ri *reconcile.ResourceInventory) error {
	util.Log().Infow("Reconciling")
	return nil
}

func (e *Example) LogDiff(ri *reconcile.ResourceInventory) {
	util.Log().Debugw("Logging diff")

	for _, state := range ri.State {
		var current, desired *UserFiles
		if state.Current != nil {
			current = state.Current.(*UserFiles)
		}
		if state.Desired != nil {
			desired = state.Desired.(*UserFiles)
		}
		if current != nil && desired == nil {
			util.Log().Infow("Deleting", "file", current.FileNames)
		} else if current == nil || current.GpgKey != desired.GpgKey {
			util.Log().Infow("Updating", "file", desired.FileNames)
		}
	}
}

func (e *Example) Setup(context.Context) error {
	util.Log().Infow("Setting up example integration")
	err := os.MkdirAll(e.config.Tempdir, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "Error while creating workdir")
	}

	return nil
}
