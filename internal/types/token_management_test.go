package types

import (
	"os"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenManagement(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Ensure no token initially
	token, err := LoadToken()
	if err != nil {
		t.Fatalf("unexpected error loading token: %v", err)
	}
	if token != nil {
		t.Fatalf("expected nil token, got %+v", token)
	}

	testToken := oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(1 * time.Hour).Truncate(time.Second),
	}

	// Save token
	if err := SaveToken(testToken); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	tokenPath, err := GetTokenFilePath()
	if err != nil {
		t.Fatalf("unexpected error getting token file path: %v", err)
	}

	// Check if file exists in fallback location if keyring wasn't used or in addition
	// Load token back
	loadedToken, err := LoadToken()
	if err != nil {
		t.Fatalf("unexpected error loading saved token: %v", err)
	}
	if loadedToken == nil {
		t.Fatal("expected token to be loaded, got nil")
	}
	if loadedToken.AccessToken != testToken.AccessToken {
		t.Fatalf("expected access token %q, got %q", testToken.AccessToken, loadedToken.AccessToken)
	}

	// Delete token
	if err := DeleteToken(); err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Fatalf("expected token file to be deleted, but stat returned %v", err)
	}

	// Calling DeleteToken again when already deleted should not error
	if err := DeleteToken(); err != nil {
		t.Fatalf("expected DeleteToken on non-existent token to succeed, got %v", err)
	}

	// LoadToken should return nil now
	loadedToken, err = LoadToken()
	if err != nil {
		t.Fatalf("unexpected error loading token after deletion: %v", err)
	}
	if loadedToken != nil {
		t.Fatalf("expected nil token after deletion, got %+v", loadedToken)
	}
}

func TestConfig_DeleteToken(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	cfg := &Config{
		Token: &TokenStorage{
			Token: oauth2.Token{
				AccessToken: "test-token",
			},
		},
	}

	if err := cfg.DeleteToken(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Token != nil {
		t.Fatalf("expected cfg.Token to be nil, got %+v", cfg.Token)
	}
}
