package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/shutdowntask"
)

var shutdownTaskCmd = &cobra.Command{
	Use:   "shutdown-task",
	Short: "Manage the Windows shutdown task for stopping workstations",
}

var flagDumpXML bool

var shutdownTaskRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a Windows shutdown task to stop all workstations on shutdown",
	RunE: func(_ *cobra.Command, _ []string) error {
		return shutdowntask.Register(flagDumpXML)
	},
}

var shutdownTaskUnregisterCmd = &cobra.Command{
	Use:   "unregister",
	Short: "Unregister the Windows shutdown task",
	RunE: func(_ *cobra.Command, _ []string) error {
		return shutdowntask.Unregister()
	},
}

var shutdownTaskStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the Windows shutdown task",
	RunE: func(_ *cobra.Command, _ []string) error {
		return shutdowntask.Status()
	},
}

var shutdownTaskDumpXMLCmd = &cobra.Command{
	Use:   "dump-xml",
	Short: "Print the task XML template to stdout",
	RunE: func(_ *cobra.Command, _ []string) error {
		return shutdowntask.DumpXML()
	},
}

func init() {
	rootCmd.AddCommand(shutdownTaskCmd)
	shutdownTaskCmd.AddCommand(shutdownTaskRegisterCmd)
	shutdownTaskRegisterCmd.Flags().BoolVar(&flagDumpXML, "dump-xml", false, "Print the rendered task XML to stderr")
	shutdownTaskCmd.AddCommand(shutdownTaskUnregisterCmd)
	shutdownTaskCmd.AddCommand(shutdownTaskStatusCmd)
	shutdownTaskCmd.AddCommand(shutdownTaskDumpXMLCmd)
}
