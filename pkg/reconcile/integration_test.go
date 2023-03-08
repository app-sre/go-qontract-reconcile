package reconcile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestIntegration struct {
	CurrentStateRun bool
	DesiredStateRun bool
	ReconcileRun    bool
	LogDiffRun      bool
	SetUpRun        bool
}

func NewTestIntegration() *TestIntegration {
	return &TestIntegration{}
}

func (e *TestIntegration) CurrentState(context.Context, *ResourceInventory) error {
	e.CurrentStateRun = true
	return nil
}

func (e *TestIntegration) DesiredState(context.Context, *ResourceInventory) error {
	e.DesiredStateRun = true
	return nil
}

func (e *TestIntegration) Reconcile(context.Context, *ResourceInventory) error {
	e.ReconcileRun = true
	return nil
}

func (e *TestIntegration) LogDiff(*ResourceInventory) {
	e.LogDiffRun = true
}

func (e *TestIntegration) Setup(context.Context) error {
	e.SetUpRun = true
	return nil
}

var _ Integration = &TestIntegration{}

func TestRunIntegrationAllRun(t *testing.T) {
	runner := IntegrationRunner{
		Runnable: NewTestIntegration(),
		config: &runnerConfig{
			Timeout: 10,
		},
	}

	runner.runIntegration()
	assert.True(t, runner.Runnable.(*TestIntegration).CurrentStateRun)
	assert.True(t, runner.Runnable.(*TestIntegration).DesiredStateRun)
	assert.True(t, runner.Runnable.(*TestIntegration).ReconcileRun)
	assert.True(t, runner.Runnable.(*TestIntegration).LogDiffRun)
	assert.True(t, runner.Runnable.(*TestIntegration).SetUpRun)
}
