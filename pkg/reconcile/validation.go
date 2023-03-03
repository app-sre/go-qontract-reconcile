package reconcile

import (
	"context"
	"os"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"
)

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
			util.Log().Errorw("Error during integration", "error", err.Error())
			v.Exiter(1)
		}
		if !enabled {
			util.Log().Warnw("Integration not enabled")
			v.Exiter(0)
		}
	}

	if err := v.Runnable.Setup(ctx); err != nil {
		util.Log().Errorw("Error during integration", "error", err.Error())
		v.Exiter(1)
	}

	validationErrors, err := v.Runnable.Validate(ctx)
	if err != nil {
		util.Log().Errorw("Error during integration", "error", err.Error())
		v.Exiter(1)
	}
	if len(validationErrors) > 0 {
		for _, e := range validationErrors {
			util.Log().Infow("Validation error", "path", e.Path, "validation", e.Validation, "error", e.Error.Error())
		}
		v.Exiter(1)
	}
}

// ConcatValidationErrors can be used to merge two list of ValiudationErros
func ConcatValidationErrors(a, b []ValidationError) []ValidationError {
	allErrors := make([]ValidationError, len(a)+len(b))
	copy(allErrors, a)
	for i, e := range b {
		allErrors[len(a)+i] = e
	}
	return allErrors
}
