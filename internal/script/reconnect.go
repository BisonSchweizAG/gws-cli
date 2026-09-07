package script

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/bisonschweizag/gws-cli/internal/types"

	_ "embed"
)

//go:embed reconnect-win.cmd
var windowsReconnectScript string

//go:embed reconnect-unix.sh
var unixReconnectScript string

func WindowsReconnectSSH(cfg *types.Config, fileName string) (name string, content []byte, err error) {
	return render(windowsReconnectScript, cfg, fileName, "cmd")
}

func UnixReconnectSSH(cfg *types.Config, fileName string) (name string, content []byte, err error) {
	return render(unixReconnectScript, cfg, fileName, "sh")
}

func render(script string, cfg *types.Config, fileName, fileExt string) (name string, content []byte, err error) {
	tmpl, err := template.New("reconnect").Parse(script)
	if err != nil {
		return "", nil, err
	}

	if fileName == "" {
		fileName = fmt.Sprintf("gws-ssh-with-reconnect-%s.%s", cfg.CurrentContextName, fileExt)
	}

	name = outFileName(fileName)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf,
		&NamedContext{
			Name:     cfg.CurrentContextName,
			FileName: name,
			Context:  *cfg.CurrentContext(),
		},
	); err != nil {
		return "", nil, err
	}

	return name, buf.Bytes(), nil
}

func outFileName(name string) string {
	if name == "" {
		return ""
	}
	return filepath.Base(name)
}

type NamedContext struct {
	types.Context
	Name     string
	FileName string
}
