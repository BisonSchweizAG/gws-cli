package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Google Cloud (simple text-based login)",
	Long: `Authenticate with Google Cloud using a simple text-based flow.
This command prints an authorization URL to the terminal and waits for you
to paste the authorization code. It does not require a browser or TUI,
making it suitable for Termux and other headless environments.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, _ []string) error {
	cfg, err := readConfig()
	if err != nil {
		return err
	}

	_, err = gcloud.LoginSimple(cmd.Context(), cfg)
	return err
}
