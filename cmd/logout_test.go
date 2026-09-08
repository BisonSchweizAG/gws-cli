package cmd

import (
	"bytes"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestLogoutCmd(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	testToken := oauth2.Token{
		AccessToken:  "logout-test-access-token",
		RefreshToken: "logout-test-refresh-token",
		Expiry:       time.Now().Add(1 * time.Hour),
	}

	if err := types.SaveToken(testToken); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Verify token is present
	tok, err := types.LoadToken()
	if err != nil || tok == nil {
		t.Fatalf("expected token to exist before logout, got err=%v tok=%v", err, tok)
	}

	// Execute logout command
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"logout"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("logout command failed: %v", err)
	}

	// Verify token is deleted
	tok, err = types.LoadToken()
	if err != nil {
		t.Fatalf("unexpected error loading token after logout: %v", err)
	}
	if tok != nil {
		t.Fatalf("expected token to be nil after logout, got %+v", tok)
	}
}
