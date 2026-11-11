package gateway

import (
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

type proxyOptionForTest struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type proxyComponentForTest struct {
	Name    string               `xml:"name,attr"`
	Options []proxyOptionForTest `xml:"option"`
}

type proxySettingsForTest struct {
	Components []proxyComponentForTest `xml:"component"`
}

func TestFixProxyFileDisablesActiveProxy(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantHost     string
		wantOtherPAC string
	}{
		{
			name: "manual proxy",
			content: `<application>
  <component name="HttpConfigurable">
    <option name="USE_HTTP_PROXY" value="true" />
    <option name="PROXY_HOST" value="proxy.example.com" />
    <option name="PROXY_PORT" value="8080" />
  </component>
  <component name="Other">
    <option name="USE_PROXY_PAC" value="true" />
  </component>
</application>`,
			wantHost:     "proxy.example.com",
			wantOtherPAC: "true",
		},
		{
			name: "PAC proxy",
			content: `<application>
  <component name="HttpConfigurable">
    <option name="USE_HTTP_PROXY" value="false" />
    <option name="USE_PROXY_PAC" value="true" />
  </component>
</application>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "proxy.settings.xml")
			original := []byte(tt.content)
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatalf("failed to write proxy settings: %v", err)
			}

			changed, err := fixProxyFile(path)
			if err != nil {
				t.Fatalf("fixProxyFile() error = %v", err)
			}
			if !changed {
				t.Fatal("fixProxyFile() changed = false, want true")
			}

			backup, err := os.ReadFile(path + ".bak")
			if err != nil {
				t.Fatalf("failed to read backup: %v", err)
			}
			if !bytes.Equal(backup, original) {
				t.Fatalf("backup = %q, want original content %q", backup, original)
			}

			settings := readProxySettingsForTest(t, path)
			options := httpOptionsForTest(t, settings)
			if got := options[useHTTPProxyOption]; got != "false" {
				t.Errorf("%s = %q, want false", useHTTPProxyOption, got)
			}
			if got := options[useProxyPACOption]; got != "false" {
				t.Errorf("%s = %q, want false", useProxyPACOption, got)
			}
			if tt.wantHost != "" && options["PROXY_HOST"] != tt.wantHost {
				t.Errorf("PROXY_HOST = %q, want %q", options["PROXY_HOST"], tt.wantHost)
			}
			if tt.wantOtherPAC != "" {
				if got := optionValueForTest(t, settings, "Other", useProxyPACOption); got != tt.wantOtherPAC {
					t.Errorf("Other.%s = %q, want %q", useProxyPACOption, got, tt.wantOtherPAC)
				}
			}
		})
	}
}

func TestFixProxyFileAlreadyNoProxyDoesNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proxy.settings.xml")
	original := []byte(
		`<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="false"/><option name="USE_PROXY_PAC" value="false"/><option name="PROXY_HOST" value="proxy.example.com"/></component></application>`,
	)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("failed to write proxy settings: %v", err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat proxy settings: %v", err)
	}

	changed, err := fixProxyFile(path)
	if err != nil {
		t.Fatalf("fixProxyFile() error = %v", err)
	}
	if changed {
		t.Fatal("fixProxyFile() changed = true, want false")
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat proxy settings after check: %v", err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Errorf("proxy settings modification time changed from %v to %v", before.ModTime(), after.ModTime())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read proxy settings: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("proxy settings changed: got %q, want %q", got, original)
	}
	if _, err := os.Stat(path + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("backup stat error = %v, want os.ErrNotExist", err)
	}
}

func TestEnsureNoProxyInCreatesMissingProxySettings(t *testing.T) {
	tests := []struct {
		name             string
		createOptionsDir bool
	}{
		{name: "options directory missing"},
		{name: "proxy file missing", createOptionsDir: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := filepath.Join(t.TempDir(), "JetBrains")
			gatewayDir := filepath.Join(baseDir, "JetBrainsGateway2024.1")
			optionsDir := filepath.Join(gatewayDir, "options")
			if err := os.MkdirAll(gatewayDir, 0o755); err != nil {
				t.Fatalf("failed to create Gateway directory: %v", err)
			}
			if tt.createOptionsDir {
				if err := os.MkdirAll(optionsDir, 0o755); err != nil {
					t.Fatalf("failed to create options directory: %v", err)
				}
			}

			if err := ensureNoProxyIn(baseDir, false); err != nil {
				t.Fatalf("ensureNoProxyIn() error = %v", err)
			}

			path := filepath.Join(optionsDir, proxySettingsFile)
			settings := readProxySettingsForTest(t, path)
			httpProxy := optionValueForTest(t, settings, httpConfigurable, useHTTPProxyOption)
			proxyPAC := optionValueForTest(t, settings, httpConfigurable, useProxyPACOption)
			if httpProxy != "false" || proxyPAC != "false" {
				t.Fatal("new proxy settings do not disable both proxy options")
			}
		})
	}
}

func TestEnsureNoProxyInUpdatesEveryGatewayDirectory(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "JetBrains")
	first := writeProxySettingsForTest(
		t,
		baseDir,
		"JetBrainsGateway2024.1",
		`<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="true"/></component></application>`,
	)
	second := writeProxySettingsForTest(
		t,
		baseDir,
		"JetBrainsGateway2024.2",
		`<application><component name="HttpConfigurable"><option name="USE_PROXY_PAC" value="true"/></component></application>`,
	)

	if err := ensureNoProxyIn(baseDir, true); err != nil {
		t.Fatalf("ensureNoProxyIn() error = %v", err)
	}
	for _, path := range []string{first, second} {
		settings := readProxySettingsForTest(t, path)
		options := httpOptionsForTest(t, settings)
		if options[useHTTPProxyOption] != "false" || options[useProxyPACOption] != "false" {
			t.Errorf("proxy options in %q = %v, want both false", path, options)
		}
		if _, err := os.Stat(path + ".bak"); err != nil {
			t.Errorf("backup for %q: %v", path, err)
		}
	}
}

func TestEnsureNoProxyInUpdatesOnlyLatestGatewayDirectoryByDefault(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "JetBrains")
	olderContent := `<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="true"/></component></application>`
	latestContent := `<application><component name="HttpConfigurable"><option name="USE_PROXY_PAC" value="true"/></component></application>`
	older := writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.2", olderContent)
	latest := writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.10", latestContent)

	if err := ensureNoProxyIn(baseDir, false); err != nil {
		t.Fatalf("ensureNoProxyIn() error = %v", err)
	}

	got, err := os.ReadFile(older)
	if err != nil {
		t.Fatalf("failed to read older proxy settings: %v", err)
	}
	if string(got) != olderContent {
		t.Errorf("older proxy settings changed: got %q, want %q", got, olderContent)
	}
	if _, err := os.Stat(older + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("older backup stat error = %v, want os.ErrNotExist", err)
	}

	settings := readProxySettingsForTest(t, latest)
	options := httpOptionsForTest(t, settings)
	if options[useHTTPProxyOption] != "false" || options[useProxyPACOption] != "false" {
		t.Errorf("latest proxy options = %v, want both false", options)
	}
	if _, err := os.Stat(latest + ".bak"); err != nil {
		t.Errorf("latest backup: %v", err)
	}
}

func TestLatestGatewayDirUsesNumericVersionOrdering(t *testing.T) {
	configDirs := []string{
		"/JetBrains/JetBrainsGateway2024.2",
		"/JetBrains/JetBrainsGateway2024.10",
		"/JetBrains/JetBrainsGateway2023.12",
	}
	if got, want := filepath.Base(latestGatewayDir(configDirs)), "JetBrainsGateway2024.10"; got != want {
		t.Errorf("latestGatewayDir() = %q, want %q", got, want)
	}
}

func TestEnsureNoProxyInNoGatewayDirectories(t *testing.T) {
	if err := ensureNoProxyIn(t.TempDir(), false); err != nil {
		t.Fatalf("ensureNoProxyIn() error = %v, want nil", err)
	}
}

func TestEnsureNoProxyInMalformedXMLLeavesFileUnchanged(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "JetBrains")
	path := writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.1", `<application><component`)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read original proxy settings: %v", err)
	}

	err = ensureNoProxyIn(baseDir, false)
	if err == nil || !strings.Contains(err.Error(), "decode proxy settings") {
		t.Fatalf("ensureNoProxyIn() error = %v, want a decode error", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read proxy settings after error: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("malformed proxy settings changed: got %q, want %q", got, original)
	}
	if _, err := os.Stat(path + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("backup stat error = %v, want os.ErrNotExist", err)
	}
}

func TestEnsureNoProxyIsOptOut(t *testing.T) {
	active := `<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="true"/></component></application>`
	tests := []struct {
		name        string
		cfg         *types.Config
		wantChanged bool
	}{
		{name: "nil config", wantChanged: true},
		{name: "no gateway config", cfg: &types.Config{}, wantChanged: true},
		{name: "empty gateway config", cfg: &types.Config{JetbrainsGateway: &types.GatewayConfig{}}, wantChanged: true},
		{
			name: "skipNoProxyCheck",
			cfg:  &types.Config{JetbrainsGateway: &types.GatewayConfig{SkipNoProxyCheck: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := isolateUserConfigDirForTest(t)
			path := writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2025.3", active)

			if err := EnsureNoProxy(t.Context(), tt.cfg); err != nil {
				t.Fatalf("EnsureNoProxy() error = %v, want nil", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read proxy settings: %v", err)
			}
			if changed := string(got) != active; changed != tt.wantChanged {
				t.Errorf("proxy settings changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}

func TestEnsureNoProxyUsesAllVersionsWhenConfigured(t *testing.T) {
	baseDir := isolateUserConfigDirForTest(t)
	active := `<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="true"/></component></application>`
	paths := []string{
		writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.2", active),
		writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.10", active),
	}
	cfg := &types.Config{JetbrainsGateway: &types.GatewayConfig{CheckAllNoProxyVersions: true}}

	if err := EnsureNoProxy(t.Context(), cfg); err != nil {
		t.Fatalf("EnsureNoProxy() error = %v", err)
	}
	for _, path := range paths {
		settings := readProxySettingsForTest(t, path)
		options := httpOptionsForTest(t, settings)
		if options[useHTTPProxyOption] != "false" || options[useProxyPACOption] != "false" {
			t.Errorf("proxy options in %q = %v, want both false", path, options)
		}
		if _, err := os.Stat(path + ".bak"); err != nil {
			t.Errorf("backup for %q: %v", path, err)
		}
	}
}

func TestCheckNoProxyWarnsWithoutWriting(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		cfg      *types.Config
		wantWarn bool
	}{
		{
			name:     "manual proxy",
			content:  `<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="true"/></component></application>`,
			wantWarn: true,
		},
		{
			name:     "pac proxy",
			content:  `<application><component name="HttpConfigurable"><option name="USE_PROXY_PAC" value="true"/></component></application>`,
			wantWarn: true,
		},
		{
			name:     "missing file",
			wantWarn: true,
		},
		{
			name:    "no proxy",
			content: `<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="false"/></component></application>`,
		},
		{
			name:    "skipNoProxyCheck",
			content: `<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="true"/></component></application>`,
			cfg:     &types.Config{JetbrainsGateway: &types.GatewayConfig{SkipNoProxyCheck: true}},
		},
	}

	previous := log.GetLogger()
	t.Cleanup(func() { log.SetLogger(previous) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := isolateUserConfigDirForTest(t)
			path := filepath.Join(baseDir, "JetBrainsGateway2025.3", "options", proxySettingsFile)
			if tt.content != "" {
				writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2025.3", tt.content)
			} else if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("failed to create gateway directory: %v", err)
			}

			var logs []string
			log.SetLogger(func(s string) { logs = append(logs, s) })

			if err := CheckNoProxy(t.Context(), tt.cfg); err != nil {
				t.Fatalf("CheckNoProxy() error = %v, want nil", err)
			}

			warned := strings.Contains(strings.Join(logs, "\n"), "⚠️  WARNING: Gateway proxy is NOT set to No proxy")
			if warned != tt.wantWarn {
				t.Errorf("warning logged = %v, want %v; logs = %q", warned, tt.wantWarn, logs)
			}

			got, err := os.ReadFile(path)
			if tt.content == "" {
				if !errors.Is(err, os.ErrNotExist) {
					t.Errorf("proxy settings stat error = %v, want os.ErrNotExist", err)
				}
			} else if err != nil || string(got) != tt.content {
				t.Errorf("proxy settings changed: got %q (err %v), want %q", got, err, tt.content)
			}
			if _, err := os.Stat(path + ".bak"); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("backup stat error = %v, want os.ErrNotExist", err)
			}
		})
	}
}

func TestCheckNoProxyUsesLatestVersionByDefault(t *testing.T) {
	active := `<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="true"/></component></application>`
	noProxy := `<application><component name="HttpConfigurable"><option name="USE_HTTP_PROXY" value="false"/></component></application>`
	tests := []struct {
		name     string
		cfg      *types.Config
		wantWarn bool
	}{
		{name: "latest only"},
		{
			name:     "all versions",
			cfg:      &types.Config{JetbrainsGateway: &types.GatewayConfig{CheckAllNoProxyVersions: true}},
			wantWarn: true,
		},
	}

	previous := log.GetLogger()
	t.Cleanup(func() { log.SetLogger(previous) })
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := isolateUserConfigDirForTest(t)
			writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.2", active)
			writeProxySettingsForTest(t, baseDir, "JetBrainsGateway2024.10", noProxy)

			var logs []string
			log.SetLogger(func(s string) { logs = append(logs, s) })
			if err := CheckNoProxy(t.Context(), tt.cfg); err != nil {
				t.Fatalf("CheckNoProxy() error = %v", err)
			}

			warned := strings.Contains(strings.Join(logs, "\n"), "⚠️  WARNING: Gateway proxy is NOT set to No proxy")
			if warned != tt.wantWarn {
				t.Errorf("warning logged = %v, want %v; logs = %q", warned, tt.wantWarn, logs)
			}
		})
	}
}

func TestNoProxyWarningIsTUIFriendly(t *testing.T) {
	lines := noProxyWarning("/home/user/.config/JetBrains/JetBrainsGateway2025.3/options/ide.general.xml")
	if len(lines) == 0 {
		t.Fatal("noProxyWarning() returned no lines")
	}

	for _, line := range lines {
		if strings.ContainsAny(line, "\r\n") {
			t.Errorf("warning line contains a newline: %q", line)
		}
		if strings.Contains(line, "!!!") {
			t.Errorf("warning line contains a banner: %q", line)
		}
		if length := len([]rune(line)); length > 100 {
			t.Errorf("warning line is too long (%d characters): %q", length, line)
		}
	}
}

// isolateUserConfigDirForTest points os.UserConfigDir to a temp directory and returns its JetBrains folder.
func isolateUserConfigDirForTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("HOME", dir)
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir() error = %v", err)
	}
	return filepath.Join(configDir, jetBrainsConfigDir)
}

func writeProxySettingsForTest(t *testing.T, baseDir, gatewayDir, content string) string {
	t.Helper()
	path := filepath.Join(baseDir, gatewayDir, "options", proxySettingsFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create proxy settings directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write proxy settings: %v", err)
	}
	return path
}

func readProxySettingsForTest(t *testing.T, path string) proxySettingsForTest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read proxy settings: %v", err)
	}
	var settings proxySettingsForTest
	if err := xml.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to decode proxy settings: %v", err)
	}
	return settings
}

func httpOptionsForTest(t *testing.T, settings proxySettingsForTest) map[string]string {
	t.Helper()
	return optionsForTest(t, settings, httpConfigurable)
}

func optionValueForTest(t *testing.T, settings proxySettingsForTest, componentName, optionName string) string {
	t.Helper()
	return optionsForTest(t, settings, componentName)[optionName]
}

func optionsForTest(t *testing.T, settings proxySettingsForTest, componentName string) map[string]string {
	t.Helper()
	for _, component := range settings.Components {
		if component.Name != componentName {
			continue
		}
		options := make(map[string]string, len(component.Options))
		for _, option := range component.Options {
			options[option.Name] = option.Value
		}
		return options
	}
	t.Fatalf("component %q not found", componentName)
	return nil
}
