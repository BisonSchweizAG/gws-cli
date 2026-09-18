package gcloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

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

func TestLogin_NoLaunchBrowser_AuthURL(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	authURLChan := make(chan string, 1)
	cfg := &types.Config{
		AuthURLChan:     authURLChan,
		NoLaunchBrowser: true,
	}

	origStdin := stdinReader
	defer func() { stdinReader = origStdin }()
	stdinReader = strings.NewReader("")

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

	q := parsedURL.Query()
	if got := q.Get("redirect_uri"); got != RemoteCallbackURL {
		t.Errorf("redirect_uri = %q, want %q", got, RemoteCallbackURL)
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
}

func TestPromptVerificationCode(t *testing.T) {
	t.Run("valid code", func(t *testing.T) {
		r := strings.NewReader("4/0Axxxxxxxxxxxx\n")
		code, err := promptVerificationCode(context.Background(), r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "4/0Axxxxxxxxxxxx" {
			t.Errorf("got code %q, want %q", code, "4/0Axxxxxxxxxxxx")
		}
	})

	t.Run("code with whitespace", func(t *testing.T) {
		r := strings.NewReader("   4/0Axxxxxxxxxxxx   \n")
		code, err := promptVerificationCode(context.Background(), r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "4/0Axxxxxxxxxxxx" {
			t.Errorf("got code %q, want %q", code, "4/0Axxxxxxxxxxxx")
		}
	})

	t.Run("full callback URL with code query param", func(t *testing.T) {
		r := strings.NewReader(
			"https://bisonschweizag.github.io/gws-cli/callback/callback.html?code=4%2F0Atestcode&state=xyz\n",
		)
		code, err := promptVerificationCode(context.Background(), r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "4/0Atestcode" {
			t.Errorf("got code %q, want %q", code, "4/0Atestcode")
		}
	})

	t.Run("empty code EOF", func(t *testing.T) {
		r := strings.NewReader("")
		_, err := promptVerificationCode(context.Background(), r)
		if err == nil {
			t.Error("expected error for empty EOF reader, got nil")
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		pr, _ := io.Pipe()
		defer pr.Close()
		_, err := promptVerificationCode(ctx, pr)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestLogin_NoLaunchBrowser_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = r.ParseForm()
			if r.Form.Get("grant_type") != "authorization_code" {
				http.Error(w, "invalid grant type", http.StatusBadRequest)
				return
			}
			if r.Form.Get("code") != "mock-code-123" {
				http.Error(w, "invalid code", http.StatusBadRequest)
				return
			}
			if r.Form.Get("redirect_uri") != RemoteCallbackURL {
				http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token": "mock-access-token",
				"token_type": "Bearer",
				"refresh_token": "mock-refresh-token",
				"expires_in": 3600
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	origEndpoint := oauthConfig.Endpoint
	origStdin := stdinReader
	defer func() {
		oauthConfig.Endpoint = origEndpoint
		stdinReader = origStdin
	}()

	oauthConfig.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	stdinReader = strings.NewReader("mock-code-123\n")

	cfg := &types.Config{
		NoLaunchBrowser: true,
	}

	tokenSource, err := Login(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	tok, err := tokenSource.Token()
	if err != nil {
		t.Fatalf("failed to get token from source: %v", err)
	}
	if tok.AccessToken != "mock-access-token" {
		t.Errorf("got access token %q, want %q", tok.AccessToken, "mock-access-token")
	}
	if tok.RefreshToken != "mock-refresh-token" {
		t.Errorf("got refresh token %q, want %q", tok.RefreshToken, "mock-refresh-token")
	}
	if cfg.Token == nil || cfg.Token.Token.AccessToken != "mock-access-token" {
		t.Error("expected token to be saved in cfg")
	}
}

func TestLogin_NoLaunchBrowser_AuthCodeChan(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = r.ParseForm()
			if r.Form.Get("code") != "chan-code-456" {
				http.Error(w, "invalid code", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token": "mock-chan-access-token",
				"token_type": "Bearer",
				"expires_in": 3600
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	origEndpoint := oauthConfig.Endpoint
	defer func() {
		oauthConfig.Endpoint = origEndpoint
	}()

	oauthConfig.Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth",
		TokenURL: ts.URL + "/token",
	}

	cfg := &types.Config{
		NoLaunchBrowser: true,
	}
	cfg.InitAuthChannels()

	go func() {
		// Simulate TUI sending code through channel after URL is received
		<-cfg.AuthURLChan
		cfg.SendAuthCode("https://bisonschweizag.github.io/gws-cli/callback/callback.html?code=chan-code-456&state=xyz")
	}()

	tokenSource, err := Login(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	tok, err := tokenSource.Token()
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}
	if tok.AccessToken != "mock-chan-access-token" {
		t.Errorf("got access token %q, want %q", tok.AccessToken, "mock-chan-access-token")
	}
}
