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
	CurrentState(context.Context, *ResourceInventory) error
	DesiredState(context.Context, *ResourceInventory) error
	Reconcile(context.Context, *ResourceInventory) error
	Setup() error
}

type Action string

const (
	Create Action = "create"
	Delete Action = "delete"
	Update Action = "update"
)

// ResourceInventory must be used to describe the diff an integration found
type ResourceInventory struct {
	State []ResourceState
}

type ResourceState struct {
	Action  Action
	Target  string
	Current interface{}
	Desired interface{}
}

// IntegrationRunner is an implementation of Runner
type IntegrationRunner struct {
	Runnable Integration
	Name     string
	config   *runnerConfig
}

func (i *IntegrationRunner) Run() {
	ri := &ResourceInventory{
		State: make([]ResourceState, 0),
	}

	ctx := context.WithValue(context.Background(), ContextIngetrationNameKey, i.Name)
	var cancel func()
	if i.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(i.config.Timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	err := i.Runnable.Setup()
	if err != nil {
		i.Exiter(1)
	}

	err = i.Runnable.CurrentState(ctx, ri)
	if err != nil {
		i.Exiter(1)
	}
	err = i.Runnable.DesiredState(ctx, ri)
	if err != nil {
		i.Exiter(1)
	}
	err = i.Runnable.Reconcile(ctx, ri)
	if err != nil {
		i.Exiter(1)
	}
}

func (i *IntegrationRunner) Exiter(exitCode int) {
	os.Exit(exitCode)
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
