//go:build windows

package gateway

import (
	"context"
	"fmt"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
	"golang.org/x/sys/windows/registry"
)

const (
	keyJetBrainsClient       = `SOFTWARE\JetBrains\JetBrainsClient`
	fullKey                  = `HKEY_CURRENT_USERS\` + keyJetBrainsClient
	valueDownloadDestination = "downloadDestination"
)

func UpdateDownloadLocation(ctx context.Context, cfg *types.Config) error {
	if cfg == nil || cfg.JetbrainsGateway == nil || cfg.JetbrainsGateway.DownloadDestination == "" {
		return nil
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, keyJetBrainsClient, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		if err != registry.ErrNotExist {
			return fmt.Errorf("open JetBrains registry key: %w", err)
		}

		// Key doesn't exist, so create it.
		log.Logf("Creating registry key %q", fullKey)
		key, _, err = registry.CreateKey(registry.CURRENT_USER, keyJetBrainsClient, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("create JetBrains registry key: %w", err)
		}
	}
	defer key.Close()

	// Read the current value.
	current, _, err := key.GetStringValue(valueDownloadDestination)
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("read %s: %w", valueDownloadDestination, err)
	}

	// Already configured correctly; don't write anything.
	if err == nil && current == cfg.JetbrainsGateway.DownloadDestination {
		log.Logf("JetbrainsGateway DownloadDestination is correctly set in %q", fullKey)
		return nil
	}

	// Value is missing or different, so update it.
	if current == "" {
		log.Logf("Creating JetbrainsGateway DownloadDestination registry entry %q in %q", cfg.JetbrainsGateway.DownloadDestination, fullKey)
	} else {
		log.Logf("Updating JetbrainsGateway DownloadDestination registry entry from %q to %q in %q", current, cfg.JetbrainsGateway.DownloadDestination, fullKey)
	}
	if err := key.SetStringValue(valueDownloadDestination, cfg.JetbrainsGateway.DownloadDestination); err != nil {
		return fmt.Errorf("set %s: %w", valueDownloadDestination, err)
	}

	return nil
}
