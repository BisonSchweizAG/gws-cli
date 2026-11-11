package gateway

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

const activeProxySettings = `<application><component name="HttpConfigurable">` +
	`<option name="USE_HTTP_PROXY" value="true"/></component></application>`

func TestRunLogsSkippedTasks(t *testing.T) {
	tests := []struct {
		name string
		cfg  *types.Config
		opts Options
		want []string
	}{
		{
			name: "nil config",
			want: []string{
				"⏭️ JetBrains Gateway downloadDestination is not configured; skipping",
				"🔍 Checking JetBrains Gateway proxy settings",
				`📂 No JetBrains Gateway config directories found in "{base}"`,
				"🏁 JetBrains Gateway tasks finished",
			},
		},
		{
			name: "no gateway config",
			cfg:  &types.Config{},
			want: []string{
				"⏭️ JetBrains Gateway downloadDestination is not configured; skipping",
				"🔍 Checking JetBrains Gateway proxy settings",
				`📂 No JetBrains Gateway config directories found in "{base}"`,
				"🏁 JetBrains Gateway tasks finished",
			},
		},
		{
			name: "proxy check skipped",
			cfg:  &types.Config{JetbrainsGateway: &types.GatewayConfig{SkipNoProxyCheck: true}},
			want: []string{
				"⏭️ JetBrains Gateway downloadDestination is not configured; skipping",
				"⏭️ JetBrains Gateway skipNoProxyCheck is enabled; skipping proxy settings",
				"🏁 JetBrains Gateway tasks finished",
			},
		},
		{
			name: "proxy check skipped in check mode",
			cfg:  &types.Config{JetbrainsGateway: &types.GatewayConfig{SkipNoProxyCheck: true}},
			opts: Options{Check: true},
			want: []string{
				"⏭️ JetBrains Gateway downloadDestination is not configured; skipping",
				"⏭️ JetBrains Gateway skipNoProxyCheck is enabled; skipping proxy settings",
				"🏁 JetBrains Gateway tasks finished",
			},
		},
	}

	previous := log.GetLogger()
	t.Cleanup(func() { log.SetLogger(previous) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := isolateUserConfigDirForTest(t)
			want := make([]string, len(tt.want))
			for i, line := range tt.want {
				want[i] = strings.ReplaceAll(line, "{base}", baseDir)
			}

			var got []string
			log.SetLogger(func(s string) { got = append(got, s) })

			if err := Run(t.Context(), tt.cfg, tt.opts); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !slices.Equal(got, want) {
				t.Fatalf("Run() logs = %q, want %q", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
		})
	}
}

func TestRunCheckDoesNotModifyProxySettings(t *testing.T) {
	baseDir := isolateUserConfigDirForTest(t)
	path := writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.1", activeProxySettings)

	previous := log.GetLogger()
	t.Cleanup(func() { log.SetLogger(previous) })
	var logs []string
	log.SetLogger(func(s string) { logs = append(logs, s) })

	if err := Run(t.Context(), nil, Options{Check: true}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read proxy settings: %v", err)
	}
	if string(got) != activeProxySettings {
		t.Errorf("proxy settings changed: got %q, want %q", got, activeProxySettings)
	}
	if _, err := os.Stat(path + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("backup stat error = %v, want file not to exist", err)
	}
	if !strings.Contains(strings.Join(logs, "\n"), "⚠️  WARNING: Gateway proxy is NOT set to No proxy") {
		t.Errorf("check warning was not logged; logs = %q", logs)
	}
}

func TestRunAllOptionUpdatesEveryGatewayVersion(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *types.Config
		all       bool
		wantOlder bool
	}{
		{name: "latest only by default"},
		{name: "all versions", all: true, wantOlder: true},
		{
			name:      "all versions from config",
			cfg:       &types.Config{JetbrainsGateway: &types.GatewayConfig{CheckAllNoProxyVersions: true}},
			wantOlder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := isolateUserConfigDirForTest(t)
			older := writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.1", activeProxySettings)
			latest := writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.10", activeProxySettings)

			if err := Run(t.Context(), tt.cfg, Options{All: tt.all}); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			assertProxySettingsUpdated(t, latest)
			if tt.wantOlder {
				assertProxySettingsUpdated(t, older)
			} else {
				got, err := os.ReadFile(older)
				if err != nil {
					t.Fatalf("failed to read older proxy settings: %v", err)
				}
				if string(got) != activeProxySettings {
					t.Errorf("older proxy settings changed: got %q, want %q", got, activeProxySettings)
				}
				if _, err := os.Stat(older + ".bak"); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("older backup stat error = %v, want file not to exist", err)
				}
			}
		})
	}
}

func assertProxySettingsUpdated(t *testing.T, path string) {
	t.Helper()
	settings := readProxySettingsForTest(t, path)
	options := httpOptionsForTest(t, settings)
	if options[useHTTPProxyOption] != "false" || options[useProxyPACOption] != "false" {
		t.Errorf("proxy options in %q = %v, want both false", path, options)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Errorf("backup for %q: %v", filepath.Base(path), err)
	}
}
