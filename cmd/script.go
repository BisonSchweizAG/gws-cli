package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/script"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

// scriptsCmd represents the generate scripts command.
var (
	scriptsCmd = &cobra.Command{
		Use:   "scripts",
		Short: "Generate complementary scripts",
	}
	flagOutput        string
	flagSaveToDefault bool
)

func init() {
	rootCmd.AddCommand(scriptsCmd)
	scriptsCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "Output file (default stdout)")
	scriptsCmd.PersistentFlags().
		BoolVarP(&flagSaveToDefault, "save", "s", false, "Save the script to the default config location")
	scriptsCmd.AddCommand(scriptsWinReconnectCmd)
	scriptsCmd.AddCommand(scriptsUnixReconnectCmd)
}

func runReconnectScript(
	cmd *cobra.Command,
	args []string,
	generate func(*types.Config, string) (string, []byte, error),
	logMsg string,
	perm os.FileMode,
) error {
	if flagOutput == "" || flagSaveToDefault {
		log.SetLogger(log.Null)
	}
	cfg, err := readConfig(args...)
	if err != nil {
		return err
	}

	name, s, err := generate(cfg, flagOutput)
	if err != nil {
		return err
	}

	outputPath := flagOutput
	if flagSaveToDefault {
		_, configDir, _ := types.DefaultConfigPaths()
		outputPath = filepath.Join(configDir, name)
	}

	if outputPath != "" {
		cmd.Printf(logMsg, cfg.CurrentContextName, outputPath)
		return os.WriteFile(outputPath, s, perm)
	}
	_, _ = cmd.OutOrStdout().Write(s)

	return nil
}

// scriptsWinReconnectCmd represents the start command.
var scriptsWinReconnectCmd = &cobra.Command{
	Use:   "win-reconnect-ssh",
	Short: "Generate Windows ssh reconnect script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReconnectScript(
			cmd,
			args,
			script.WindowsReconnectSSH,
			"💾 Writing SSH reconnect script for context %q on Windows to %q\n",
			0o644,
		)
	},
}

// scriptsUnixReconnectCmd represents the start command.
var scriptsUnixReconnectCmd = &cobra.Command{
	Use:   "bash-reconnect-ssh",
	Short: "Generate bash ssh reconnect script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReconnectScript(
			cmd,
			args,
			script.UnixReconnectSSH,
			"💾 Writing SSH bash reconnect script for context %q to %q\n",
			0o755,
		)
	},
}
