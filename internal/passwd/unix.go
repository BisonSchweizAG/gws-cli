//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package passwd

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"

	"github.com/bisonschweizag/gws-cli/internal/log"
)

func Prompt(prompt string) (string, error) {
	// restore terminal state on interrupt https://github.com/golang/go/issues/31180
	oldState, err := term.GetState(syscall.Stdin)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(syscall.Stdin, oldState) }()

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	go func() {
		for range sigch {
			_ = term.Restore(syscall.Stdin, oldState)
			os.Exit(0)
		}
	}()

	log.Logf("%s \n", prompt)
	key, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	return string(key), nil
}
