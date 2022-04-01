package pkg

import (
	"context"
	"os"
	"time"

	"github.com/spf13/viper"
)

type IntegrationNameKey string

var ContextIngetrationNameKey IntegrationNameKey = "integrationName"

// RunnerConfig is used to unmarshal yaml configuration Runners
type runnerConfig struct {
	Timeout         int
	IntegrationName string
}

// NewRunnerConfig creates a new IntegationConfig from viper, v can be nil
func newRunnerConfig() *runnerConfig {
	v := viper.GetViper()
	var ic runnerConfig
	v.SetDefault("timeout", 0)

	v.BindEnv("timeout", "RUNNER_TIMEOUT")
	v.BindEnv("integrationname", "RUNNER_INTEGRATION_NAME")

	if err := v.Unmarshal(&ic); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}

	return &ic
}

// Integration describes the set of methods Integrations must implement
type Integration interface {
	Diff() ([]Diff, error)
	Reconcile() error
	Setup() error
}

// Diff must be used to describe the diff an integration found
type Diff struct {
	Action  string
	Target  string
	Current string
	Desired string
}

// Validation describes the methods an Validation must implement
type Validation interface {
	// Setup method is used to fetch secrets, setup clients or prepare state...
	Setup(context.Context) error
	// Validate is doing the actual validation
	Validate(context.Context) ([]ValidationError, error)
}

// ValidationError contains errors, that are discovered during Validate()
type ValidationError struct {
	Path       string
	Validation string
	Error      error
}

type exitFunc func(int)

// Runner can be used to actually run Validations or Integrations
type Runner interface {
	Run() error
	Exiter(int)
}

// ValidationRunner is an implementation of Runner
type ValidationRunner struct {
	Target Validation
	Exiter exitFunc
	config *runnerConfig
}

// NewValidationRunner creates a ValidationRunner for a given Validation
func NewValidationRunner(target Validation) *ValidationRunner {
	c := newRunnerConfig()
	v := &ValidationRunner{
		Target: target,
		config: c,
	}
	v.Exiter = func(exitCode int) {
		os.Exit(exitCode)
	}
	return v
}

// Run executes the validation configured as target
func (v *ValidationRunner) Run() error {
	ctx := context.WithValue(context.Background(), ContextIngetrationNameKey, v.config.IntegrationName)
	var cancel func()
	if v.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(v.config.Timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	if err := v.Target.Setup(ctx); err != nil {
		return err
	}
	validationErrors, err := v.Target.Validate(ctx)
	if err != nil {
		return err
	}
	if len(validationErrors) > 0 {
		for _, e := range validationErrors {
			Log().Infow("Validation error", "path", e.Path, "validation", e.Validation, "error", e.Error.Error())
		}
		v.Exiter(1)
	}
	return nil
}
