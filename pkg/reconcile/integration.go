package reconcile

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type integrationNameKey string

// ContextIngetrationNameKey is the key used to store the integration name in the context
var ContextIngetrationNameKey integrationNameKey = "integrationName"

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

// NewResourceInventory creates a new ResourceInventory
func NewResourceInventory() *ResourceInventory {
	return &ResourceInventory{
		State: map[string]*ResourceState{},
	}
}

// AddResourceState adds a ResourceState to the ResourceInventory
func (ri *ResourceInventory) AddResourceState(target string, rs *ResourceState) {
	ri.State[target] = rs
}

// GetResourceState returns a ResourceState from the ResourceInventory
func (ri *ResourceInventory) GetResourceState(target string) *ResourceState {
	return ri.State[target]
}

// ResourceState describes the state of a resource
type ResourceState struct {
	// Config is the configuration of the resource, usually the GraphQL response object
	Config interface{}
	// Current state of the resource
	Current interface{}
	// Desired state of the resource
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
	Exiter   exitFunc
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
	v.Exiter = func(exitCode int) {
		if v.metrics != nil {
			v.metrics.status.Set(float64(exitCode))
		}
		util.Log().Debugw("Exiting", "exitCode", exitCode)
		os.Exit(exitCode)
	}
	return v
}

func (i *IntegrationRunner) runIntegration() {
	ctx := context.WithValue(context.Background(), ContextIngetrationNameKey, i.Name)
	var cancel func()
	if i.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(i.config.Timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	ri := NewResourceInventory()

	err := i.Runnable.Setup(ctx)
	if err != nil {
		util.Log().Errorw("Error during setup", "error", err.Error())
		i.Exiter(1)
	}

	err = i.Runnable.CurrentState(ctx, ri)
	if err != nil {
		util.Log().Errorw("Error during CurrentState", "error", err.Error())
		i.Exiter(1)
	}
	err = i.Runnable.DesiredState(ctx, ri)
	if err != nil {
		util.Log().Errorw("Error during DesiredState", "error", err.Error())
		i.Exiter(1)
	}
	i.Runnable.LogDiff(ri)
	if !i.config.DryRun {
		err = i.Runnable.Reconcile(ctx, ri)
		if err != nil {
			util.Log().Errorw("Error during Reconcile", "error", err.Error())
			i.Exiter(1)
		}
	} else {
		util.Log().Debugw("DryRun is enabled, not running Reconcile")
	}
	if i.metrics != nil {
		i.metrics.status.Set(float64(0))
	}
}

// Run runs the integration
func (i *IntegrationRunner) Run() {
	go func(i *IntegrationRunner) {
		http.Handle("/metrics", promhttp.HandlerFor(i.registry, promhttp.HandlerOpts{Registry: i.registry}))
		util.Log().Fatal(http.ListenAndServe(fmt.Sprintf(":%d", i.config.PrometheusPort), nil))
	}(i)

	for {
		start := time.Now()
		i.runIntegration()
		end := time.Now()
		i.metrics.time.Set(end.Sub(start).Seconds())
		if i.config.RunOnce {
			i.Exiter(0)
		} else {
			util.Log().Debugw("Sleeping", "seconds", i.config.SleepDurationSecs)
			time.Sleep(time.Duration(i.config.SleepDurationSecs) * time.Second)
		}
	}
}
