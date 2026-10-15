package cmd

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"cloud.google.com/go/workstations/apiv1/workstationspb"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/spinner"
	"github.com/bisonschweizag/gws-cli/internal/tui"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

// launchPort is the port to prepend to the workstation URL (default 80).
var launchPort int

// openWorkstationURL fetches the workstation URL, logs it, and opens it in the browser.
func openWorkstationURL(ctx context.Context, cfg *types.Config) error {
	url, err := gcloud.GetWorkstationURL(ctx, cfg)
	if err != nil {
		return err
	}

	url = fmt.Sprintf("https://%d-%s", launchPort, url)

	log.Logf("Opening workstation URL: %s", url)
	gcloud.OpenBrowser(ctx, cfg, url)
	log.Log("Workstation URL opened in browser")
	return nil
}

// launchCmd represents the launch command.
var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Open the default URL of the cloud workstation",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := readConfig(args...)
		if err != nil {
			return err
		}

		ctx := cmd.Context()

		// Check if the workstation is already running
		state, err := gcloud.GetWorkstationState(ctx, cfg)
		if err != nil {
			return err
		}

		if state == workstationspb.Workstation_STATE_RUNNING {
			// Already running — skip TUI, just open the URL
			return openWorkstationURL(ctx, cfg)
		}

		// Not running — start via TUI, then open URL
		spinner.Disable()
		m := tui.NewModel(ctx, cfg, ">_ GWS Launch", func(ctx context.Context) error {
			if err := gcloud.StartWorkstation(ctx, cfg, ""); err != nil {
				return err
			}
			return openWorkstationURL(ctx, cfg)
		})
		m.AutoQuit = true

		p := tea.NewProgram(m)
		resModel, err := p.Run()
		if err == nil {
			log.SetLogger(log.Stdout)
			if tm, ok := resModel.(*tui.Model); ok {
				log.Log(tm.LastLog())
			}
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)
	launchCmd.Flags().IntVarP(&launchPort, "port", "p", 80, "The workstation port to use")
}
