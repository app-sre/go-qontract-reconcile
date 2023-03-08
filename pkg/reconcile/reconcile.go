// Package reconcile contains code to run Integrations and Validations
package reconcile

import (
	"context"

	"github.com/app-sre/go-qontract-reconcile/pkg/unleash"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/spf13/viper"
)

type exitFunc func(int)

// Runner can be used to actually run Validations or Integrations
type Runner interface {
	Run()
	Exiter(int)
}

// RunnerConfig is used to unmarshal yaml configuration Runners
type runnerConfig struct {
	Timeout           int
	UseFeatureToggle  bool
	DryRun            bool
	RunOnce           bool
	SleepDurationSecs int
}

// newRunnerConfig creates a new IntegationConfig from viper, v can be nil
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
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}

	return &ic
}

func isFeatureEnabled(ctx context.Context, runnable string) (bool, error) {
	client, err := unleash.NewUnleashClient()
	if err != nil {
		return false, err
	}
	f, err := client.GetFeature(ctx, runnable)
	if err != nil {
		return false, err
	}
	return f.Enabled, nil
}
