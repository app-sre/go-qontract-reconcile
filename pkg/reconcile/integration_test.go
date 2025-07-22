package reconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestIntegration struct {
	CurrentStateRun bool
	DesiredStateRun bool
	ReconcileRun    bool
	LogDiffRun      bool
	SetUpRun        bool
	ThrowSetupError bool
}

func NewTestIntegration(throwError bool) *TestIntegration {
	return &TestIntegration{
		ThrowSetupError: throwError,
	}
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
	if e.ThrowSetupError {
		return errors.New("setup error")
	}
	return nil
}

var _ Integration = &TestIntegration{}

func TestRunIntegrationAllRun(t *testing.T) {
	runner := IntegrationRunner{
		Runnable: NewTestIntegration(false),
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

func TestRunIntegrationSetupError(t *testing.T) {
	runner := IntegrationRunner{
		Runnable: NewTestIntegration(true),
		config: &runnerConfig{
			Timeout: 10,
		},
	}
	runner.Exiter = func(code int) {
		exitCalled = true
	}
	runner.runIntegration()
	assert.True(t, exitCalled)
	assert.True(t, runner.Runnable.(*TestIntegration).SetUpRun)
	assert.False(t, runner.Runnable.(*TestIntegration).CurrentStateRun)
	assert.False(t, runner.Runnable.(*TestIntegration).DesiredStateRun)
	assert.False(t, runner.Runnable.(*TestIntegration).ReconcileRun)
	assert.False(t, runner.Runnable.(*TestIntegration).LogDiffRun)
}
