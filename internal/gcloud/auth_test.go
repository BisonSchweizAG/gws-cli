package gcloud

import (
	"strings"
	"testing"

	"github.com/bisonschweizag/gws-cli/icon"
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
