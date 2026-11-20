package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gateway"
	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/sessionend"
	"github.com/bisonschweizag/gws-cli/internal/shutdownlog"
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
			if err := gateway.CheckNoProxy(ctx, cfg); err != nil {
				return err
			}

			if err := gcloud.StartWorkstation(ctx, cfg, flagBoostConfig); err != nil {
				return err
			}

			log.Log(tui.StopSpinner)
			return gcloud.TCPTunnelWithPassphrase(ctx, cfg, flagLocalPort)
		})

		m.BoostConfig = flagBoostConfig
		m.AddHeader("SSH User", m.Styles.SSH.Render(cfg.CurrentContext().User))
		m.AddHeader("SSH Host", m.Styles.SSH.Render(cfg.CurrentContext().Host))
		port := flagLocalPort
		if port == 0 {
			port = cfg.CurrentContext().Port
		}
		m.AddHeader("SSH Port", m.Styles.SSH.Render(strconv.Itoa(port)))

		// Enable shutdown hook toggle (only for tunnel command)
		m.ShutdownHookEnabled = true

		// Listen for SIGHUP (terminal close) so shutdown runs when the terminal is closed.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)
		defer signal.Stop(sigCh)

		sessionEndCh, releaseSessionEnd := sessionend.Listen()
		defer releaseSessionEnd()
		stopTrigger := "tunnel exit"
		var stopTriggerMu sync.Mutex
		logShutdown := func(trigger, result string) {
			workstationName := ""
			if currentContext := cfg.CurrentContext(); currentContext != nil && currentContext.GCloud != nil {
				workstationName = currentContext.GCloud.Name
			}
			message := fmt.Sprintf(
				"context=%q workstation=%q trigger=%q result=%s",
				cfg.CurrentContextName,
				workstationName,
				trigger,
				result,
			)
			_ = shutdownlog.Append(shutdownlog.Dir(cfg), message, time.Now())
		}
		stopOnExit := sync.OnceFunc(func() {
			if currentContext := cfg.CurrentContext(); currentContext == nil || !currentContext.StopWorkstationOnExitTunnel {
				return
			}

			stopTriggerMu.Lock()
			trigger := stopTrigger
			stopTriggerMu.Unlock()
			logShutdown(trigger, "stop requested")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			stopErr := gcloud.StopWorkstationsForContexts(ctx, cfg)
			cancel()

			if stopErr != nil {
				logShutdown(trigger, fmt.Sprintf("error: %q", stopErr.Error()))
			} else {
				logShutdown(trigger, "stop successful")
			}
		})

		p := tea.NewProgram(m)
		done := make(chan struct{})
		go func() {
			trigger := ""
			select {
			case sig := <-sigCh:
				trigger = "signal " + sig.String()
			case <-sessionEndCh:
				trigger = "Windows session end"
			case <-done:
				return
			}
			stopTriggerMu.Lock()
			stopTrigger = trigger
			stopTriggerMu.Unlock()
			stopOnExit()
			releaseSessionEnd()
			p.Send(tea.QuitMsg{})
		}()

		_, err = p.Run()
		close(done)

		// Stop workstations on exit only if shutdown is still active.
		// If the user selected "Don't Stop" in the confirmation prompt, ShutdownHookActive is false.
		if m.ShutdownHookActive {
			stopOnExit()
		}

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
