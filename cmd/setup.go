package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/setup"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Create a new or update the config.yaml and create a context configuration",
	Long: `Create a new or update the config.yaml and create a context configuration using an
interactive terminal setup wizard.`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return setup.Run(cfg, flagContext)
}
