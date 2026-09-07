package cmd

import (
	"os"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/workstations/apiv1/workstationspb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/spinner"
)

// statusCmd represents the status command.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "List all configured workstations with their state",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := readConfig()
		if err != nil {
			return err
		}

		output, _ := cmd.Flags().GetString("output")
		wide := output == "wide"

		spinner.Disable()
		log.SetLogger(log.Null)
		states, err := gcloud.GetWorkstationStates(cmd.Context(), cfg)
		log.SetLogger(log.Stdout)
		if err != nil {
			return err
		}

		slices.SortFunc(states, func(a, b gcloud.WorkstationState) int {
			return strings.Compare(a.Context, b.Context)
		})

		t := table.NewWriter()
		t.SetStyle(table.StyleRounded)
		t.SetOutputMirror(os.Stdout)
		if wide {
			t.AppendHeader(table.Row{"CONTEXT", "PROJECT", "CONFIG", "NAME", "STATE", "UPTIME"})
		} else {
			t.AppendHeader(table.Row{"CONTEXT", "NAME", "STATE", "UPTIME"})
		}
		for _, s := range states {
			if wide {
				t.AppendRow(table.Row{s.Context, s.Project, s.Config, s.Name, formatState(s.State), formatUptime(s.Uptime)})
			} else {
				t.AppendRow(table.Row{s.Context, s.Name, formatState(s.State), formatUptime(s.Uptime)})
			}
		}
		t.Render()
		return nil
	},
}

func formatState(s workstationspb.Workstation_State) string {
	state := s.String()
	state = strings.ReplaceAll(state, "Workstation_STATE_", "")
	state = strings.ReplaceAll(state, "STATE_", "")
	switch s {
	case workstationspb.Workstation_STATE_RUNNING:
		return "🟢 " + state
	case workstationspb.Workstation_STATE_STOPPED:
		return "🔴 " + state
	case workstationspb.Workstation_STATE_STARTING, workstationspb.Workstation_STATE_STOPPING:
		return "🟡 " + state
	default:
		return "⚪ " + state
	}
}

func formatUptime(u *time.Duration) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("output", "o", "", "Output format. One of: wide")
}
