package cmd

import (
	"time"

	defaultlog "log"

	. "github.com/app-sre/user-validator/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "user-validator",
		Short: "user-validator integration",
		Long:  "Integration, that verifies the content of users in app-interface",
	}

	userValidatorCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate users",
		Long:  "Run validations for pgp keys, usernames and github logins",
		Run: func(cmd *cobra.Command, args []string) {
			userValidator()
		},
	}
)

// Execute executes the rootCmd
func Execute() {
	rootCmd.Execute()
}

func init() {
	configureLogging()
	rootCmd.AddCommand(userValidatorCmd)

	cobra.OnInitialize(initConfig)
	userValidatorCmd.Flags().StringVarP(&cfgFile, "cfgFile", "c", "", "Configuration File")
	userValidatorCmd.PersistentFlags().Bool("dry_run", false, "Dry run, skip actuall changes")
	viper.BindPFlag("dry_run", userValidatorCmd.PersistentFlags().Lookup("dry_run"))
}

func initConfig() {
	viper.SetConfigFile(cfgFile)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		Log().Infow("Using configuration", "config", cfgFile)
	}
}

func configureLogging() {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)

	logger, err := loggerConfig.Build()
	zap.ReplaceGlobals(logger)

	if err != nil {
		defaultlog.Fatal(err)
	}
}
