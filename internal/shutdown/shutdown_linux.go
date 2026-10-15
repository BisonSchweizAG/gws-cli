package shutdown

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

// WaitForShutdown blocks until the system is shutting down (power off, not reboot).
// It listens for the PrepareForShutdown signal from systemd-logind via D-Bus.
func WaitForShutdown() error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("connect to system bus: %w", err)
	}
	defer conn.Close()

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
