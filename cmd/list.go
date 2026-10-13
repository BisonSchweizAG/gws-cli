package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/spinner"
)

// listCmd represents the list command.
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all cloud workstations in the current context",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := readConfig()
		if err != nil {
			return err
		}

		currentCtx := cfg.CurrentContext()
		if currentCtx == nil || currentCtx.GCloud == nil {
			return errors.New("no current context selected with gcloud configuration")
		}

		project := currentCtx.GCloud.Project
		region := currentCtx.GCloud.Region
		cluster := currentCtx.GCloud.Cluster

		output, _ := cmd.Flags().GetString("output")
		wide := output == "wide"
		sortBy, _ := cmd.Flags().GetString("sort")
		filter, _ := cmd.Flags().GetString("filter")

		validSorts := []string{"name", "user", "project", "config", "state", "last-started"}
		if !slices.Contains(validSorts, sortBy) {
			return fmt.Errorf("invalid sort option %q, must be one of: %s", sortBy, strings.Join(validSorts, ", "))
		}

		spinner.Disable()
		log.SetLogger(log.Null)
		workstations, err := gcloud.ListProjectWorkstations(cmd.Context(), cfg, project, region, cluster)
		log.SetLogger(log.Stdout)
		if err != nil {
			return err
		}

		// Filter by name and user (case-insensitive)
		if filter != "" {
			filterLower := strings.ToLower(filter)
			filtered := make([]gcloud.ProjectWorkstation, 0, len(workstations))
			for _, ws := range workstations {
				nameMatch := strings.Contains(strings.ToLower(ws.Name), filterLower)
				userMatch := strings.Contains(strings.ToLower(ws.User), filterLower)
				if nameMatch || userMatch {
					filtered = append(filtered, ws)
				}
			}
			workstations = filtered
		}

		slices.SortFunc(workstations, func(a, b gcloud.ProjectWorkstation) int {
			switch sortBy {
			case "name":
				return strings.Compare(a.Name, b.Name)
			case "user":
				return strings.Compare(a.User, b.User)
			case "project":
				return strings.Compare(a.Project, b.Project)
			case "config":
				return strings.Compare(a.Config, b.Config)
			case "state":
				return int(a.State) - int(b.State)
			default:
				if cmp := compareDate(a.LastStarted, b.LastStarted); cmp != 0 {
					return cmp
				}
				return strings.Compare(a.Name, b.Name)
			}
		})

		var header table.Row
		if wide {
			header = table.Row{
				"PROJECT",
				"CONFIG",
				"NAME",
				"USER",
				"STATE",
				"LAST STARTED",
			}
		} else {
			header = table.Row{"NAME", "USER", "STATE", "LAST STARTED"}
		}

		rows := make([]table.Row, 0, len(workstations))
		for _, ws := range workstations {
			if wide {
				rows = append(rows, table.Row{
					ws.Project,
					ws.Config,
					ws.Name,
					ws.User,
					formatState(ws.State),
					formatDateTime(ws.LastStarted),
				})
			} else {
				rows = append(rows, table.Row{
					ws.Name,
					ws.User,
					formatState(ws.State),
					formatDate(ws.LastStarted),
				})
			}
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

func formatDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	//nolint:gosmopolitan // Display date in the user's local timezone
	return t.Local().Format("02.01.2006")
}

func compareDate(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	// Truncate to date only (midnight in local timezone)
	//nolint:gosmopolitan // Display date in the user's local timezone
	aLocal := a.Local()
	//nolint:gosmopolitan // Display date in the user's local timezone
	bLocal := b.Local()
	aDate := time.Date(aLocal.Year(), aLocal.Month(), aLocal.Day(), 0, 0, 0, 0, aLocal.Location())
	bDate := time.Date(bLocal.Year(), bLocal.Month(), bLocal.Day(), 0, 0, 0, 0, bLocal.Location())
	if aDate.Equal(bDate) {
		return 0
	}
	// Newest first: reverse the comparison
	if aDate.Before(bDate) {
		return 1
	}
	return -1
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("output", "o", "", "Output format. One of: wide")
	listCmd.Flags().StringP("sort", "s", "last-started", "Sort by column: name, user, project, config, state, last-started")
	listCmd.Flags().StringP("filter", "f", "", "Filter workstations by name or user (case-insensitive)")
}
