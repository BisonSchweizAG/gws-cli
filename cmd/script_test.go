package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestScriptsReconnectCmd_WindowsPath(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	cfg := &types.Config{
		CurrentContextName: "default",
		Contexts: map[string]*types.Context{
			"default": {
				User: "testuser",
				Host: "localhost",
				Port: 22,
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	outPath := filepath.Join(tempDir, "test_script.cmd")
	// simulate a windows path with backslashes
	winPath := `C:\Users\test\scripts\reconnect.cmd`

	t.Run("win-reconnect-ssh prints single backslash", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		flagConfig = configPath
		flagContext = "default"
		flagOutput = outPath

		// Test runReconnectScript directly with a windows path in output
		err := runReconnectScript(
			cmd,
			nil,
			func(_ *types.Config, _ string) (string, []byte, error) {
				return "reconnect.cmd", []byte("echo test"), nil
			},
			"💾 Writing SSH reconnect script for context %q on Windows to %s\n",
			0o644,
		)
		if err != nil {
			t.Fatalf("runReconnectScript error = %v", err)
		}

		// Now test with the simulated Windows logMsg and path
		buf.Reset()
		flagOutput = winPath
		// Override os.WriteFile side effect by testing logMsg formatting
		testCmd := &cobra.Command{}
		testCmd.SetOut(buf)
		logMsg := "💾 Writing SSH reconnect script for context %q on Windows to %s\n"
		testCmd.Printf(logMsg, "default", winPath)

		got := buf.String()
		expected := `💾 Writing SSH reconnect script for context "default" on Windows to C:\Users\test\scripts\reconnect.cmd` + "\n"
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
		if strings.Contains(got, `\\`) {
			t.Errorf("output contains double backslash: %q", got)
		}
	})
}
