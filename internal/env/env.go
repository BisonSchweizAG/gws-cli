package env

import (
	"os"
	"regexp"
)

func ExpandEnv(s string) string {
	// Replace %VAR% with $VAR for compatibility with os.ExpandEnv
	re := regexp.MustCompile(`%([A-Za-z0-9_]+)%`)
	s = re.ReplaceAllString(s, `$$${1}`) // Replace %VAR% with $VAR
	return os.ExpandEnv(s)
}
