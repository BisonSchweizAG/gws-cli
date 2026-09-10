package setup_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bisonschweizag/gws-cli/internal/setup"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestSetupView_LoginStep(t *testing.T) {
	cfg := &types.Config{
		CurrentContextName: "default",
	}

	m := setup.InitialModel(cfg)
	m.Width = 100
	m.Height = 35
	m.AuthURL = "https://accounts.google.com/o/oauth2/auth?client_id=setup_browser_test"

	view := m.View()
	content := view.Content

	if !strings.Contains(content, "STEP 1/4") {
		t.Error("expected STEP 1/4 indicator")
	}
	if !strings.Contains(content, "Opening Google Cloud authentication in your default browser") {
		t.Error("expected default browser info message")
	}
	if !strings.Contains(content, "https://accounts.google.com/o/oauth2/auth?client_id=setup_browser_test") {
		t.Error("expected auth URL in browser mode view")
	}
	if !strings.Contains(content, "Ctrl+Y") {
		t.Error("expected Ctrl+Y copy URL help hint in browser mode")
	}
}

func TestSetupView_FormStep(t *testing.T) {
	cfg := &types.Config{
		CurrentContextName: "test-ctx",
		Contexts: map[string]*types.Context{
			"test-ctx": {
				User: "dev-user",
				Port: 22022,
			},
		},
	}

	m := setup.InitialModel(cfg)
	m.Step = 3 // stepForm
	m.Width = 100
	m.Height = 40

	view := m.View()
	content := view.Content

	if !strings.Contains(content, "STEP 4/4") {
		t.Error("expected STEP 4/4 indicator")
	}
	if !strings.Contains(content, "Configure GWS Context") {
		t.Error("expected Configure GWS Context title")
	}
	if !strings.Contains(content, "Submit Configuration") {
		t.Error("expected Submit Configuration button")
	}
}

func TestSetupView_SelectionSteps(t *testing.T) {
	cfg := &types.Config{}
	m := setup.InitialModel(cfg)
	m.Step = 1 // stepProject
	m.Width = 100
	m.Height = 30

	view := m.View()
	if !strings.Contains(view.Content, "STEP 2/4") {
		t.Error("expected STEP 2/4 indicator")
	}
	if !strings.Contains(view.Content, "Select Google Cloud Project") {
		t.Error("expected Select Google Cloud Project title")
	}
	if !strings.Contains(view.Content, "Filter") {
		t.Error("expected Filter prompt")
	}

	m.Step = 2 // stepConfig
	view = m.View()
	if !strings.Contains(view.Content, "STEP 3/4") {
		t.Error("expected STEP 3/4 indicator")
	}
	if !strings.Contains(view.Content, "Select Cloud Workstation") {
		t.Error("expected Select Cloud Workstation title")
	}
}

func TestSetup_ClipboardCopy(t *testing.T) {
	cfg := &types.Config{
		CurrentContextName: "default",
	}

	m := setup.InitialModel(cfg)
	m.AuthURL = "https://accounts.google.com/o/oauth2/auth?client_id=setup_clipboard_test"

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	m2, ok := newModel.(setup.Model)
	if !ok {
		t.Fatalf("expected newModel to be setup.Model, got %T", newModel)
	}
	if !m2.CopiedToClipboard {
		t.Error("expected CopiedToClipboard to be true")
	}
	if !strings.Contains(m2.ClipboardMsg, "copied to clipboard") {
		t.Errorf("unexpected ClipboardMsg: %s", m2.ClipboardMsg)
	}
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd for SetClipboard")
	}
}
