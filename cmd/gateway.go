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
		all, _ := cmd.Flags().GetBool("all")
		check, _ := cmd.Flags().GetBool("check")
		return gateway.Run(cmd.Context(), cfg, gateway.Options{All: all, Check: check})
	},
}

func init() {
	rootCmd.AddCommand(gatewayCmd)
	gatewayCmd.Flags().Bool("all", false, "Check or update all installed JetBrains Gateway versions")
	gatewayCmd.Flags().Bool("check", false, "Only check settings without modifying them")
}
