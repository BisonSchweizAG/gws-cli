package icon

import (
	"strings"
	"testing"
)

func TestIconSVG(t *testing.T) {
	if IconSVG == "" {
		t.Fatal("expected IconSVG not to be empty")
	}
	if !strings.HasPrefix(IconSVG, "<svg") {
		t.Fatalf("expected IconSVG to start with '<svg', got %q", IconSVG[:min(len(IconSVG), 20)])
	}
}
