package script

import (
	"github.com/bisonschweizag/gws-cli/internal/types"

	_ "embed"
)

//go:embed ssh-config.tmpl
var sshConfigTemplate string

func SSHConfig(cfg *types.Config, fileName string) (name string, content []byte, err error) {
	if fileName == "" {
		fileName = "gws-ssh-config-" + cfg.CurrentContextName
	}

	return renderNamed(sshConfigTemplate, cfg, fileName, "gws-ssh-config", "")
}
