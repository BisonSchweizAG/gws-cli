package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/types"
	"github.com/bisonschweizag/gws-cli/version"
)

// rootCmd represents the base command when called without any subcommands.
var (
	rootCmd = &cobra.Command{
		Use:     "gws",
		Short:   "Google Cloud Workstation Utils",
		Version: version.Version,
	}
	flagConfig  string
	flagContext string
)

func Execute() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	defer func() { _ = gcloud.CloseClient() }()
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagContext, "ctx", "", "The context to be used")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", types.ConfigFileName, "The config file to be used")
}

func readConfig(args ...string) (*types.Config, error) {
	if flagContext == "" && len(args) == 1 {
		flagContext = args[0]
	}
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	if flagContext != "" {
		err = config.SwitchContext(flagContext, false)
	}

	return config, err
}

func loadConfig() (*types.Config, error) {
	config := &types.Config{Contexts: make(map[string]*types.Context)}
	err := config.Load(flagConfig)
	return config, err
}
