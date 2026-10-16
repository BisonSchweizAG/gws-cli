//go:build linux

package shutdown

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	logindDest  = "org.freedesktop.login1"
	logindPath  = "/org/freedesktop/login1"
	logindIface = "org.freedesktop.login1.Manager"
)

// WaitForShutdown blocks until the system is powering off (not rebooting).
// It holds a delay inhibitor lock with systemd-logind, so the shutdown is delayed
// (up to logind's InhibitDelayMaxSec) until the returned ReleaseFunc is called.
func WaitForShutdown() (ReleaseFunc, error) {
	log("WaitForShutdown: starting Linux shutdown watcher (systemd-logind)")
	// Use a private connection, closing the shared dbus.SystemBus() would break other users.
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log("WaitForShutdown: failed to connect to system bus: %v", err)
		return nil, fmt.Errorf("connect to system bus: %w", err)
	}

	release, err := waitForPowerOff(conn)
	if err != nil {
		_ = conn.Close()
		log("WaitForShutdown: waitForPowerOff error: %v", err)
		return nil, err
	}
	return sync.OnceFunc(func() {
		log("WaitForShutdown: release function called")
		release()
		_ = conn.Close()
	}), nil
}

func waitForPowerOff(conn *dbus.Conn) (func(), error) {
	logind := conn.Object(logindDest, logindPath)

	err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath(logindPath),
		dbus.WithMatchInterface(logindIface),
		dbus.WithMatchMember("PrepareForShutdown"),
	)
	if err != nil {
		return nil, fmt.Errorf("add match signal: %w", err)
	}

	sig := make(chan *dbus.Signal, 10)
	conn.Signal(sig)

	lock, err := inhibit(logind)
	if err != nil {
		return nil, err
	}

	for msg := range sig {
		if msg.Name != logindIface+".PrepareForShutdown" || len(msg.Body) == 0 {
			continue
		}
		active, ok := msg.Body[0].(bool)
		if !ok {
			continue
		}

		if !active {
			log("PrepareForShutdown: signal received, active=false (shutdown/reboot canceled)")
			// Shutdown/reboot was canceled: make sure we hold a lock for the next one.
			if lock == nil {
				if lock, err = inhibit(logind); err != nil {
					return nil, err
				}
			}
			continue
		}

		st := shutdownType(logind)
		log("PrepareForShutdown: signal received, active=true, shutdownType=%q, isPowerOff=%v", st, isPowerOff(st))
		if isPowerOff(st) {
			log("PrepareForShutdown: triggering shutdown hook — power off detected")
			return func() { _ = lock.Close() }, nil
		}

		// Reboot (or similar): do not delay it.
		log("PrepareForShutdown: ignoring — reboot detected (shutdownType=%q)", st)
		if lock != nil {
			_ = lock.Close()
			lock = nil
		}
	}

	if lock != nil {
		_ = lock.Close()
	}
	return nil, errors.New("D-Bus connection closed")
}

// inhibit takes a delay inhibitor lock. The lock is held as long as the returned file is open.
func inhibit(logind dbus.BusObject) (*os.File, error) {
	var fd dbus.UnixFD
	err := logind.Call(
		logindIface+".Inhibit", 0,
		"shutdown",                  // what
		"GWS CLI",                   // who
		"Stopping GWS workstations", // why
		"delay",                     // mode
	).Store(&fd)
	if err != nil {
		return nil, fmt.Errorf("inhibit shutdown: %w", err)
	}
	return os.NewFile(uintptr(fd), "logind-inhibitor"), nil
}

// shutdownType returns the type of the shutdown in progress ("power-off", "reboot", ...)
// or an empty string if logind does not provide it (systemd < 255).
func shutdownType(logind dbus.BusObject) string {
	v, err := logind.GetProperty(logindIface + ".PreparingForShutdownWithMetadata")
	if err != nil {
		return ""
	}
	return shutdownTypeFromMetadata(v)
}

func shutdownTypeFromMetadata(v dbus.Variant) string {
	md, ok := v.Value().(map[string]dbus.Variant)
	if !ok {
		return ""
	}
	t, ok := md["type"]
	if !ok {
		return ""
	}
	if s, ok := t.Value().(string); ok {
		return s
	}
	return ""
}

// isPowerOff reports whether the shutdown type means the machine goes down.
// An unknown type (older systemd) is treated as power-off to not leave workstations running.
func isPowerOff(shutdownType string) bool {
	switch shutdownType {
	case "", "power-off", "halt":
		return true
	default: // reboot, soft-reboot, kexec
		return false
	}
}
