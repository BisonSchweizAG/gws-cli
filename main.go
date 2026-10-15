package main

import (
	"os"
	"runtime"

	"github.com/bisonschweizag/gws-cli/cmd"
)

func main() {
	// Workaround for Termux on Android where the binary path is injected as an extra argument.
	// See: https://github.com/termux/termux-app/issues/4630
	if runtime.GOOS == "android" && len(os.Args) >= 2 {
		execPath := os.Args[0]
		firstArg := os.Args[1]
		// If the first argument looks like the binary path, remove it
		if firstArg == execPath || (len(firstArg) > 1 && firstArg[0] == '/' && endsWith(firstArg, execPath)) {
			os.Args = append(os.Args[:1], os.Args[2:]...)
		}
	}
	cmd.Execute()
}

// endsWith returns true if s ends with suffix, comparing the last len(suffix) bytes.
func endsWith(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
