//go:build unix

package shutdown

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fakeNotifyutil writes a script that mimics `notifyutil -R -w <key>`.
func fakeNotifyutil(t *testing.T, body string) (bin, pidFile string) {
	t.Helper()
	dir := t.TempDir()
	pidFile = filepath.Join(dir, "pid")
	bin = filepath.Join(dir, "notifyutil")
	script := "#!/bin/sh\necho $$ > " + pidFile + "\n" + body + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin, pidFile
}

func runWithTimeout(t *testing.T, bin string) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- waitForNotification(bin, notifyShutdownInitiated) }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("waitForNotification did not return")
		return nil
	}
}

func TestWaitForNotification_Detected(t *testing.T) {
	// print unrelated keys first, then the expected one and keep running like notifyutil -w does
	bin, pidFile := fakeNotifyutil(t, `[ "$1" = "-R" ] && [ "$2" = "-w" ] || exit 1
echo com.apple.system.loginwindow.restartinitiated
echo "$3"
exec sleep 60`)

	if err := runWithTimeout(t, bin); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// the notifyutil process must be killed once we return
	pid, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.Atoi(strings.TrimSpace(string(pid)))
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(p, 0); err == nil {
		t.Errorf("notifyutil process %d still running", p)
	}
}

func TestWaitForNotification_Exited(t *testing.T) {
	bin, _ := fakeNotifyutil(t, `echo some.other.key
exit 1`)

	if err := runWithTimeout(t, bin); err == nil {
		t.Fatal("expected error when notifyutil exits")
	}
}

func TestWaitForNotification_MissingBinary(t *testing.T) {
	if err := runWithTimeout(t, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error for missing notifyutil")
	}
}
