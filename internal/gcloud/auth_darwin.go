//go:build darwin

package gcloud

import (
	"context"
	"os/exec"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

// OpenBrowser opens the given URL in the default browser.
func OpenBrowser(_ context.Context, _ *types.Config, authURL string) {
	_ = exec.CommandContext(context.Background(), "open", authURL).Start()
}
