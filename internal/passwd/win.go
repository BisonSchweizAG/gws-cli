//go:build windows

package passwd

import (
	"os"

	"golang.org/x/term"

	"github.com/bisonschweizag/gws-cli/internal/log"
)

func Prompt(prompt string) (string, error) {
	log.Logf("%s \n", prompt)
	key, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	return string(key), nil
}
