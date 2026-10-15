//go:build windows

package gcloud

import (
	"context"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

// OpenBrowser opens the given URL in the default browser.
func OpenBrowser(ctx context.Context, cfg *types.Config, authURL string) {
	_ = windowsCmd(ctx, cfg, authURL).Start()
}
