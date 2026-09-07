package cmd

import (
	"context"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gateway"
	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/spinner"
	"github.com/bisonschweizag/gws-cli/internal/tui"
)

var (
	flagLocalPort  int
	flagTokenCheck bool
	flagRestart    bool
)

// tunnelCmd represents the tunnel command.
var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "tunnel a workstation",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := readConfig(args...)
		if err != nil {
			return err
		}
		cfg.TokenCheck = flagTokenCheck

		spinner.Disable()
		m := tui.NewModel(cmd.Context(), cfg, ">_ GWS Tunnel", func(ctx context.Context) error {
			if flagRestart {
				if err := gcloud.StopWorkstation(ctx, cfg); err != nil {
					return err
				}
			}

			if err := gateway.UpdateDownloadLocation(ctx, cfg); err != nil {
				return err
			}

			if err := gcloud.StartWorkstation(ctx, cfg, flagBoostConfig); err != nil {
				return err
			}

			log.Log(tui.StopSpinner)
			return gcloud.TCPTunnelWithPassphrase(ctx, cfg, flagLocalPort)
		})

		port := flagLocalPort
		if port == 0 {
			port = cfg.CurrentContext().Port
		}
		m.AddHeader("Local Port", strconv.Itoa(port))

		p := tea.NewProgram(m)
		_, err = p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
	tunnelCmd.PersistentFlags().
		IntVarP(&flagLocalPort, "local-host-port", "p", 0, "The local host port to open (default ist the port from the config)")
	tunnelCmd.PersistentFlags().
		BoolVar(&flagTokenCheck, "check-token", true, "Enable periodic token check")
	tunnelCmd.PersistentFlags().
		BoolVar(&flagRestart, "restart", false, "Restart the workstation before opening the tunnel")
	withBoost(tunnelCmd)
}
