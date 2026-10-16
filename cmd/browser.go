package cmd

import (
	"context"
	"strings"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

// openURL logs and opens the given URL in the browser.
// The name is used for the log messages, e.g. "Workstation" or "Dashboard".
func openURL(ctx context.Context, cfg *types.Config, name, url string) {
	log.Logf("Opening %s URL: %s", strings.ToLower(name), url)
	gcloud.OpenBrowser(ctx, cfg, url)
	log.Logf("%s URL opened in browser", name)
}
