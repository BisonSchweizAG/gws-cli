package cmd

import (
	"errors"
	"os"
	"slices"
	"strings"

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

		spinner.Disable()
		log.SetLogger(log.Null)
		workstations, err := gcloud.ListProjectWorkstations(cmd.Context(), cfg, project, region, cluster)
		log.SetLogger(log.Stdout)
		if err != nil {
			return err
		}

		slices.SortFunc(workstations, func(a, b gcloud.ProjectWorkstation) int {
			if a.Cluster != b.Cluster {
				return strings.Compare(a.Cluster, b.Cluster)
			}
			if a.Config != b.Config {
				return strings.Compare(a.Config, b.Config)
			}
			return strings.Compare(a.Name, b.Name)
		})

		var header table.Row
		if wide {
			header = table.Row{
				"CLUSTER",
				"CONFIG",
				"NAME",
				"STATE",
				"LAST STARTED",
				"LAST UPDATED",
			}
		} else {
			header = table.Row{"CLUSTER", "CONFIG", "NAME", "STATE", "LAST UPDATED"}
		}

		rows := make([]table.Row, 0, len(workstations))
		for _, ws := range workstations {
			if wide {
				rows = append(rows, table.Row{
					ws.Cluster,
					ws.Config,
					ws.Name,
					formatState(ws.State),
					formatDateTime(ws.LastStarted),
					formatDateTime(ws.LastUpdated),
				})
			} else {
				rows = append(rows, table.Row{
					ws.Cluster,
					ws.Config,
					ws.Name,
					formatState(ws.State),
					formatDateTime(ws.LastUpdated),
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

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("output", "o", "", "Output format. One of: wide")
}
