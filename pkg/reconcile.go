package pkg

import (
	"github.com/spf13/viper"
)

// RunnerConfig is used to unmarshal yaml configuration Runners
type RunnerConfig struct {
	DryRun         bool
	QontractConfig *QontractConfig
	VaultConfig    *VaultConfig
}

// NewRunnerConfig creates a new IntegationConfig from viper, v can be nil
func NewRunnerConfig(v *viper.Viper) *RunnerConfig {
	if v == nil {
		v = viper.GetViper()
	}
	ic := RunnerConfig{
		DryRun: v.GetBool("dry_run"),
	}

	if ic.DryRun {
		Log().Debugw("dry_run is enabled")
	}

	ic.QontractConfig = NewQontractConfig(v)
	ic.VaultConfig = NewVaultConfig(v)

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
	Setup() error
	// Validate is doing the actual validation
	Validate() ([]ValidationError, error)
}

// ValidationError contains errors, that are discovered during Validate()
type ValidationError struct {
	Path       string
	Validation string
	Error      error
}

// Runner can be used to actually run Validations or Integrations
type Runner interface {
	Run() error
}

// ValidationRunner is an implementation of Runner
type ValidationRunner struct {
	Target Validation
}

// NewValidationRunner creates a ValidationRunner for a given Validation
func NewValidationRunner(target Validation) *ValidationRunner {
	return &ValidationRunner{
		Target: target,
	}
}

// Run executes the validation configured as target
func (v *ValidationRunner) Run() error {
	if err := v.Target.Setup(); err != nil {
		return err
	}
	validationErrors, err := v.Target.Validate()
	if err != nil {
		return err
	}
	if len(validationErrors) > 0 {
		for _, e := range validationErrors {
			Log().Infow("Validation error", "path", e.Path, "validation", e.Validation, "error", e.Error.Error())
		}
	}
	return nil
}
