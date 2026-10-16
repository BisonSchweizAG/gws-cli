package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// dashboardURLTemplate is the Google Cloud workstations overview URL.
const dashboardURLTemplate = "https://console.cloud.google.com/workstations/overview?authuser=0&project=%s"

// dashboardCmd represents the dashboard command.
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the Google Cloud Workstations Dashboard",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := readConfig(args...)
		if err != nil {
			return err
		}

		currentContext := cfg.CurrentContext()
		if currentContext == nil || currentContext.GCloud == nil || currentContext.GCloud.Project == "" {
			return errors.New("no google cloud project configured for the current context")
		}

		openURL(cmd.Context(), cfg, "Dashboard", fmt.Sprintf(dashboardURLTemplate, currentContext.GCloud.Project))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
