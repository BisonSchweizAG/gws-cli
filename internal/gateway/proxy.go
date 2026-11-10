package gateway

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

const (
	jetBrainsConfigDir = "JetBrains"
	gatewayConfigGlob  = "JetBrainsGateway*"
	proxySettingsFile  = "proxy.settings.xml"
	httpConfigurable   = "HttpConfigurable"
	useHTTPProxyOption = "USE_HTTP_PROXY"
	useProxyPACOption  = "USE_PROXY_PAC"
)

type proxyXMLNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr     `xml:",any,attr"`
	Text     string         `xml:",chardata"`
	Children []proxyXMLNode `xml:",any"`
}

// SkipNoProxyCheck reports whether the JetBrains Gateway No proxy check is disabled in the config.
func SkipNoProxyCheck(cfg *types.Config) bool {
	return cfg != nil && cfg.JetbrainsGateway != nil && cfg.JetbrainsGateway.SkipNoProxyCheck
}

// EnsureNoProxy sets every JetBrains Gateway config to No proxy unless skipNoProxyCheck is set.
func EnsureNoProxy(_ context.Context, cfg *types.Config) error {
	if SkipNoProxyCheck(cfg) {
		return nil
	}

	baseDir, err := jetBrainsBaseDir()
	if err != nil {
		return err
	}

	return ensureNoProxyIn(baseDir)
}

// CheckNoProxy logs a prominent warning for every JetBrains Gateway config that is not set to No proxy.
// It never modifies any file and does nothing when skipNoProxyCheck is set.
func CheckNoProxy(_ context.Context, cfg *types.Config) error {
	if SkipNoProxyCheck(cfg) {
		return nil
	}

	baseDir, err := jetBrainsBaseDir()
	if err != nil {
		return err
	}

	return checkNoProxyIn(baseDir)
}

func jetBrainsBaseDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config directory: %w", err)
	}
	return filepath.Join(configDir, jetBrainsConfigDir), nil
}

func checkNoProxyIn(baseDir string) error {
	return forEachProxySettingsFile(baseDir, func(path string) error {
		ok, err := isNoProxyFile(path)
		if err != nil {
			return err
		}
		if ok {
			log.Logf(`JetBrains Gateway proxy settings are correctly set in "%s"`, path)
			return nil
		}
		log.Log(noProxyWarning(path))
		return nil
	})
}

func noProxyWarning(path string) string {
	banner := strings.Repeat("!", 100)
	return strings.Join([]string{
		banner,
		"!!! WARNING: JetBrains Gateway is NOT set to \"No proxy\"",
		"!!! Settings file: " + path,
		"!!! Connections through the gws tunnel may fail behind a proxy.",
		"!!! Run `gws jetbrains-gateway` to fix it and restart Gateway,",
		"!!! or set `jetbrainsGateway.skipNoProxyCheck: true` to disable this check.",
		banner,
	}, "\n")
}

func ensureNoProxyIn(baseDir string) error {
	return forEachProxySettingsFile(baseDir, func(path string) error {
		changed, err := fixProxyFile(path)
		if err != nil {
			return err
		}
		if changed {
			log.Logf(`JetBrains Gateway proxy settings set to No proxy in "%s"; restart Gateway to apply the change`, path)
		} else {
			log.Logf(`JetBrains Gateway proxy settings are correctly set in "%s"`, path)
		}
		return nil
	})
}

func forEachProxySettingsFile(baseDir string, fn func(path string) error) error {
	pattern := filepath.Join(baseDir, gatewayConfigGlob)
	configDirs, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("find JetBrains Gateway config directories in %q: %w", baseDir, err)
	}

	found := false
	for _, configDir := range configDirs {
		info, err := os.Stat(configDir)
		if err != nil {
			return fmt.Errorf("stat JetBrains Gateway config directory %q: %w", configDir, err)
		}
		if !info.IsDir() {
			continue
		}
		found = true

		if err := fn(filepath.Join(configDir, "options", proxySettingsFile)); err != nil {
			return err
		}
	}

	if !found {
		log.Logf(`No JetBrains Gateway config directories found in "%s"`, baseDir)
	}

	return nil
}

// isNoProxyFile reports whether the file at path is explicitly set to No proxy. A missing file is reported as not set.
func isNoProxyFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read proxy settings %q: %w", path, err)
	}

	root, err := decodeProxySettings(data, path)
	if err != nil {
		return false, err
	}
	return !disableProxy(&root), nil
}

func fixProxyFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data, err = marshalNoProxySettings()
		if err != nil {
			return false, fmt.Errorf("create No proxy settings: %w", err)
		}
		if err := writeProxySettings(path, data); err != nil {
			return false, err
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read proxy settings %q: %w", path, err)
	}

	root, err := decodeProxySettings(data, path)
	if err != nil {
		return false, err
	}
	if !disableProxy(&root) {
		return false, nil
	}

	normalizeXMLWhitespace(&root)
	updated, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode proxy settings %q: %w", path, err)
	}
	updated = append(updated, '\n')

	if err := os.WriteFile(path+".bak", data, 0o600); err != nil {
		return false, fmt.Errorf("back up proxy settings %q: %w", path, err)
	}
	if err := os.WriteFile(path, updated, 0o600); err != nil {
		return false, fmt.Errorf("write proxy settings %q: %w", path, err)
	}

	return true, nil
}

func decodeProxySettings(data []byte, path string) (proxyXMLNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root proxyXMLNode
	if err := decoder.Decode(&root); err != nil {
		return proxyXMLNode{}, fmt.Errorf("decode proxy settings %q: %w", path, err)
	}
	if root.XMLName.Local != "application" {
		return proxyXMLNode{}, fmt.Errorf(
			"decode proxy settings %q: expected <application> root element, got <%s>",
			path,
			root.XMLName.Local,
		)
	}

	for {
		_, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return proxyXMLNode{}, fmt.Errorf("decode proxy settings %q: %w", path, err)
		}
	}

	return root, nil
}

func disableProxy(root *proxyXMLNode) bool {
	var httpComponent *proxyXMLNode
	var active bool
	var hasHTTPProxy bool
	var hasProxyPAC bool

	for i := range root.Children {
		component := &root.Children[i]
		if component.XMLName.Local != "component" || xmlAttribute(component, "name") != httpConfigurable {
			continue
		}
		if httpComponent == nil {
			httpComponent = component
		}

		for j := range component.Children {
			option := &component.Children[j]
			if option.XMLName.Local != "option" {
				continue
			}

			name := xmlAttribute(option, "name")
			value := xmlAttribute(option, "value")
			switch name {
			case useHTTPProxyOption:
				hasHTTPProxy = true
				active = active || value == "true"
			case useProxyPACOption:
				hasProxyPAC = true
				active = active || value == "true"
			}
		}
	}

	if !active {
		return false
	}

	for i := range root.Children {
		component := &root.Children[i]
		if component.XMLName.Local != "component" || xmlAttribute(component, "name") != httpConfigurable {
			continue
		}
		for j := range component.Children {
			option := &component.Children[j]
			if option.XMLName.Local != "option" {
				continue
			}
			switch xmlAttribute(option, "name") {
			case useHTTPProxyOption, useProxyPACOption:
				setXMLAttribute(option, "value", "false")
			}
		}
	}

	if !hasHTTPProxy {
		httpComponent.Children = append(httpComponent.Children, newProxyOption(useHTTPProxyOption))
	}
	if !hasProxyPAC {
		httpComponent.Children = append(httpComponent.Children, newProxyOption(useProxyPACOption))
	}

	return true
}

func newProxyOption(name string) proxyXMLNode {
	option := proxyXMLNode{XMLName: xml.Name{Local: "option"}}
	setXMLAttribute(&option, "name", name)
	setXMLAttribute(&option, "value", "false")
	return option
}

func xmlAttribute(node *proxyXMLNode, name string) string {
	for _, attr := range node.Attrs {
		if attr.Name.Local == name && attr.Name.Space == "" {
			return attr.Value
		}
	}
	return ""
}

func setXMLAttribute(node *proxyXMLNode, name, value string) {
	for i := range node.Attrs {
		if node.Attrs[i].Name.Local == name && node.Attrs[i].Name.Space == "" {
			node.Attrs[i].Value = value
			return
		}
	}
	node.Attrs = append(node.Attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

func normalizeXMLWhitespace(node *proxyXMLNode) {
	if strings.TrimSpace(node.Text) == "" {
		node.Text = ""
	}
	for i := range node.Children {
		normalizeXMLWhitespace(&node.Children[i])
	}
}

func marshalNoProxySettings() ([]byte, error) {
	component := proxyXMLNode{XMLName: xml.Name{Local: "component"}}
	setXMLAttribute(&component, "name", httpConfigurable)
	component.Children = append(component.Children,
		newProxyOption(useHTTPProxyOption),
		newProxyOption(useProxyPACOption),
	)

	data, err := xml.MarshalIndent(proxyXMLNode{
		XMLName:  xml.Name{Local: "application"},
		Children: []proxyXMLNode{component},
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func writeProxySettings(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create proxy settings directory for %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write proxy settings %q: %w", path, err)
	}
	return nil
}
