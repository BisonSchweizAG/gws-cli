//go:build windows

package shutdown

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmQueryEndSession = 0x0011
	wmEndSession      = 0x0016

	endSessionCloseApp = 0x00000001
	endSessionLogoff   = 0x80000000

	// ewxReboot is the EWX_REBOOT flag carried in wParam of WM_QUERYENDSESSION
	// on modern Windows, indicating a restart rather than a power-off shutdown.
	ewxReboot = 0x00000002

	errorClassAlreadyExists = 1410

	windowClassName = "GWSShutdownWatcher"
	blockReason     = "Stopping GWS workstations"

	// releaseTimeout limits how long the session end is blocked if the caller never releases.
	releaseTimeout = 30 * time.Second
)

var (
	user32                         = windows.NewLazySystemDLL("user32.dll")
	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procShutdownBlockReasonCreate  = user32.NewProc("ShutdownBlockReasonCreate")
	procShutdownBlockReasonDestroy = user32.NewProc("ShutdownBlockReasonDestroy")
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type point struct {
	X, Y int32
}

type winMsg struct {
	Hwnd     windows.HWND
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

// WaitForShutdown blocks until the Windows session ends (shutdown or logoff).
// It ignores restart (reboot) events.
//
// Console control events (CTRL_SHUTDOWN_EVENT / CTRL_LOGOFF_EVENT) are not delivered to
// processes that load user32.dll (which gws does, e.g. for the clipboard), and the process
// is terminated as soon as such a handler returns. Therefore, a hidden top-level window is
// used to receive WM_QUERYENDSESSION / WM_ENDSESSION. The session end is blocked until the
// returned ReleaseFunc is called (or releaseTimeout elapsed).
func WaitForShutdown() (ReleaseFunc, error) {
	logf("WaitForShutdown: starting Windows shutdown watcher")
	detected := make(chan struct{})
	released := make(chan struct{})
	errCh := make(chan error, 1)

	go messageLoop(detected, released, errCh)

	select {
	case <-detected:
		logf("WaitForShutdown: shutdown detected, returning release function")
		return sync.OnceFunc(func() {
			logf("WaitForShutdown: release function called")
			close(released)
		}), nil
	case err := <-errCh:
		logf("WaitForShutdown: error from message loop: %v", err)
		return nil, err
	}
}

// isSessionEnd returns false for Restart Manager requests (e.g. installers) that only
// want the application to close while the session continues.
func isSessionEnd(lParam uintptr) bool {
	return lParam&endSessionCloseApp == 0 || lParam&endSessionLogoff != 0
}

func messageLoop(detected chan<- struct{}, released <-chan struct{}, errCh chan<- error) {
	// The window and its message loop must stay on the same OS thread.
	// The thread is never unlocked; it is terminated together with the goroutine.
	runtime.LockOSThread()

	reason, err := windows.UTF16PtrFromString(blockReason)
	if err != nil {
		errCh <- err
		return
	}
	className, err := windows.UTF16PtrFromString(windowClassName)
	if err != nil {
		errCh <- err
		return
	}

	var notify sync.Once
	wndProc := func(hwnd, msg, wParam, lParam uintptr) uintptr {
		switch msg {
		case wmQueryEndSession:
			isReboot := wParam&ewxReboot != 0
			sessionEnd := isSessionEnd(lParam)
			logf("WM_QUERYENDSESSION received: wParam=0x%08x, lParam=0x%08x, ewxReboot=%v, isSessionEnd=%v",
				wParam, lParam, isReboot, sessionEnd)
			// Skip reboot: wParam carries the EWX_REBOOT flag on modern Windows.
			if wParam&ewxReboot == 0 && isSessionEnd(lParam) {
				logf("WM_QUERYENDSESSION: blocking shutdown (not reboot, is session end)")
				// Shows the reason to the user and gives us time to stop the workstations.
				_, _, _ = procShutdownBlockReasonCreate.Call(hwnd, uintptr(unsafe.Pointer(reason)))
			} else {
				if isReboot {
					logf("WM_QUERYENDSESSION: skipping — reboot detected via EWX_REBOOT flag")
				} else if !sessionEnd {
					logf("WM_QUERYENDSESSION: skipping — not a session end (close-app only)")
				}
			}
			return 1 // allow the session to end
		case wmEndSession:
			isReboot := wParam&ewxReboot != 0
			sessionEnd := isSessionEnd(lParam)
			logf("WM_ENDSESSION received: wParam=0x%08x, lParam=0x%08x, ewxReboot=%v, isSessionEnd=%v, wParamNonZero=%v",
				wParam, lParam, isReboot, sessionEnd, wParam != 0)
			// Skip reboot: wParam carries the EWX_REBOOT flag on modern Windows.
			if wParam&ewxReboot != 0 {
				logf("WM_ENDSESSION: ignoring — reboot detected via EWX_REBOOT flag")
				return 0
			}
			if !isSessionEnd(lParam) {
				logf("WM_ENDSESSION: ignoring — not a session end (close-app only)")
				return 0
			}
			if wParam != 0 { // session is really ending
				logf("WM_ENDSESSION: triggering shutdown hook — session is ending")
				notify.Do(func() { close(detected) })
				// The process may be terminated as soon as we return: wait for the caller.
				select {
				case <-released:
					logf("WM_ENDSESSION: released by caller")
				case <-time.After(releaseTimeout):
					logf("WM_ENDSESSION: release timeout elapsed (%v)", releaseTimeout)
				}
			}
			_, _, _ = procShutdownBlockReasonDestroy.Call(hwnd)
			return 0
		}
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return r
	}

	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		errCh <- fmt.Errorf("get module handle: %w", err)
		return
	}

	wc := wndClassEx{
		WndProc:   windows.NewCallback(wndProc),
		Instance:  instance,
		ClassName: className,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		if !errors.Is(err, windows.Errno(errorClassAlreadyExists)) {
			errCh <- fmt.Errorf("register window class: %w", err)
			return
		}
	}

	// A hidden top-level window (not a message-only window, those don't receive WM_QUERYENDSESSION).
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(className)),
		0,          // style: WS_OVERLAPPED, not visible
		0, 0, 0, 0, // x, y, width, height
		0, 0, // parent, menu
		uintptr(instance),
		0,
	)
	if hwnd == 0 {
		errCh <- fmt.Errorf("create window: %w", err)
		return
	}

	var m winMsg
	for {
		r, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		switch int32(r) {
		case -1:
			errCh <- fmt.Errorf("get message: %w", err)
			return
		case 0: // WM_QUIT
			errCh <- errors.New("message loop ended")
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
