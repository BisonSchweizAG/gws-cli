package cmd

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/spinner"
	"github.com/bisonschweizag/gws-cli/internal/tui"
)

var (
	flagStopAll bool

	// stopCmd represents the stop command.
	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop a workstation",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(args...)
			if err != nil {
				return err
			}

			spinner.Disable()
			m := tui.NewModel(cmd.Context(), cfg, ">_ GWS Stop", func(ctx context.Context) error {
				if flagStopAll {
					return gcloud.StopAllWorkstations(ctx, cfg)
				}
				return gcloud.StopWorkstation(ctx, cfg)
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
)

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.PersistentFlags().
		BoolVar(&flagStopAll, "all", false, "Stop all workstations")
}
