//go:build darwin

package shutdown

import (
	"fmt"
	"os"
)

const notifyutilPath = "/usr/bin/notifyutil"

// WaitForShutdown blocks until the user initiates a system shutdown (not a restart or logout).
// It listens for the loginwindow Darwin notification using notifyutil, so it works for
// builds without cgo (release builds use CGO_ENABLED=0).
//
// Note: the notification is posted when the shutdown is initiated, before apps are terminated.
// A shutdown via the command line (e.g. `sudo shutdown -h now`) is not detected.
func WaitForShutdown() (ReleaseFunc, error) {
	if _, err := os.Stat(notifyutilPath); err != nil {
		return nil, fmt.Errorf("shutdown detection not available: %w", err)
	}
	if err := waitForNotification(notifyutilPath, notifyShutdownInitiated); err != nil {
		return nil, err
	}
	return func() {}, nil
}
