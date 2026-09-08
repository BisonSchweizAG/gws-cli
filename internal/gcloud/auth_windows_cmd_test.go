package gcloud

import (
	"slices"
	"testing"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestWindowsCmd_Default(t *testing.T) {
	ctx := t.Context()
	authURL := "https://accounts.google.com/o/oauth2/auth"

	cmd := windowsCmd(ctx, nil, authURL)
	expectedArgs := []string{"cmd.exe", "/c", "start", "", authURL}
	if !slices.Equal(cmd.Args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, cmd.Args)
	}
}

func TestWindowsCmd_WithChromeBrowser(t *testing.T) {
	ctx := t.Context()
	authURL := "https://accounts.google.com/o/oauth2/auth"
	cfg := &types.Config{
		ChromeBrowser: &types.ChromeBrowserConfig{
			ExecutablePath:   `C:\Program Files\Google\Chrome\Application\chrome.exe`,
			ProfileDirectory: "Profile 1",
		},
	}

	cmd := windowsCmd(ctx, cfg, authURL)
	expectedArgs := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		"--profile-directory=Profile 1",
		authURL,
	}
	if !slices.Equal(cmd.Args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, cmd.Args)
	}
}
