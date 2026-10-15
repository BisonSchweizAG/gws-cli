//go:build windows

package gcloud

import (
	"context"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

// OpenBrowser opens the given URL in the default browser.
func OpenBrowser(_ context.Context, cfg *types.Config, authURL string) {
	_ = windowsCmd(context.Background(), cfg, authURL).Start()
}
