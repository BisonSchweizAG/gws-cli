//go:build !linux && !darwin && !windows

package shutdown

import "fmt"

// WaitForShutdown blocks until the system is shutting down.
// This is a stub implementation for platforms that don't support
// system shutdown detection (Windows, etc.).
func WaitForShutdown() error {
	return fmt.Errorf("shutdown detection is not supported on this platform")
}
