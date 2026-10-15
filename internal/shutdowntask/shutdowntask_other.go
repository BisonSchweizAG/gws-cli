//go:build !windows

package shutdowntask

import "errors"

// registerShutdownTask is a stub for non-Windows platforms.
func registerShutdownTask() error {
	return errors.New("shutdown-task is only available on Windows")
}

// unregisterShutdownTask is a stub for non-Windows platforms.
func unregisterShutdownTask() error {
	return errors.New("shutdown-task is only available on Windows")
}

// shutdownTaskStatus is a stub for non-Windows platforms.
func shutdownTaskStatus() error {
	return errors.New("shutdown-task is only available on Windows")
}
