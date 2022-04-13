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
	Timeout          int
	UseFeatureToggle bool
}

// NewRunnerConfig creates a new IntegationConfig from viper, v can be nil
func newRunnerConfig() *runnerConfig {
	v := viper.GetViper()
	var ic runnerConfig
	v.SetDefault("timeout", 0)
	v.SetDefault("usefeaturetoggle", false)

	v.BindEnv("timeout", "RUNNER_TIMEOUT")
	v.BindEnv("usefeaturetoggle", "RUNNER_USE_FEATURE_TOGGLE")

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
	Run()
	Exiter(int)
}

func isFeatureEnabled(ctx context.Context, runnable string) (bool, error) {
	client, err := NewUnleashClient()
	if err != nil {
		return false, err
	}
	f, err := client.GetFeature(ctx, runnable)
	if err != nil {
		return false, err
	}
	return f.Enabled, nil
}

// ValidationRunner is an implementation of Runner
type ValidationRunner struct {
	Runnable Validation
	Name     string
	Exiter   exitFunc
	config   *runnerConfig
}

// NewValidationRunner creates a ValidationRunner for a given Validation
func NewValidationRunner(runnable Validation, name string) *ValidationRunner {
	c := newRunnerConfig()
	v := &ValidationRunner{
		Runnable: runnable,
		Name:     name,
		config:   c,
	}
	v.Exiter = func(exitCode int) {
		os.Exit(exitCode)
	}
	return v
}

// Run executes the validation configured as target
func (v *ValidationRunner) Run() {
	ctx := context.WithValue(context.Background(), ContextIngetrationNameKey, v.Name)
	var cancel func()
	if v.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(v.config.Timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	if v.config.UseFeatureToggle {
		enabled, err := isFeatureEnabled(ctx, v.Name)
		if err != nil {
			Log().Errorw("Error during integration", "error", err.Error())
			v.Exiter(1)
		}
		if !enabled {
			Log().Warnw("Integration not enabled")
			v.Exiter(0)
		}
	}

	if err := v.Runnable.Setup(ctx); err != nil {
		Log().Errorw("Error during integration", "error", err.Error())
		v.Exiter(1)
	}

	validationErrors, err := v.Runnable.Validate(ctx)
	if err != nil {
		Log().Errorw("Error during integration", "error", err.Error())
		v.Exiter(1)
	}
	if len(validationErrors) > 0 {
		for _, e := range validationErrors {
			Log().Infow("Validation error", "path", e.Path, "validation", e.Validation, "error", e.Error.Error())
		}
		v.Exiter(1)
	}
}
