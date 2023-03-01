package example

import (
	"context"

	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
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

func (e *Example) CurrentState(context.Context, *reconcile.ResourceInventory) error {
	util.Log().Infow("Getting current state")
	return nil
}

func (e *Example) DesiredState(context.Context, *reconcile.ResourceInventory) error {
	util.Log().Infow("Getting desired state")
	return nil
}

func (e *Example) Reconcile(context.Context, *reconcile.ResourceInventory) error {
	util.Log().Infow("Reconciling")
	return nil
}

func (e *Example) LogDiff(*reconcile.ResourceInventory) {
	util.Log().Infow("Logging diff")
}

func (e *Example) Setup(context.Context) error {
	util.Log().Infow("Setting up example integration")
	return nil
}
