package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bisonschweizag/gws-cli/cmd"
)

func main() {
	// Workaround for Termux on Android where the binary path is injected as an extra argument.
	// See: https://github.com/termux/termux-app/issues/4630
	// On Termux, os.Args[0] may be "./gws" while the injected arg is the absolute path
	// "/data/data/com.termux/files/home/gws". We need to detect this by comparing
	// the basename of the injected path with the basename of os.Args[0].
	if runtime.GOOS == "android" {
		os.Args = stripTermuxDuplicateArg(os.Args)
	}
	cmd.Execute()
}

// stripTermuxDuplicateArg removes the injected binary path from os.Args on Termux.
// The bug injects the absolute path of the binary as an extra argument, usually
// at position 1 (right after os.Args[0]).
func stripTermuxDuplicateArg(args []string) []string {
	if len(args) < 2 {
		return args
	}

	execBase := filepath.Base(args[0])

	// Check if args[1] looks like an absolute path to the same binary
	first := args[1]
	if !strings.HasPrefix(first, "/") {
		return args
	}

	// Compare basenames: if args[1] is an absolute path ending with the same
	// basename as args[0], it's the injected duplicate.
	if filepath.Base(first) == execBase {
		return append(args[:1], args[2:]...)
	}

	return args
}
