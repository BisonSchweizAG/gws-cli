package gcloud

import (
	"context"
	"os/exec"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func windowsCmd(ctx context.Context, cfg *types.Config, authURL string) *exec.Cmd {
	if cfg != nil && cfg.ChromeBrowser != nil && cfg.ChromeBrowser.ExecutablePath != "" &&
		cfg.ChromeBrowser.ProfileDirectory != "" {
		return exec.CommandContext(ctx, cfg.ChromeBrowser.ExecutablePath,
			"--profile-directory="+cfg.ChromeBrowser.ProfileDirectory,
			authURL,
		)
	}
	return exec.CommandContext(ctx, "cmd.exe", "/c", "start", "", authURL)
}
