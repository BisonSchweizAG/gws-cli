//go:build windows

package shutdown

import (
	"fmt"
	"syscall"
)

const (
	CTRL_LOGOFF_EVENT   = 5
	CTRL_SHUTDOWN_EVENT = 6
)

// WaitForShutdown blocks until the system is shutting down or the user is logging off.
// It uses SetConsoleCtrlHandler to detect CTRL_SHUTDOWN_EVENT and CTRL_LOGOFF_EVENT.
// Returns nil on shutdown/logoff, error if not attached to a console.
func WaitForShutdown() error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleCtrlHandler := kernel32.NewProc("SetConsoleCtrlHandler")

	// Channel signaled when shutdown or logoff is detected
	done := make(chan error, 1)

	// Register the control handler callback
	handler := syscall.NewCallback(func(ctrlType uint32) uint32 {
		switch ctrlType {
		case CTRL_SHUTDOWN_EVENT, CTRL_LOGOFF_EVENT:
			// System is shutting down or user is logging off
			done <- nil
			return 1 // TRUE: we handled it
		default:
			return 0 // FALSE: pass to next handler
		}
	})

	r, _, err := procSetConsoleCtrlHandler.Call(handler, 1)
	if r == 0 {
		return fmt.Errorf("failed to register console control handler: %w", err)
	}

	// Block until shutdown signal is received
	<-done
	return nil
}
