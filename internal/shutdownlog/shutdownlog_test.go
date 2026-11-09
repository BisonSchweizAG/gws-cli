package shutdownlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestAppendCreatesAndAppends(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, time.November, 9, 13, 38, 0, 0, time.UTC)
	firstMessage := "context=dev workstation=ws result=stop requested"
	secondMessage := "context=prod workstation=ws2 result=stop requested"

	if err := Append(dir, firstMessage, now); err != nil {
		t.Fatalf("Append() first call error = %v", err)
	}
	if err := Append(dir, secondMessage, now.Add(time.Second)); err != nil {
		t.Fatalf("Append() second call error = %v", err)
	}

	path := filepath.Join(dir, "shutdown.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := now.Format(time.RFC3339) + " " + firstMessage + "\n" +
		now.Add(time.Second).Format(time.RFC3339) + " " + secondMessage + "\n"
	if got := string(data); got != want {
		t.Fatalf("shutdown.log = %q, want %q", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got, wantMode := info.Mode().Perm(), os.FileMode(0o600); got != wantMode {
		t.Errorf("shutdown.log mode = %04o, want %04o", got, wantMode)
	}
}

func TestAppendPrunesOldEntriesAndKeepsRecentAndUnparseableLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, time.November, 9, 13, 38, 0, 0, time.UTC)
	oldLine := now.Add(-8*24*time.Hour).Format(time.RFC3339) + " old entry"
	cutoffLine := now.Add(-7*24*time.Hour).Format(time.RFC3339) + " cutoff entry"
	recentLine := now.Add(-time.Hour).Format(time.RFC3339) + " recent entry"
	invalidLine := "legacy entry without timestamp"
	initial := oldLine + "\n" + cutoffLine + "\n" + recentLine + "\n" + invalidLine + "\n"
	if err := os.WriteFile(filepath.Join(dir, "shutdown.log"), []byte(initial), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	newMessage := "context=dev workstation=ws result=stop requested"
	if err := Append(dir, newMessage, now); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "shutdown.log"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := cutoffLine + "\n" + recentLine + "\n" + invalidLine + "\n" + now.Format(time.RFC3339) + " " + newMessage + "\n"
	if got := string(data); got != want {
		t.Fatalf("shutdown.log = %q, want %q", got, want)
	}
}

func TestDir(t *testing.T) {
	t.Parallel()

	customConfigPath := filepath.Join(t.TempDir(), "config.yaml")
	if got, want := Dir(&types.Config{FilePath: customConfigPath}), filepath.Dir(customConfigPath); got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}

	_, defaultConfigDir, _ := types.DefaultConfigPaths()
	if got := Dir(&types.Config{}); got != defaultConfigDir {
		t.Errorf("Dir() without config path = %q, want %q", got, defaultConfigDir)
	}
}
