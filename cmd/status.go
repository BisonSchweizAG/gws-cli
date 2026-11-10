package cmd

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/workstations/apiv1/workstationspb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/spinner"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

// statusCmd represents the status command.
var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "List all configured workstations with their state",
	Aliases: []string{"st"},
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := readConfig()
		if err != nil {
			return err
		}

		output, _ := cmd.Flags().GetString("output")
		detail := output == "detail" || output == "transpose"
		if output == "transpose" {
			cmd.PrintErrln(`Warning: output format "transpose" is deprecated; use "detail" instead.`)
		}
		wide := output == "wide" || detail

		spinner.Disable()
		log.SetLogger(log.Null)
		states, err := gcloud.GetWorkstationStates(cmd.Context(), cfg, flagContext)
		log.SetLogger(log.Stdout)
		if err != nil {
			return err
		}

		slices.SortFunc(states, func(a, b gcloud.WorkstationState) int {
			return strings.Compare(a.Context, b.Context)
		})

		var header table.Row
		if wide {
			header = table.Row{
				"CONTEXT",
				"PROJECT",
				"CONFIG",
				"BOOST",
				"NAME",
				"STATE",
				"UPTIME",
				"SHUTDOWN",
				"RUNNING T/O",
				"IDLE T/O",
				"LAST UPDATED",
			}
			if detail {
				header = slices.Insert(header, machineTypeColumn, any("MACHINE TYPE"))
			}
		} else {
			header = table.Row{"CONTEXT", "NAME", "STATE", "UPTIME", "SHUTDOWN"}
		}
		rows := make([]table.Row, 0, len(states))
		for _, s := range states {
			contextName := formatContext(s.Context, cfg.CurrentContextName, cfg.Contexts)
			shutdown := formatTime(s.ExpectedShutdown)
			if s.RunningTimeout == nil || *s.RunningTimeout == 0 {
				shutdown = ""
			}
			st := workstationState{state: s.State, notFound: s.NotFound}
			if wide {
				row := table.Row{
					contextName,
					s.Project,
					s.Config,
					formatBoostConfigs(s.BoostConfigs, s.ActiveBoostConfig),
					formatName(s.Name, s.ActiveBoostConfig != ""),
					formatState(st),
					formatUptime(s.Uptime),
					shutdown,
					formatUptime(s.RunningTimeout),
					formatUptime(s.IdleTimeout),
					formatDateTime(s.LastUpdated),
				}
				if detail {
					row = slices.Insert(row, machineTypeColumn, any(s.MachineType))
				}
				rows = append(rows, row)
			} else {
				rows = append(
					rows,
					table.Row{
						contextName,
						formatName(s.Name, s.ActiveBoostConfig != ""),
						formatState(st),
						formatUptime(s.Uptime),
						shutdown,
					},
				)
			}
		}

		if detail {
			contextNames := make([]string, len(states))
			for i, s := range states {
				contextNames[i] = formatContext(s.Context, cfg.CurrentContextName, cfg.Contexts)
			}
			// Drop the CONTEXT column since it becomes the header in transposed mode.
			noContextHeader := header[1:]
			noContextRows := make([]table.Row, len(rows))
			for i, r := range rows {
				if len(r) > 0 {
					noContextRows[i] = r[1:]
				} else {
					noContextRows[i] = r
				}
			}
			header, rows = transposeTable(noContextHeader, noContextRows, contextNames)
		}

		t := table.NewWriter()
		t.SetStyle(table.StyleRounded)
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(header)
		for _, row := range rows {
			t.AppendRow(row)
		}
		t.Render()
		return nil
	},
}

// machineTypeColumn is the position of the MACHINE TYPE column (after BOOST),
// which is only shown in detail output.
const machineTypeColumn = 4

func formatName(name string, hasActiveBoost bool) string {
	if hasActiveBoost {
		return name + " 🚀"
	}
	return name
}

func formatBoostConfigs(configs []string, active string) string {
	formatted := make([]string, len(configs))
	for i, config := range configs {
		if config == active && active != "" {
			config = "🟢 " + config
		}
		formatted[i] = config
	}
	return strings.Join(formatted, " | ")
}

// transposeTable swaps rows and columns: the header becomes the first
// column, and each original row becomes a column. The given names are
// used as the new header labels (e.g. context names), falling back to
// a 1-based index if not provided.
func transposeTable(header table.Row, rows []table.Row, names []string) (table.Row, []table.Row) {
	newHeader := table.Row{""}
	for i := range rows {
		if i < len(names) && names[i] != "" {
			newHeader = append(newHeader, names[i])
		} else {
			newHeader = append(newHeader, strconv.Itoa(i+1))
		}
	}

	newRows := make([]table.Row, len(header))
	for i, h := range header {
		row := table.Row{h}
		for _, r := range rows {
			if i < len(r) {
				row = append(row, r[i])
			} else {
				row = append(row, "")
			}
		}
		newRows[i] = row
	}
	return newHeader, newRows
}

func formatContext(name, current string, contexts map[string]*types.Context) string {
	var sb strings.Builder
	if ctx, ok := contexts[name]; ok {
		sb.WriteString(ctx.FormatName(name))
	} else {
		sb.WriteString(name)
	}
	if current != "" && name == current {
		sb.WriteString(" ✅")
	}
	return sb.String()
}

func formatState(s workstationState) string {
	if s.notFound {
		return "❌ NOT_FOUND"
	}
	state := s.state.String()
	state = strings.ReplaceAll(state, "Workstation_STATE_", "")
	state = strings.ReplaceAll(state, "STATE_", "")
	switch s.state {
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

type workstationState struct {
	state    workstationspb.Workstation_State
	notFound bool
}

func formatUptime(u *time.Duration) string {
	if u == nil || *u == 0 {
		return ""
	}
	d := *u
	var sb strings.Builder
	days := d / (24 * time.Hour)
	d %= 24 * time.Hour
	hours := d / time.Hour
	d %= time.Hour
	minutes := d / time.Minute
	d %= time.Minute
	seconds := d / time.Second

	if days > 0 {
		fmt.Fprintf(&sb, "%dd", days)
	}
	if hours > 0 {
		fmt.Fprintf(&sb, "%dh", hours)
	}
	if minutes > 0 {
		fmt.Fprintf(&sb, "%dm", minutes)
	}
	if seconds > 0 {
		fmt.Fprintf(&sb, "%ds", seconds)
	}
	return sb.String()
}

func formatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	//nolint:gosmopolitan // Display time in the user's local timezone
	return t.Local().Format(time.TimeOnly)
}

func formatDateTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	//nolint:gosmopolitan // Display time in the user's local timezone
	return t.Local().Format("02.01.2006 15:04:05")
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("output", "o", "", "Output format. One of: wide, detail")
}
