package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gateway"
)

// gatewayCmd represents the jetbrains-gateway command.
var gatewayCmd = &cobra.Command{
	Use:   "jetbrains-gateway",
	Short: "Apply the JetBrains Gateway settings from the config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := readConfig(args...)
		if err != nil {
			return err
		}
		return gateway.Run(cmd.Context(), cfg)
	},
}

func init() {
	rootCmd.AddCommand(gatewayCmd)
}
