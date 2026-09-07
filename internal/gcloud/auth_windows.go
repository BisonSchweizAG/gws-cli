//go:build windows

package gcloud

import (
	"context"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func openBrowser(ctx context.Context, cfg *types.Config, authURL string) {
	_ = windowsCmd(ctx, cfg, authURL).Start()
}
