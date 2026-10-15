//go:build aix || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package gcloud

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

// OpenBrowser opens the given URL in the default browser.
func OpenBrowser(_ context.Context, cfg *types.Config, authURL string) {
	var cmd *exec.Cmd
	if IsWSL() {
		cmd = windowsCmd(context.Background(), cfg, authURL)
	} else {
		cmd = exec.CommandContext(context.Background(), "xdg-open", authURL)
	}
	_ = cmd.Start()
}

func IsWSL() bool {
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft") || strings.Contains(strings.ToLower(string(b)), "wsl")
}
