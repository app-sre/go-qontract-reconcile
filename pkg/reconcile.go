package pkg

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

type IntegrationNameKey string

var ContextIngetrationNameKey IntegrationNameKey = "integrationName"

// RunnerConfig is used to unmarshal yaml configuration Runners
type runnerConfig struct {
	Timeout           int
	UseFeatureToggle  bool
	DryRun            bool
	RunOnce           bool
	SleepDurationSecs int
}

// NewRunnerConfig creates a new IntegationConfig from viper, v can be nil
func newRunnerConfig() *runnerConfig {
	v := viper.GetViper()
	var ic runnerConfig
	v.SetDefault("timeout", 0)
	v.SetDefault("usefeaturetoggle", false)
	v.SetDefault("dryrun", true)
	v.SetDefault("runonce", false)
	v.SetDefault("sleepdurationsecs", 600)

	v.BindEnv("timeout", "RUNNER_TIMEOUT")
	v.BindEnv("usefeaturetoggle", "RUNNER_USE_FEATURE_TOGGLE")
	v.BindEnv("dryrun", "DRY_RUN")
	v.BindEnv("runonce", "RUN_ONCE")
	v.BindEnv("sleepdurationsecs", "SLEEP_DURATION_SECS")

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
	LogDiff(*ResourceInventory)
	Setup(context.Context) error
}

// ResourceInventory must be used to describe the diff an integration found
type ResourceInventory struct {
	State map[string]*ResourceState
}

func NewResourceInventory() *ResourceInventory {
	return &ResourceInventory{
		State: map[string]*ResourceState{},
	}
}

func (ri *ResourceInventory) AddResourceState(target string, rs *ResourceState) {
	ri.State[target] = rs
}

func (ri *ResourceInventory) GetResourceState(target string) *ResourceState {
	return ri.State[target]
}

type ResourceState struct {
	Current interface{}
	Desired interface{}
}

type integrationRunnerMetrics struct {
	status prometheus.Gauge
	time   prometheus.Gauge
}

func newIntegrationRunnerMetrics(reg prometheus.Registerer, integration string) *integrationRunnerMetrics {
	labels := prometheus.Labels{"integration": integration}

	m := &integrationRunnerMetrics{
		status: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "qontract_reconcile_last_run_status",
			Help:        "Last run status",
			ConstLabels: labels,
		}),
		time: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "qontract_reconcile_last_run_seconds",
			Help:        "Last run duration in seconds",
			ConstLabels: labels,
		}),
	}
	reg.MustRegister(m.status)
	reg.MustRegister(m.time)
	return m
}

// IntegrationRunner is an implementation of Runner
type IntegrationRunner struct {
	Runnable Integration
	Name     string
	config   *runnerConfig
	metrics  *integrationRunnerMetrics
	registry *prometheus.Registry
}

// NewIntegrationRunner creates a IntegrationRunner for a given Integration
func NewIntegrationRunner(runnable Integration, name string) *IntegrationRunner {
	c := newRunnerConfig()
	registry := prometheus.NewRegistry()
	v := &IntegrationRunner{
		Runnable: runnable,
		Name:     name,
		config:   c,
		registry: registry,
		metrics:  newIntegrationRunnerMetrics(registry, name),
	}
	return v
}

func (i *IntegrationRunner) runIntegration() {
	ri := NewResourceInventory()

	ctx := context.WithValue(context.Background(), ContextIngetrationNameKey, i.Name)
	var cancel func()
	if i.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(i.config.Timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	err := i.Runnable.Setup(ctx)
	if err != nil {
		Log().Errorw("Error during setup", "error", err.Error())
		i.Exiter(1)
	}

	err = i.Runnable.CurrentState(ctx, ri)
	if err != nil {
		Log().Errorw("Error during CurrentState", "error", err.Error())
		i.Exiter(1)
	}
	err = i.Runnable.DesiredState(ctx, ri)
	if err != nil {
		Log().Errorw("Error during DesiredState", "error", err.Error())
		i.Exiter(1)
	}
	i.Runnable.LogDiff(ri)
	if !i.config.DryRun {
		err = i.Runnable.Reconcile(ctx, ri)
		if err != nil {
			Log().Errorw("Error during Reconcile", "error", err.Error())
			i.Exiter(1)
		}
	} else {
		Log().Infow("DryRun is enabled, not running Reconcile")
	}
}

func (i *IntegrationRunner) Run() {
	go func(i *IntegrationRunner) {
		http.Handle("/metrics", promhttp.HandlerFor(i.registry, promhttp.HandlerOpts{Registry: i.registry}))
		Log().Fatal(http.ListenAndServe(":9090", nil))
	}(i)

	for {
		start := time.Now()
		i.runIntegration()
		end := time.Now()
		i.metrics.time.Set(end.Sub(start).Seconds())
		Log().Debugw("Sleeping", "seconds", i.config.SleepDurationSecs)
		time.Sleep(time.Duration(i.config.SleepDurationSecs) * time.Second)
		if i.config.RunOnce {
			i.Exiter(0)
		}
	}
}

func (i *IntegrationRunner) Exiter(exitCode int) {
	i.metrics.status.Set(float64(exitCode))
	if i.config.RunOnce {
		os.Exit(exitCode)
	} else {
		Log().Debugw("RunOnce is disabled, not exiting", "exitCode", exitCode)
	}
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
