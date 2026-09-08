package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestRootCmd_NoBrowserFlag(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configDir := filepath.Join(tempHome, types.ConfigDir)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configFile := filepath.Join(configDir, types.ConfigFileName)
	configYAML := `
currentContext: default
contexts:
  default:
    host: localhost
    port: 22022
    user: user
`
	if err := os.WriteFile(configFile, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	flagConfig = configFile
	flagNoBrowser = true
	defer func() {
		flagNoBrowser = false
		flagConfig = types.ConfigFileName
	}()

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}
	if !cfg.NoBrowser {
		t.Error("expected cfg.NoBrowser to be true when flagNoBrowser is true")
	}
}
