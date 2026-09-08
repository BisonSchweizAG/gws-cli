package gcloud

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/phayes/freeport"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/bisonschweizag/gws-cli/icon"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

const (
	userAgent      = "google-cloud-sdk"
	OOBRedirectURI = "urn:ietf:wg:oauth:2.0:oob"
)

var (
	ClientID     = ""
	ClientSecret = ""

	authStdin io.Reader = os.Stdin

	oauthConfig = &oauth2.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/cloud-platform",
		},
		Endpoint: google.Endpoint,
	}
)

// GeneratePKCE generates PKCE Code Verifier and SHA-256 Code Challenge.
func GeneratePKCE() (codeVerifier, codeChallenge string, err error) {
	verifierBytes := make([]byte, 32)
	_, err = rand.Read(verifierBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	codeVerifier = base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Create the SHA-256 hash of the verifier
	hash := sha256.Sum256([]byte(codeVerifier))

	// Base64 URL encode the hash to create the code challenge
	codeChallenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return codeVerifier, codeChallenge, nil
}

// BuildAuthURL builds the authorization URL with PKCE and context settings.
func BuildAuthURL(cfg *types.Config, redirectURL, codeChallenge string) (authURL, state string) {
	localOAuth := *oauthConfig
	localOAuth.RedirectURL = redirectURL

	state = "state"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		state = base64.RawURLEncoding.EncodeToString(b)
	}

	options := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	if sshContext := cfg.CurrentContext(); sshContext != nil && sshContext.GCloud != nil && sshContext.GCloud.Account != "" {
		options = append(options, oauth2.SetAuthURLParam("login_hint", sshContext.GCloud.Account))
	} else {
		options = append(options, oauth2.SetAuthURLParam("prompt", "select_account"))
	}

	return localOAuth.AuthCodeURL(state, options...), state
}

// ExchangeAuthCode exchanges an authorization code for an OAuth2 token and persists it.
func ExchangeAuthCode(ctx context.Context, cfg *types.Config, code, codeVerifier, redirectURL string) (*oauth2.Token, error) {
	localOAuth := *oauthConfig
	localOAuth.RedirectURL = redirectURL

	token, err := localOAuth.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
		oauth2.SetAuthURLParam("client_secret", localOAuth.ClientSecret),
	)
	if err != nil {
		return nil, err
	}

	// Save token (may not contain a refresh token if consent not granted)
	if err := cfg.SetToken(*token); err != nil {
		// log save error but continue - we still return the token for in-memory usage
		log.Logf("Failed to persist token: %v", err)
	}

	// Warn if the refresh token was not provided.
	if token.RefreshToken == "" {
		log.Log("Warning: no refresh token returned. You may need to re-auth with prompt=consent to get a refresh token.")
	}

	return token, nil
}

func Login(ctx context.Context, cfg *types.Config) (oauth2.TokenSource, error) {
	if cfg == nil {
		return nil, errors.New("cfg must be provided")
	}

	httpClient := &http.Client{
		Transport: &userAgentTransport{
			base: http.DefaultTransport,
			ua:   userAgent,
		},
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	var existingToken oauth2.Token
	if cfg.Token != nil {
		existingToken = cfg.Token.Token
	}

	// Try refreshing the token
	if existingToken.RefreshToken != "" {
		tokenSource := oauthConfig.TokenSource(ctx, &existingToken)
		token, err := tokenSource.Token()
		if err == nil {
			_ = cfg.SetToken(*token)
			return newTokenSourceWithRefreshCheck(ctx, token, cfg), nil
		}
	}

	codeVerifier, codeChallenge, err := GeneratePKCE()
	if err != nil {
		return nil, err
	}

	if cfg.NoBrowser {
		return loginNoBrowser(ctx, cfg, codeVerifier, codeChallenge)
	}

	return loginBrowser(ctx, cfg, codeVerifier, codeChallenge)
}

func loginNoBrowser(ctx context.Context, cfg *types.Config, codeVerifier, codeChallenge string) (oauth2.TokenSource, error) {
	authURL, _ := BuildAuthURL(cfg, OOBRedirectURI, codeChallenge)

	linkStyle := lipgloss.NewStyle().Underline(true).Hyperlink(authURL)
	fmt.Printf("Go to the following link in your browser:\n\n%s\n\n", linkStyle.Render(authURL))
	fmt.Print("Enter authorization code: ")

	reader := bufio.NewReader(authStdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %w", err)
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("authorization code cannot be empty")
	}

	token, err := ExchangeAuthCode(ctx, cfg, code, codeVerifier, OOBRedirectURI)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	return newTokenSourceWithRefreshCheck(ctx, token, cfg), nil
}

func loginBrowser(ctx context.Context, cfg *types.Config, codeVerifier, codeChallenge string) (oauth2.TokenSource, error) {
	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}

	// Use a per-request copy so we don't race other callers that may rely on oauthConfig
	//nolint:revive // http is ok for a local callback
	redirectURL := fmt.Sprintf("http://%s/", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	authURL, state := BuildAuthURL(cfg, redirectURL, codeChallenge)

	// Open URL in browser
	log.Log("Opening URL: " + authURL)
	openBrowser(ctx, cfg, authURL)

	// Create a channel for shutdown signaling that can carry a token or an error.
	type authResult struct {
		token *oauth2.Token
		err   error
	}
	shutdownChan := make(chan authResult, 1)

	// Use a dedicated ServeMux to avoid interfering with global handlers.
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadHeaderTimeout: 1 * time.Second,
		Handler:           mux,
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("state") != state {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			shutdownChan <- authResult{nil, errors.New("invalid state")}
			return
		}
		code := query.Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			shutdownChan <- authResult{nil, errors.New("missing code")}
			return
		}

		token, err := ExchangeAuthCode(ctx, cfg, code, codeVerifier, redirectURL)
		if err != nil {
			http.Error(w, "Failed to get token", http.StatusInternalServerError)
			log.Logf("🚨 OAuth exchange error: %v", err)
			shutdownChan <- authResult{nil, err}
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, callbackHTML())
		shutdownChan <- authResult{token, nil}
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Report start-up failure back
			shutdownChan <- authResult{nil, fmt.Errorf("failed to start server: %w", err)}
		}
	}()

	log.Log("Waiting for authentication...")
	// Block until we receive a shutdown signal
	res := <-shutdownChan
	_ = server.Shutdown(ctx)
	if res.err != nil {
		return nil, res.err
	}
	log.Log("Authenticated...")

	return newTokenSourceWithRefreshCheck(ctx, res.token, cfg), nil
}

func callbackHTML() string {
	return fmt.Sprintf(`
<html>
<head><title>gws Google Authentication Successful</title></head>
<body style="font-family: sans-serif; text-align: center; padding-top: 50px;">
	<div style="width: 128px; height: 128px; margin: 0 auto 20px auto;">
%s
	</div>
	<h1>gws Google Authentication Successful!</h1>
	<p>You can now close this window and return to the tool.</p>
	<script>window.onload = function() { setTimeout(function() { window.close(); }, 1000); }</script>
</body>
</html>
`, icon.IconSVG)
}

type userAgentTransport struct {
	base http.RoundTripper
	ua   string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.ua)
	return t.base.RoundTrip(req)
}

type TokenSourceWithRefreshCheck struct {
	source      oauth2.TokenSource
	checkPeriod time.Duration
	lastToken   *oauth2.Token
	done        chan struct{}
	cancel      context.CancelFunc
	cfg         *types.Config
}

func newTokenSourceWithRefreshCheck(ctx context.Context, token *oauth2.Token, cfg *types.Config) oauth2.TokenSource {
	newCtx, cancel := context.WithCancel(ctx)
	ts := &TokenSourceWithRefreshCheck{
		checkPeriod: 10 * time.Minute,
		source:      oauthConfig.TokenSource(newCtx, token),
		cfg:         cfg,
		done:        make(chan struct{}),
		cancel:      cancel,
	}

	if cfg.TokenCheck {
		// Start periodic check using the cancelable context so Stop() cancels it.
		go ts.periodicCheck(newCtx)
	}
	return ts
}

func (ts *TokenSourceWithRefreshCheck) periodicCheck(ctx context.Context) {
	ticker := time.NewTicker(ts.checkPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ts.done:
			return
		case <-ticker.C:
			token, err := ts.source.Token()
			if err != nil {
				continue
			}

			if ts.lastToken == nil || ts.lastToken.AccessToken != token.AccessToken {
				_ = ts.cfg.SetToken(*token)
				ts.lastToken = token
				log.Log("🔑 Refreshed OAuth2 token")
			}
		}
	}
}

func (ts *TokenSourceWithRefreshCheck) Token() (*oauth2.Token, error) {
	token, err := ts.source.Token()
	if err != nil {
		return nil, err
	}

	// Also check for refresh during direct Token() calls
	if ts.lastToken == nil || ts.lastToken.AccessToken != token.AccessToken {
		_ = ts.cfg.SetToken(*token)
		ts.lastToken = token
		log.Log("🔑 Refreshed OAuth2 token")
	}

	return token, nil
}

// Stop stops the periodic check.
func (ts *TokenSourceWithRefreshCheck) Stop() {
	if ts.cancel != nil {
		ts.cancel()
	}
	// Close done non-blocking, protect against double close
	select {
	case <-ts.done:
		// already closed / drained
	default:
		close(ts.done)
	}
}
