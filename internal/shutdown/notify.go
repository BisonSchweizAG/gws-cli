//go:build unix

package shutdown

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// notifyShutdownInitiated is posted by the macOS loginwindow when the user selects "Shut Down".
// Restarts post "com.apple.system.loginwindow.restartinitiated" and are therefore ignored.
const notifyShutdownInitiated = "com.apple.system.loginwindow.shutdownInitiated"

// notifyScript runs notifyutil ($0) for the key ($1) and kills it as soon as stdin is closed,
// which happens when waitForNotification returns or when the gws process dies. This guarantees
// that no orphaned notifyutil process is left behind.
const notifyScript = `exec 3<&0
"$0" -R -w "$1" &
pid=$!
{ read -r _ <&3; kill "$pid"; } >/dev/null 2>&1 &
wait "$pid"`

// waitForNotification blocks until notifyutil reports a notification for key.
func waitForNotification(notifyutil, key string) error {
	// The process lifetime is bound to stdin (see notifyScript), no cancellation via context needed.
	cmd := exec.CommandContext(context.Background(), "/bin/sh", "-c", notifyScript, notifyutil, key)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("notifyutil stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("notifyutil stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start notifyutil: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == key {
			return nil
		}
	}
	return errors.New("notifyutil exited")
}
