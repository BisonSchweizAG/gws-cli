package setup

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestSetup_NoBrowser(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	cfg := &types.Config{
		NoBrowser: true,
	}

	m := InitialModel(cfg)
	if m.Step != stepLogin {
		t.Fatalf("expected stepLogin, got %v", m.Step)
	}
	if m.AuthURL == "" {
		t.Fatal("expected non-empty AuthURL")
	}
	if m.CodeVerifier == "" {
		t.Fatal("expected non-empty CodeVerifier")
	}
	if !strings.Contains(m.AuthURL, "urn%3Aietf%3Awg%3Aoauth%3A2.0%3Aoob") {
		t.Errorf("expected AuthURL to contain OOB redirect URI, got %s", m.AuthURL)
	}

	m.Width = 80
	view := m.View().Content
	if !strings.Contains(view, "Go to the following link in your browser") {
		t.Errorf("expected view to contain instructions, got %s", view)
	}
	if !strings.Contains(view, m.AuthURL) {
		t.Errorf("expected view to contain AuthURL, got %s", view)
	}
	if !strings.Contains(view, "Enter authorization code:") {
		t.Errorf("expected view to contain code prompt, got %s", view)
	}
	// Verify link is printed outside/below the framed box
	if strings.Index(view, "╰") > strings.Index(view, "Go to the following link in your browser") {
		t.Error("expected link to be outside (below) the framed box")
	}

	// Test WindowSizeMsg updates width
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("expected Model type")
	}
	if m.Width != 80 {
		t.Errorf("expected m.Width to be 80, got %d", m.Width)
	}
	viewAfterResize := m.View().Content
	if !strings.Contains(viewAfterResize, m.AuthURL) {
		t.Errorf("expected view to contain AuthURL, got %s", viewAfterResize)
	}

	// Test Esc key aborts
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	resModel, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("expected Model type")
	}
	if !resModel.Aborted {
		t.Error("expected Aborted to be true after ESC")
	}
	_ = cmd
}

func TestSetup_Browser(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	cfg := &types.Config{
		NoBrowser: false,
	}

	m := InitialModel(cfg)
	if m.Step != stepLogin {
		t.Fatalf("expected stepLogin, got %v", m.Step)
	}

	m.Width = 80
	viewBefore := m.View().Content
	if !strings.Contains(viewBefore, "Please login in your browser...") {
		t.Errorf("expected view to contain browser login message, got %s", viewBefore)
	}
	if strings.Contains(viewBefore, "Go to the following link in your browser") {
		t.Error("expected no link before AuthURL is set")
	}

	testURL := "https://accounts.google.com/o/oauth2/auth?client_id=12345&redirect_uri=http%3A%2F%2F127.0.0.1%3A12345%2F&state=abc"
	updatedModel, _ := m.Update(logMsg("Opening URL: " + testURL))
	m, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("expected Model type")
	}
	if m.AuthURL != testURL {
		t.Errorf("expected AuthURL %s, got %s", testURL, m.AuthURL)
	}

	// Verify log message is "Opening browser..." without containing the raw URL
	foundCleanLog := false
	for _, l := range m.Logs {
		if l == "Opening browser..." {
			foundCleanLog = true
		}
		if strings.Contains(l, testURL) {
			t.Errorf("expected logs to not contain raw URL, found: %s", l)
		}
	}
	if !foundCleanLog {
		t.Errorf("expected logs to contain 'Opening browser...', got %v", m.Logs)
	}

	viewAfter := m.View().Content
	if !strings.Contains(viewAfter, "Go to the following link in your browser") {
		t.Errorf("expected view to contain instructions, got %s", viewAfter)
	}
	// Verify instructions appear exactly once
	if strings.Count(viewAfter, "Go to the following link in your browser") != 1 {
		t.Errorf(
			"expected exactly 1 link prompt in view, got %d",
			strings.Count(viewAfter, "Go to the following link in your browser"),
		)
	}
	// Verify link is printed outside/below the framed box as a continuous string
	if !strings.Contains(viewAfter, testURL) {
		t.Errorf("expected view to contain testURL, got %s", viewAfter)
	}
	if strings.Index(viewAfter, "╰") > strings.Index(viewAfter, "Go to the following link in your browser") {
		t.Error("expected link to be outside (below) the framed box")
	}
}
