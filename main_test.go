package main

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// widthAmbiguousIcons are runes that terminals render with differing widths,
// which shifts TUI borders on some Linux environments.
var widthAmbiguousIcons = map[rune]string{
	'\uFE0F': "variation selector-16 (emoji presentation)",
	'\uFE0E': "variation selector-15 (text presentation)",
	'\u200D': "zero width joiner",
	'\u26A0': "warning sign",
	'\u2699': "gear",
}

func TestGoFilesHaveNoWidthAmbiguousIcons(t *testing.T) {
	skipDirs := map[string]bool{"bin": true, "dist": true, "vendor": true, "testdata": true}

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != "." && (skipDirs[name] || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for lineNo := 1; scanner.Scan(); lineNo++ {
			for _, r := range scanner.Text() {
				if desc, ok := widthAmbiguousIcons[r]; ok {
					t.Errorf("%s:%d contains %U (%s)", path, lineNo, r, desc)
				}
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatalf("walk go files: %v", err)
	}
}
