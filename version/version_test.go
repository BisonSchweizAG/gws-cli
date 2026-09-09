package version_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bisonschweizag/gws-cli/version"
)

func TestLogo(t *testing.T) {
	if version.Logo == "" {
		t.Fatal("expected Logo not to be empty")
	}
	if !strings.Contains(version.Logo, "\033[38;2;248;113;113m●\033[38;5;63m") {
		t.Error("expected Logo to contain red dot \\033[38;2;248;113;113m●\\033[38;5;63m")
	}
	if !strings.Contains(version.Logo, "\033[38;2;251;191;36m●\033[38;5;63m") {
		t.Error("expected Logo to contain yellow dot \\033[38;2;251;191;36m●\\033[38;5;63m")
	}
	if !strings.Contains(version.Logo, "\033[38;2;52;211;153m●\033[38;5;63m") {
		t.Error("expected Logo to contain green dot \\033[38;2;52;211;153m●\\033[38;5;63m")
	}
	if !strings.Contains(version.Logo, "\033[37m") {
		t.Error("expected Logo to color cloud in white")
	}

	// Verify alignment of box lines
	lines := strings.Split(strings.TrimRight(version.Logo, "\n"), "\n")
	// The box is lines 3 through 8 (0-indexed: 3=" ┌──────────────┐", 4=" │ ... │", 5=" ├──────────────┤", 6=" │ ... │", 7=" │ ... │", 8=" └──────────────┘")
	for i := 3; i < len(lines); i++ {
		w := lipgloss.Width(lines[i])
		if w != 17 {
			t.Errorf("expected line %d (%q) to have visual width 17, got %d", i, lines[i], w)
		}
	}
}
