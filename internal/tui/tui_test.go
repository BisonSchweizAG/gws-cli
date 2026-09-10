package tui_test

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bisonschweizag/gws-cli/internal/tui"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestTUIView(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
		Contexts: map[string]*types.Context{
			"default": {
				GCloud: &types.GCloud{
					Name: "my-workstation",
				},
				Port: 2222,
			},
		},
	}
	_ = cfg.UseContext("default")

	for _, w := range []int{80, 100, 120} {
		m := tui.NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
			return nil
		})
		m.Width = w
		m.Height = 30
		m.AddHeader("Local Port", "2222")
		view := m.View()

		// Verify that the view contains version.Logo elements and header
		if !strings.Contains(view.Content, "GWS Tunnel") {
			t.Errorf("View missing title at width %d", w)
		}
		if !strings.Contains(view.Content, "my-workstation") {
			t.Errorf("View missing workstation at width %d", w)
		}

		t.Logf("Width %d rendered view:\n%s\n", w, view.Content)
	}
}

func TestStylesLogo(t *testing.T) {
	styles := tui.DefaultStyles()
	if styles.Logo.GetForeground() != tui.Indigo {
		t.Errorf("Expected Logo style foreground to be %v, got %v", tui.Indigo, styles.Logo.GetForeground())
	}
}

func TestTUIView_AuthURL(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := tui.NewModel(context.Background(), cfg, ">_ GWS Login", func(context.Context) error {
		return nil
	})
	m.Width = 100
	m.Height = 40
	m.AuthURL = "https://accounts.google.com/o/oauth2/auth?client_id=123"
	m.ClipboardMsg = "✓ Auth URL copied to clipboard!"

	view := m.View()
	if !strings.Contains(view.Content, "AUTHENTICATION REQUIRED") {
		t.Error("View missing AUTHENTICATION REQUIRED card")
	}
	if !strings.Contains(view.Content, "https://accounts.google.com/o/oauth2/auth?client_id=123") {
		t.Error("View missing Auth URL")
	}
	if !strings.Contains(view.Content, "copied to clipboard") {
		t.Error("View missing clipboard notice")
	}
	if !strings.Contains(view.Content, "Ctrl+Y") {
		t.Error("View missing Ctrl+Y help")
	}
}

func TestTUI_ClipboardCopy(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := tui.NewModel(context.Background(), cfg, ">_ GWS Login", func(context.Context) error {
		return nil
	})
	m.AuthURL = "https://accounts.google.com/o/oauth2/auth?client_id=123"

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	m2, ok := newModel.(*tui.Model)
	if !ok {
		t.Fatalf("expected newModel to be *tui.Model, got %T", newModel)
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
