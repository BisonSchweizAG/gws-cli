// Package shutdown detects a system power-off so that running workstations can be
// stopped before the machine goes down.
//
// Platform support:
//   - linux: systemd-logind PrepareForShutdown signal (D-Bus) with a delay inhibitor lock.
//     Reboots are ignored on systemd >= 255; older versions can not distinguish a reboot.
//   - darwin: loginwindow "shutdownInitiated" notification via /usr/bin/notifyutil (no cgo needed).
//     Restarts and logouts are ignored.
//   - windows: WM_QUERYENDSESSION / WM_ENDSESSION on a hidden window. The EWX_REBOOT flag
//     in wParam is used to distinguish reboot from shutdown on modern Windows.
//   - other: not supported.
//
// Logging:
//
//	All shutdown events are logged to a `shutdown.log` file in the GWS config directory
//	(~/.config/gws/shutdown.log on Linux/macOS, %USERPROFILE%\.config\gws\shutdown.log on Windows).
//	The log includes timestamps, event types, and decision details (e.g. reboot vs shutdown),
//	which helps diagnose issues with the shutdown detection.
package shutdown

// ReleaseFunc must be called once the caller has finished its shutdown work.
// It releases the resources (e.g. the inhibitor lock) that delay the system shutdown.
// It is safe to call it multiple times.
type ReleaseFunc func()
