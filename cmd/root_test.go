package cmd

import (
	"testing"
)

func TestNoLaunchBrowserFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("no-launch-browser")
	if flag == nil {
		t.Fatal("expected flag --no-launch-browser to be registered on rootCmd")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("expected flag type bool, got %s", flag.Value.Type())
	}
	if flag.DefValue != "false" {
		t.Errorf("expected default value false, got %s", flag.DefValue)
	}
}
