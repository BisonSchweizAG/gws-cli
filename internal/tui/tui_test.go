package tui_test

import (
	"context"
	"strings"
	"testing"

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
