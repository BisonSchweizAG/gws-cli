//go:build linux

package shutdown

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

// WaitForShutdown blocks until the system is shutting down (power off, not reboot).
// It registers a delay inhibitor lock with systemd-logind, then listens for the
// PrepareForShutdown signal via D-Bus. The inhibitor lock is required so that
// logind actually sends the signal to this process.
func WaitForShutdown() error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("connect to system bus: %w", err)
	}
	defer conn.Close()

	logind := conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")

	// Register a delay inhibitor lock so logind sends PrepareForShutdown to this process.
	// Without this, the signal is not delivered on many systems (e.g. Ubuntu).
	var fd dbus.UnixFD
	err = logind.Call(
		"org.freedesktop.login1.Manager.Inhibit", 0,
		"shutdown",          // what
		"GWS CLI",           // who
		"Graceful shutdown", // why
		"delay",             // mode
	).Store(&fd)
	if err != nil {
		return fmt.Errorf("inhibit shutdown: %w", err)
	}
	// Keep the fd alive for the duration of this function so the inhibitor lock
	// remains active. When the function returns (or the conn is closed), the fd
	// is dropped and the lock is released.
	_ = fd

	// Subscribe to PrepareForShutdown signals from systemd-logind
	err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
		dbus.WithMatchMember("PrepareForShutdown"),
	)
	if err != nil {
		return fmt.Errorf("add match signal: %w", err)
	}

	// Create a signal channel
	sig := make(chan *dbus.Signal, 3)
	conn.Signal(sig)

	for msg := range sig {
		if msg.Name == "org.freedesktop.login1.Manager.PrepareForShutdown" {
			// The boolean argument indicates true=preparing for shutdown, false=shutdown canceled
			if len(msg.Body) > 0 {
				if active, ok := msg.Body[0].(bool); ok && active {
					return nil
				}
			}
		}
	}
	return errors.New("D-Bus connection closed")
}
