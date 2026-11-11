package gateway

import (
	"context"
	"runtime"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

// Options controls which JetBrains Gateway tasks Run performs.
type Options struct {
	All   bool
	Check bool
}

// Run executes all JetBrains Gateway tasks according to the config and logs their outcome.
func Run(ctx context.Context, cfg *types.Config, opts Options) error {
	var gw types.GatewayConfig
	if cfg != nil && cfg.JetbrainsGateway != nil {
		gw = *cfg.JetbrainsGateway
	}
	all := opts.All || checkAllNoProxyVersions(cfg)

	switch {
	case gw.DownloadDestination == "":
		log.Log("JetBrains Gateway downloadDestination is not configured; skipping")
	case runtime.GOOS != "windows":
		log.Logf("JetBrains Gateway downloadDestination is only supported on Windows; skipping on %s", runtime.GOOS)
	default:
		log.Log("Checking JetBrains Gateway downloadDestination")
		var err error
		if opts.Check {
			err = CheckDownloadLocation(ctx, cfg)
		} else {
			err = UpdateDownloadLocation(ctx, cfg)
		}
		if err != nil {
			return err
		}
	}

	if gw.SkipNoProxyCheck {
		log.Log("JetBrains Gateway skipNoProxyCheck is enabled; skipping proxy settings")
	} else {
		log.Log("Checking JetBrains Gateway proxy settings")
		baseDir, err := jetBrainsBaseDir()
		if err != nil {
			return err
		}
		if opts.Check {
			err = checkNoProxyIn(baseDir, all)
		} else {
			err = ensureNoProxyIn(baseDir, all)
		}
		if err != nil {
			return err
		}
	}

	log.Log("JetBrains Gateway tasks finished")
	return nil
}
