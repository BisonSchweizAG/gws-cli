//go:build windows

package gateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
	"golang.org/x/sys/windows/registry"
)

const (
	keyJetBrainsClient       = `SOFTWARE\JetBrains\JetBrainsClient`
	fullKey                  = `HKEY_CURRENT_USER\` + keyJetBrainsClient
	valueDownloadDestination = "downloadDestination"
)

func CheckDownloadLocation(_ context.Context, cfg *types.Config) error {
	if cfg == nil || cfg.JetbrainsGateway == nil || cfg.JetbrainsGateway.DownloadDestination == "" {
		return nil
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, keyJetBrainsClient, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			log.Logf(
				`⚠️ WARNING: JetbrainsGateway DownloadDestination registry key "%s" does not exist; expected "%s"`,
				fullKey,
				cfg.JetbrainsGateway.DownloadDestination,
			)
			return nil
		}
		return fmt.Errorf("open JetBrains registry key: %w", err)
	}
	defer key.Close()

	current, _, err := key.GetStringValue(valueDownloadDestination)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			log.Logf(
				`⚠️ WARNING: JetbrainsGateway DownloadDestination is not configured in "%s"; expected "%s"`,
				fullKey,
				cfg.JetbrainsGateway.DownloadDestination,
			)
			return nil
		}
		return fmt.Errorf("read %s: %w", valueDownloadDestination, err)
	}

	if current == cfg.JetbrainsGateway.DownloadDestination {
		log.Logf(`✅ JetbrainsGateway DownloadDestination is correctly set in "%s"`, fullKey)
	} else {
		log.Logf(
			`⚠️ WARNING: JetbrainsGateway DownloadDestination is "%s" in "%s", expected "%s"`,
			current,
			fullKey,
			cfg.JetbrainsGateway.DownloadDestination,
		)
	}
	return nil
}

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
		log.Logf(`🔧 Creating registry key "%s"`, fullKey)
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
		log.Logf(`✅ JetbrainsGateway DownloadDestination is correctly set in "%s"`, fullKey)
		return nil
	}

	// Value is missing or different, so update it.
	if current == "" {
		log.Logf(`🔧 Creating JetbrainsGateway DownloadDestination registry entry "%s" in "%s"`, cfg.JetbrainsGateway.DownloadDestination, fullKey)
	} else {
		log.Logf(`🔧 Updating JetbrainsGateway DownloadDestination registry entry from "%s" to "%s" in "%s"`, current, cfg.JetbrainsGateway.DownloadDestination, fullKey)
	}
	if err := key.SetStringValue(valueDownloadDestination, cfg.JetbrainsGateway.DownloadDestination); err != nil {
		return fmt.Errorf("set %s: %w", valueDownloadDestination, err)
	}

	return nil
}
