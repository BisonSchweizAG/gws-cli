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
	flagOutput          string
	flagSaveToDefault   bool
	flagUpdateSSHConfig bool
	flagForwardAgent    bool
	flagSSHConfigName   string
)

func init() {
	rootCmd.AddCommand(scriptsCmd)
	scriptsCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "Output file (default stdout)")
	scriptsCmd.PersistentFlags().
		BoolVarP(&flagSaveToDefault, "save", "s", false, "Save the script to the default config location")
	scriptsCmd.AddCommand(scriptsWinReconnectCmd)
	scriptsCmd.AddCommand(scriptsUnixReconnectCmd)
	scriptsSSHConfigCmd.Flags().
		BoolVarP(&flagUpdateSSHConfig, "update-ssh-config", "u", false,
			"Add the \""+script.SSHConfigIncludeDirective+"\" directive to ~/.ssh/config if it is missing")
	scriptsSSHConfigCmd.Flags().
		BoolVarP(&flagForwardAgent, "forward-agent", "a", false,
			"Add the \"ForwardAgent yes\" directive to the generated ssh config")
	scriptsSSHConfigCmd.Flags().
		StringVarP(&flagSSHConfigName, "name", "n", "",
			"Name of the generated ssh config host (default: context name)")
	scriptsCmd.AddCommand(scriptsSSHConfigCmd)
}

func runReconnectScript(
	cmd *cobra.Command,
	args []string,
	generate func(*types.Config, string) (string, []byte, error),
	logMsg string,
	perm os.FileMode,
	saveDir func() (string, error),
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
		dir, err := saveDir()
		if err != nil {
			return err
		}
		outputPath = filepath.Join(dir, name)
	}

	if outputPath != "" {
		cmd.Printf(logMsg, cfg.CurrentContextName, outputPath)
		return os.WriteFile(outputPath, s, perm)
	}
	_, _ = cmd.OutOrStdout().Write(s)

	return nil
}

// handleSSHConfigInclude warns when ~/.ssh/config does not include the
// config.d directory. The main ssh config is only modified when the
// --update-ssh-config flag is set.
func handleSSHConfigInclude(cmd *cobra.Command) error {
	if flagUpdateSSHConfig {
		sshConfigPath, added, err := script.AddSSHConfigInclude()
		if err != nil {
			return err
		}
		if added {
			cmd.Printf("🔧 Added %q to %s\n", script.SSHConfigIncludeDirective, sshConfigPath)
		}

		return nil
	}

	sshConfigPath, present, err := script.HasSSHConfigInclude()
	if err != nil {
		return err
	}
	if !present {
		cmd.PrintErrf(
			"⚠️  %s does not include %q, generated snippets are ignored by ssh. Run with --update-ssh-config to add it.\n",
			sshConfigPath,
			script.SSHConfigIncludeDirective,
		)
	}

	return nil
}

func defaultConfigDir() (string, error) {
	_, configDir, _ := types.DefaultConfigPaths()
	return configDir, nil
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
			"💾 Writing SSH reconnect script for context %q on Windows to %s\n",
			0o644,
			defaultConfigDir,
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
			"💾 Writing SSH bash reconnect script for context %q to %s\n",
			0o755,
			defaultConfigDir,
		)
	},
}

var scriptsSSHConfigCmd = &cobra.Command{
	Use:   "ssh-config",
	Short: "Generate SSH config for the current workstation",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := handleSSHConfigInclude(cmd); err != nil {
			return err
		}

		return runReconnectScript(
			cmd,
			args,
			script.SSHConfigWithOptions(flagForwardAgent, flagSSHConfigName),
			"💾 Writing SSH config for context %q to %s\n",
			0o600,
			script.SSHConfigDir,
		)
	},
}
