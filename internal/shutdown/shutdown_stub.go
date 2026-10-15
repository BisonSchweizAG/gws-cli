//go:build !linux && !darwin && !windows

package shutdown

import "errors"

// WaitForShutdown is a stub implementation for platforms that don't support
// system shutdown detection.
func WaitForShutdown() (ReleaseFunc, error) {
	return nil, errors.New("shutdown detection is not supported on this platform")
}
