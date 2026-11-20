//go:build !windows

// Package sessionend provides notifications for operating-system session shutdown.
package sessionend

// Listen returns a channel that is notified when the current session is ending
// and a function that releases the listener. On platforms without session-end
// notifications, the channel is nil and the release function is a no-op.
func Listen() (<-chan struct{}, func()) {
	return nil, func() {}
}
