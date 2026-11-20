//go:build windows

package sessionend

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmDestroy         = 0x0002
	wmQueryEndSession = 0x0011
	wmEndSession      = 0x0016
	wmClose           = 0x0010

	shutdownWaitTimeout = 20 * time.Second
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")

	getModuleHandleW          = kernel32.NewProc("GetModuleHandleW")
	registerClassExW          = user32.NewProc("RegisterClassExW")
	createWindowExW           = user32.NewProc("CreateWindowExW")
	getMessageW               = user32.NewProc("GetMessageW")
	translateMessage          = user32.NewProc("TranslateMessage")
	dispatchMessageW          = user32.NewProc("DispatchMessageW")
	defWindowProcW            = user32.NewProc("DefWindowProcW")
	postMessageW              = user32.NewProc("PostMessageW")
	postQuitMessage           = user32.NewProc("PostQuitMessage")
	unregisterClassW          = user32.NewProc("UnregisterClassW")
	shutdownBlockReasonCreate = user32.NewProc("ShutdownBlockReasonCreate")
)

type windowClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   syscall.Handle
	Icon       syscall.Handle
	Cursor     syscall.Handle
	Background syscall.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     syscall.Handle
}

type windowPoint struct {
	X int32
	Y int32
}

type windowMessage struct {
	HWnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      windowPoint
}

// Listen creates a hidden top-level window to receive Windows session-end
// messages. The event is sent after Windows confirms the session is ending;
// release allows shutdown processing to continue or exits after 20 seconds.
func Listen() (<-chan struct{}, func()) {
	className, err := syscall.UTF16PtrFromString("GWSWorkstationSessionEnd")
	if err != nil {
		return nil, func() {}
	}
	title, err := syscall.UTF16PtrFromString("")
	if err != nil {
		return nil, func() {}
	}
	shutdownReason, err := syscall.UTF16PtrFromString("Stopping the workstation before Windows shuts down")
	if err != nil {
		return nil, func() {}
	}

	ended := make(chan struct{}, 1)
	releaseCh := make(chan struct{})
	ready := make(chan uintptr, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		instance, _, _ := getModuleHandleW.Call(0)
		if instance == 0 {
			ready <- 0
			return
		}

		windowProc := syscall.NewCallback(func(hwnd, message, wParam, lParam uintptr) uintptr {
			switch message {
			case wmQueryEndSession:
				_, _, _ = shutdownBlockReasonCreate.Call(hwnd, uintptr(unsafe.Pointer(shutdownReason)))
				return 1
			case wmEndSession:
				if wParam != 0 {
					ended <- struct{}{}
					timer := time.NewTimer(shutdownWaitTimeout)
					defer timer.Stop()
					select {
					case <-releaseCh:
					case <-timer.C:
					}
				}
				return 0
			case wmDestroy:
				_, _, _ = postQuitMessage.Call(0)
				return 0
			default:
				result, _, _ := defWindowProcW.Call(hwnd, message, wParam, lParam)
				return result
			}
		})

		class := windowClassEx{
			Size:      uint32(unsafe.Sizeof(windowClassEx{})),
			WndProc:   windowProc,
			Instance:  syscall.Handle(instance),
			ClassName: className,
		}
		classAtom, _, _ := registerClassExW.Call(uintptr(unsafe.Pointer(&class)))
		if classAtom == 0 {
			ready <- 0
			return
		}

		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(title)),
			0,
			0, 0, 0, 0,
			0, 0,
			instance,
			0,
		)
		if hwnd == 0 {
			_, _, _ = unregisterClassW.Call(uintptr(unsafe.Pointer(className)), instance)
			ready <- 0
			return
		}

		ready <- hwnd
		defer unregisterClassW.Call(uintptr(unsafe.Pointer(className)), instance)

		var message windowMessage
		for {
			result, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
			if int32(result) <= 0 {
				break
			}
			_, _, _ = translateMessage.Call(uintptr(unsafe.Pointer(&message)))
			_, _, _ = dispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
		}
	}()

	hwnd := <-ready
	if hwnd == 0 {
		return nil, func() {}
	}

	return ended, sync.OnceFunc(func() {
		close(releaseCh)
		_, _, _ = postMessageW.Call(hwnd, wmClose, 0, 0)
	})
}
