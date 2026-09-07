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

// restartCmd represents the restart command.
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart a workstation",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := readConfig(args...)
		if err != nil {
			return err
		}

		spinner.Disable()
		m := tui.NewModel(cmd.Context(), cfg, ">_ GWS Restart", func(ctx context.Context) error {
			if err := gcloud.StopWorkstation(ctx, cfg); err != nil {
				return err
			}
			return gcloud.StartWorkstation(ctx, cfg, flagBoostConfig)
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
	rootCmd.AddCommand(restartCmd)
	withBoost(restartCmd)
}
