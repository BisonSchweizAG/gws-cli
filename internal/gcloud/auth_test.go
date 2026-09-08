package gcloud

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/bisonschweizag/gws-cli/icon"
	"github.com/bisonschweizag/gws-cli/internal/log"
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
}

func TestGeneratePKCE(t *testing.T) {
	v1, c1, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v1 == "" || c1 == "" {
		t.Errorf("expected non-empty PKCE pair, got verifier=%q, challenge=%q", v1, c1)
	}

	v2, c2, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v1 == v2 || c1 == c2 {
		t.Error("expected distinct PKCE values on each call")
	}
}

func TestBuildAuthURL(t *testing.T) {
	cfg := &types.Config{}
	authURL, state := BuildAuthURL(cfg, OOBRedirectURI, "test-challenge")
	if !strings.Contains(authURL, "redirect_uri=urn%3Aietf%3Awg%3Aoauth%3A2.0%3Aoob") {
		t.Errorf("expected authURL to contain OOB redirect URI, got %s", authURL)
	}
	if !strings.Contains(authURL, "code_challenge=test-challenge") {
		t.Errorf("expected authURL to contain code_challenge, got %s", authURL)
	}
	if !strings.Contains(authURL, "state="+state) {
		t.Errorf("expected authURL to contain state, got %s", authURL)
	}
}

func TestLoginNoBrowser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			code := r.Form.Get("code")
			if code == "test-auth-code" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"access_token": "mock-access-token",
					"token_type": "Bearer",
					"expires_in": 3600,
					"refresh_token": "mock-refresh-token"
				}`))
				return
			}
			http.Error(w, "invalid code", http.StatusBadRequest)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	origOAuthConfig := *oauthConfig
	defer func() { oauthConfig = &origOAuthConfig }()

	oauthConfig = &oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  server.URL + "/auth",
			TokenURL: server.URL + "/token",
		},
	}

	origStdin := authStdin
	defer func() { authStdin = origStdin }()

	authStdin = strings.NewReader("test-auth-code\n")

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	origLogger := log.GetLogger()
	log.SetLogger(log.Null)
	defer log.SetLogger(origLogger)

	cfg := &types.Config{NoBrowser: true}
	ts, err := Login(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error during Login: %v", err)
	}

	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error getting token: %v", err)
	}
	if tok.AccessToken != "mock-access-token" {
		t.Errorf("expected token mock-access-token, got %s", tok.AccessToken)
	}
}
