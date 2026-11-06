package script

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bisonschweizag/gws-cli/internal/types"

	_ "embed"
)

//go:embed ssh-config.tmpl
var sshConfigTemplate string

// SSHConfigIncludeDirective is the directive added to ~/.ssh/config so that
// snippets dropped into ~/.ssh/config.d are picked up by ssh.
const SSHConfigIncludeDirective = "Include config.d/*"

func SSHConfig(cfg *types.Config, fileName string) (name string, content []byte, err error) {
	return SSHConfigWithForwardAgent(false)(cfg, fileName)
}

// SSHConfigWithForwardAgent returns an ssh config generator that optionally
// adds the "ForwardAgent yes" directive to the generated snippet.
func SSHConfigWithForwardAgent(forwardAgent bool) func(*types.Config, string) (string, []byte, error) {
	return SSHConfigWithOptions(forwardAgent, "")
}

// SSHConfigWithOptions returns an ssh config generator that optionally adds the
// "ForwardAgent yes" directive and overrides the host name of the generated
// snippet. An empty hostName falls back to the current context name.
func SSHConfigWithOptions(forwardAgent bool, hostName string) func(*types.Config, string) (string, []byte, error) {
	return func(cfg *types.Config, fileName string) (name string, content []byte, err error) {
		configName := hostName
		if configName == "" {
			configName = cfg.CurrentContextName
		}

		if fileName == "" {
			fileName = "gws-" + configName
		}

		return renderNamed(sshConfigTemplate, cfg, fileName, "gws", "", func(ctx *NamedContext) {
			ctx.ForwardAgent = forwardAgent
			ctx.Name = configName
		})
	}
}

// SSHConfigDir returns the directory where generated ssh config snippets are
// saved: ~/.ssh/config.d. The directory is created if it does not exist.
func SSHConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".ssh", "config.d")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	return dir, nil
}

// SSHConfigPath returns the path of the user's main ssh config: ~/.ssh/config.
func SSHConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".ssh", "config"), nil
}

// HasSSHConfigInclude reports whether ~/.ssh/config includes a config.d
// directory. It never modifies the ssh config. A missing config file is
// reported as not including it.
func HasSSHConfigInclude() (configPath string, present bool, err error) {
	configPath, err = SSHConfigPath()
	if err != nil {
		return "", false, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return configPath, false, nil
		}
		return configPath, false, err
	}

	return configPath, containsConfigDirInclude(string(data)), nil
}

// AddSSHConfigInclude adds the config.d include directive to ~/.ssh/config if
// it is not present yet. The directive is prepended, since ssh applies the
// first matching option and includes must not be swallowed by a preceding Host
// block. It reports the ssh config path and whether the directive was added.
func AddSSHConfigInclude() (configPath string, added bool, err error) {
	configPath, err = SSHConfigPath()
	if err != nil {
		return "", false, err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return configPath, false, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return configPath, false, err
	}

	if containsConfigDirInclude(string(data)) {
		return configPath, false, nil
	}

	content := SSHConfigIncludeDirective + "\n"
	if len(data) > 0 {
		content += "\n" + string(data)
	}

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		return configPath, false, err
	}

	return configPath, true, nil
}

func containsConfigDirInclude(content string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || !strings.EqualFold(fields[0], "Include") {
			continue
		}
		if slices.ContainsFunc(fields[1:], includesConfigDir) {
			return true
		}
	}
	return false
}

func includesConfigDir(pattern string) bool {
	pattern = strings.Trim(pattern, `"`)
	pattern = strings.ReplaceAll(pattern, `\`, "/")
	dir := strings.TrimSuffix(path.Dir(pattern), "/")

	return strings.HasSuffix(dir, "config.d")
}
