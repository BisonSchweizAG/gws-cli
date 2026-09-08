package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

// logoutCmd represents the logout command.
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove the stored Google access token",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := types.DeleteToken(); err != nil {
			return err
		}
		log.Log("Logged out successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
