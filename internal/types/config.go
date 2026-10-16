package types

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = "config.yaml"
	ConfigDir      = ".config/gws"
	DirectionUp    = "up"
	DirectionDown  = "down"

	defaultSSHTimeoutSeconds   = 30
	defaultStartTimeoutSeconds = 300
)

// yamlUnicodeEscapeRe matches YAML unicode escape sequences like \U0001F3E0.
// The gopkg.in/yaml.v3 encoder escapes non-ASCII characters (including emojis)
// to these sequences. This regex is used to convert them back to UTF-8.
var yamlUnicodeEscapeRe = regexp.MustCompile(`\\U([0-9a-fA-F]{8})`)

// unescapeYAMLUnicode converts YAML unicode escape sequences back to their
// corresponding UTF-8 characters, so emojis are preserved in the config file.
func unescapeYAMLUnicode(s string) string {
	return yamlUnicodeEscapeRe.ReplaceAllStringFunc(s, func(match string) string {
		codePoint, err := strconv.ParseInt(match[2:], 16, 32)
		if err != nil {
			return match
		}
		return string(rune(codePoint))
	})
}

type Config struct {
	Contexts            map[string]*Context `validate:"required,dive,required"    yaml:"contexts"`
	CurrentContextName  string              `yaml:"currentContext"`
	FilePath            string              `yaml:"-"`
	TokenCheck          bool                `yaml:"-"`
	SSHTimeoutSeconds   int                 `yaml:"sshTimeoutSeconds,omitempty"`
	StartTimeoutSeconds int                 `yaml:"startTimeoutSeconds,omitempty"`
	currentContext      *Context
	Token               *TokenStorage `yaml:"-"`

	ChromeBrowser    *ChromeBrowserConfig `yaml:"chromeBrowser,omitempty"`
	JetbrainsGateway *GatewayConfig       `yaml:"jetbrainsGateway,omitempty"`

	TerminateOnShutdown bool `yaml:"terminateOnShutdown,omitempty"`

	AuthURLChan chan string `yaml:"-"`
}

type ChromeBrowserConfig struct {
	ExecutablePath   string `validate:"required,file" yaml:"executablePath"`
	ProfileDirectory string `validate:"required"      yaml:"profileDirectory"`
}

type GatewayConfig struct {
	DownloadDestination string `validate:"required" yaml:"downloadDestination"`
}

func (c *Config) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())

	validate.RegisterStructValidation(func(sl validator.StructLevel) {
		f, ok := sl.Current().Interface().(File)
		if !ok {
			return
		}

		switch f.Direction {
		case "", DirectionUp:
			info, err := os.Stat(f.SourcePath)
			if err != nil {
				sl.ReportError(f.SourcePath, "SourcePath", "sourcePath", "file", "")
				return
			}
			if !info.Mode().IsRegular() {
				sl.ReportError(f.SourcePath, "SourcePath", "sourcePath", "file", "")
				return
			}

		case DirectionDown:
			info, err := os.Stat(f.SourcePath)
			if err != nil {
				if os.IsNotExist(err) {
					return // ok if file not exists
				}
				sl.ReportError(f.SourcePath, "SourcePath", "sourcePath", "file", "")
				return
			}
			if !info.Mode().IsRegular() {
				// if exists it must be a file
				sl.ReportError(f.SourcePath, "SourcePath", "sourcePath", "file", "")
				return
			}
		}
	}, File{})

	err := validate.Struct(c)
	if err != nil {
		if validationErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
			for _, e := range validationErrors {
				switch e.Tag() {
				case "file":
					return fmt.Errorf("field '%s' with value '%v' must be a valid existing file path", e.Namespace(), e.Value())
				case "required":
					return fmt.Errorf("field '%s' is required", e.Namespace())
				}
			}
			return err
		}
	}
	return err
}

func (c *Config) CurrentContext() *Context {
	if c == nil {
		return nil
	}
	return c.currentContext
}

func (c *Config) Load(fileName string) error {
	var file string
	file, data, err := ReadGWSFile(fileName)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, c)
	if err != nil {
		return err
	}

	c.FilePath = file

	currentChanged := false
	if _, ok := c.Contexts[c.CurrentContextName]; !ok {
		c.CurrentContextName = ""
		currentChanged = true
	}

	if c.CurrentContextName == "" && len(c.Contexts) > 0 {
		for k := range c.Contexts {
			c.CurrentContextName = k
			currentChanged = true
			break
		}
	}

	tk, err := LoadToken()
	if err != nil {
		return err
	}
	c.Token = &TokenStorage{}
	if tk != nil {
		c.Token.Token = *tk
	}

	c.applyDefaults()

	if err := c.Validate(); err != nil {
		return err
	}
	return c.SwitchContext(c.CurrentContextName, currentChanged)
}

func (c *Config) applyDefaults() {
	for _, ctx := range c.Contexts {
		if ctx == nil {
			continue
		}
		for i := range ctx.Files {
			if ctx.Files[i].Direction == "" {
				ctx.Files[i].Direction = DirectionUp
			}
		}
	}
}

func (c *Config) SSHTimeout() time.Duration {
	if c == nil || c.SSHTimeoutSeconds <= 0 {
		return defaultSSHTimeoutSeconds * time.Second
	}
	return time.Duration(c.SSHTimeoutSeconds) * time.Second
}

func (c *Config) StartTimeout() time.Duration {
	if c == nil || c.StartTimeoutSeconds <= 0 {
		return defaultStartTimeoutSeconds * time.Second
	}
	return time.Duration(c.StartTimeoutSeconds) * time.Second
}

func ReadGWSFile(fileName string) (absoluteFile string, data []byte, err error) {
	var file string
	if fileName != "" {
		if _, err := os.Stat(fileName); err == nil {
			file = fileName
		}
	}

	if file == "" {
		// Try new location first
		newConfigPath, _, _ := DefaultConfigPaths()
		if _, err := os.Stat(newConfigPath); err == nil {
			file = newConfigPath
		}
	}

	abs, err := filepath.Abs(file)
	if err != nil {
		return "", nil, err
	}
	data, err = os.ReadFile(file)

	return abs, data, err
}

func DefaultConfigPaths() (newConfigPath, configDir, userHomeDir string) {
	userHomeDir, _ = os.UserHomeDir()

	newConfigPath = filepath.Join(userHomeDir, ConfigDir, ConfigFileName)
	configDir = filepath.Join(userHomeDir, ConfigDir)
	return newConfigPath, configDir, userHomeDir
}

func (c *Config) SwitchContext(newContext string, force bool) error {
	// Allow empty context when no contexts are defined (e.g., during initial setup)
	if newContext == "" && len(c.Contexts) == 0 {
		c.CurrentContextName = ""
		c.currentContext = nil
		return nil
	}

	if _, ok := c.Contexts[newContext]; !ok {
		return fmt.Errorf("context with name %q not defined", newContext)
	}

	if force || c.CurrentContextName != newContext {
		c.CurrentContextName = newContext

		if err := c.save(); err != nil {
			return err
		}
	}
	c.currentContext = c.Contexts[newContext]

	return nil
}

func (c *Config) UseContext(contextName string) error {
	if _, ok := c.Contexts[contextName]; !ok {
		return fmt.Errorf("context with name %q not defined", contextName)
	}
	c.currentContext = c.Contexts[contextName]
	return nil
}

func (c *Config) save() error {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err := encoder.Encode(c)
	if err != nil {
		return err
	}
	// The gopkg.in/yaml.v3 encoder escapes non-ASCII characters (e.g. emojis)
	// to Unicode escape sequences like \U0001F3E0. Convert them back to UTF-8
	// so the config file remains human-readable.
	data := []byte(unescapeYAMLUnicode(buf.String()))
	return os.WriteFile(c.FilePath, data, 0o600)
}

// SaveConfig writes the current config to disk.
func (c *Config) SaveConfig() error {
	return c.save()
}

func (c *Config) SetToken(token oauth2.Token) error {
	if c == nil {
		return nil
	}
	if c.Token == nil {
		c.Token = &TokenStorage{Token: token}
	}

	if c.Token.Token.AccessToken != token.AccessToken {
		c.Token.Token = token
		return SaveToken(c.Token.Token)
	}
	return nil
}

func (c *Config) DeleteToken() error {
	if c != nil {
		c.Token = nil
	}
	return DeleteToken()
}

// InitAuthChannels initializes the channels used for interactive login flow.
func (c *Config) InitAuthChannels() {
	if c == nil {
		return
	}
	if c.AuthURLChan == nil {
		c.AuthURLChan = make(chan string, 1)
	}
}

// SendAuthURL dispatches an authorization URL to the waiting TUI if configured.
func (c *Config) SendAuthURL(url string) {
	if c != nil && c.AuthURLChan != nil {
		select {
		case c.AuthURLChan <- url:
		default:
		}
	}
}
