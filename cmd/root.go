// Package cmd contains the main entrypoint for the go-qontract-reconcile binary.
package cmd

import (
	"time"

	defaultlog "log"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile  string
	logLevel string

	rootCmd = &cobra.Command{
		Use:   "qo-contract-reconcile",
		Short: "qo-contract-reconcile",
		Long:  "CLI for integrations written in Golang",
	}

	userValidatorCmd = &cobra.Command{
		Use:   "user-validator",
		Short: "Validate users",
		Long:  "Run validations for PGP keys, usernames and github logins",
		Run: func(cmd *cobra.Command, args []string) {
			userValidator()
		},
	}

	accountNotifierCmd = &cobra.Command{
		Use:   "account-notifier",
		Short: "Sent out notifications on new passwords",
		Long:  "Send PGP encrypted passwords to users",
		Run: func(cmd *cobra.Command, args []string) {
			accountNotifier()
		},
	}

	gitPartitionSyncProducerCmd = &cobra.Command{
		Use:   "git-partition-sync-producer",
		Short: "Produce messages for git partition sync",
		Long:  "Produce messages for git partition sync",
		Run: func(cmd *cobra.Command, args []string) {
			gitPartitionSyncProducer()
		},
	}

	advancedClusterSecurityRbacCmd = &cobra.Command{
		Use:   "advanced-cluster-security-rbac",
		Short: "Reconciles acs rbac resources",
		Long:  "Reconciles resources for Red Hat Advanced Cluster Security RBAC",
		Run: func(cmd *cobra.Command, args []string) {
			advancedClusterSecurityRbac()
		},
	}
)

// Execute executes the rootCmd
func Execute() {
	rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(userValidatorCmd)
	rootCmd.AddCommand(accountNotifierCmd)
	rootCmd.AddCommand(gitPartitionSyncProducerCmd)
	rootCmd.AddCommand(advancedClusterSecurityRbacCmd)
	rootCmd.PersistentFlags().StringVarP(&logLevel, "logLevel", "l", "info", "Log level")
	userValidatorCmd.Flags().StringVarP(&cfgFile, "cfgFile", "c", "", "Configuration File")
	accountNotifierCmd.Flags().StringVarP(&cfgFile, "cfgFile", "c", "", "Configuration File")
	gitPartitionSyncProducerCmd.Flags().StringVarP(&cfgFile, "cfgFile", "c", "", "Configuration File")
	advancedClusterSecurityRbacCmd.Flags().StringVarP(&cfgFile, "cfgFile", "c", "", "Configuration File")

	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(configureLogging)
}

func initConfig() {
	viper.SetConfigFile(cfgFile)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		util.Log().Debugw("Using configuration", "config", cfgFile)
	}
}

func configureLogging() {
	loggerConfig := zap.NewDevelopmentConfig()

	switch logLevel {
	case "info":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "debug":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "error":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "warn":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "fatal":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	case "panic":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.PanicLevel)
	default:
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)

	logger, err := loggerConfig.Build()
	zap.ReplaceGlobals(logger)

	if err != nil {
		defaultlog.Fatal(err)
	}
}
