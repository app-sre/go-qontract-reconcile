package reconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type throwErrorSettings struct {
	ThrowCurrentStateRunError bool
	ThrowDesiredStateRunError bool
	ThrowReconcileRunError    bool
	ThrowSetUpRunError        bool
}
type TestIntegration struct {
	CurrentStateRun bool
	DesiredStateRun bool
	ReconcileRun    bool
	LogDiffRun      bool
	SetUpRun        bool
	ThrowSetupError bool
	errorSettings   throwErrorSettings
}

func NewTestIntegration(errorSettings throwErrorSettings) *TestIntegration {
	return &TestIntegration{
		errorSettings: errorSettings,
	}
}

func (e *TestIntegration) CurrentState(context.Context, *ResourceInventory) error {
	if e.errorSettings.ThrowCurrentStateRunError {
		return errors.New("current state error")
	}
	e.CurrentStateRun = true
	return nil
}

func (e *TestIntegration) DesiredState(context.Context, *ResourceInventory) error {
	if e.errorSettings.ThrowDesiredStateRunError {
		return errors.New("desired state error")
	}
	e.DesiredStateRun = true
	return nil
}

func (e *TestIntegration) Reconcile(context.Context, *ResourceInventory) error {
	if e.errorSettings.ThrowReconcileRunError {
		return errors.New("reconcile error")
	}
	e.ReconcileRun = true
	return nil
}

func (e *TestIntegration) LogDiff(*ResourceInventory) {
	e.LogDiffRun = true
}

func (e *TestIntegration) Setup(context.Context) error {
	if e.errorSettings.ThrowSetUpRunError {
		return errors.New("setup error")
	}
	e.SetUpRun = true
	return nil
}

var _ Integration = &TestIntegration{}

func TestRunIntegrationAllRun(t *testing.T) {
	errorSettings := throwErrorSettings{}
	runner := IntegrationRunner{
		Runnable: NewTestIntegration(errorSettings),
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

func TestRunIntegrationErrors(t *testing.T) {
	type testCase struct {
		name          string
		errorSettings throwErrorSettings
		shouldFail    bool
	}

	testCases := []testCase{
		{name: "setup error", errorSettings: throwErrorSettings{ThrowSetUpRunError: true}, shouldFail: true},
		{name: "current state error", errorSettings: throwErrorSettings{ThrowCurrentStateRunError: true}, shouldFail: true},
		{name: "desired state error", errorSettings: throwErrorSettings{ThrowDesiredStateRunError: true}, shouldFail: true},
		{name: "reconcile error", errorSettings: throwErrorSettings{ThrowReconcileRunError: true}, shouldFail: true},
		{name: "run successfully", errorSettings: throwErrorSettings{}, shouldFail: false},
	}

	for _, testCase := range testCases {
		exitCalled := false
		runner := IntegrationRunner{
			Runnable: NewTestIntegration(testCase.errorSettings),
			config: &runnerConfig{
				Timeout: 10,
			},
			Exiter: func(exitCode int) {
				exitCalled = true
			},
		}
		runner.runIntegration()
		if testCase.shouldFail {
			assert.True(t, exitCalled)
		} else {
			assert.False(t, exitCalled)
		}
	}
}
