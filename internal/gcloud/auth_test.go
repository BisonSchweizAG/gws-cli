package gcloud

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bisonschweizag/gws-cli/docs/icon"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestCallbackHTML(t *testing.T) {
	html := callbackHTML()

	if !strings.Contains(html, icon.IconSVG) {
		t.Error("expected callbackHTML to contain icon.IconSVG")
	}
	if !strings.Contains(html, "<svg") {
		t.Error("expected callbackHTML to contain '<svg'")
	}
	if !strings.Contains(html, "gws Google Authentication Successful!") {
		t.Error("expected callbackHTML to contain success message")
	}
	if !strings.Contains(html, "prefers-color-scheme: dark") {
		t.Error("expected callbackHTML to support dark theme via prefers-color-scheme")
	}
	if !strings.Contains(html, `content="light dark"`) {
		t.Error("expected callbackHTML to declare light dark color-scheme meta")
	}
	if !strings.Contains(html, "icon.ico") {
		t.Error("expected callbackHTML to link icon.ico as favicon")
	}
}

func TestLogin_Browser_AuthURL(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	authURLChan := make(chan string, 1)
	cfg := &types.Config{
		AuthURLChan: authURLChan,
	}

	errChan := make(chan error, 1)
	go func() {
		_, err := Login(ctx, cfg)
		errChan <- err
	}()

	var generatedURL string
	select {
	case generatedURL = <-authURLChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for auth URL")
	}

	cancel()
	<-errChan

	parsedURL, err := url.Parse(generatedURL)
	if err != nil {
		t.Fatalf("failed to parse generated auth URL: %v", err)
	}

	if parsedURL.Scheme != "https" || parsedURL.Host != "accounts.google.com" || parsedURL.Path != "/o/oauth2/auth" {
		t.Errorf("unexpected endpoint: %s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)
	}

	q := parsedURL.Query()
	if got := q.Get("client_id"); got != ClientID {
		t.Errorf("client_id = %q, want %q", got, ClientID)
	}
	if got := q.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q, want %q", got, "code")
	}
	if got := q.Get("access_type"); got != "offline" {
		t.Errorf("access_type = %q, want %q", got, "offline")
	}
	if got := q.Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q, want %q", got, "S256")
	}
	if q.Get("code_challenge") == "" {
		t.Error("code_challenge should not be empty")
	}
	redirectURI := q.Get("redirect_uri")
	if !strings.HasPrefix(redirectURI, "http://localhost:") {
		t.Errorf("redirect_uri should start with 'http://localhost:', got %q", redirectURI)
	}
	if got := q.Get("prompt"); got != "select_account" {
		t.Errorf("prompt = %q, want %q", got, "select_account")
	}

	expectedScopes := strings.Join(CloudScopes, " ")
	if got := q.Get("scope"); got != expectedScopes {
		t.Errorf("scope = %q, want %q", got, expectedScopes)
	}
}
