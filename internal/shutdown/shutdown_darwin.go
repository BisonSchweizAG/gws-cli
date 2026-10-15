//go:build darwin

package shutdown

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <notify.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

// Shared state for notification handling
static int g_shutdownReceived = 0;
static int g_restartReceived = 0;

// Mach port callback for Darwin notifications
void notifyCallback(CFMachPortRef port, void *msg, CFIndex size, void *info) {
    uint32_t *token = (uint32_t *)msg;
    if (token == NULL) return;

    // Check the notification state to see if it fired
    int state = notify_check(*token, NULL);
    if (state == NOTIFY_STATUS_INSTALLED) {
        // Notification has fired
        char *portName = (char *)info;
        if (portName != NULL) {
            if (strcmp(portName, "com.apple.system.loginwindow.shutdownInitiated") == 0) {
                g_shutdownReceived = 1;
            } else if (strcmp(portName, "com.apple.system.loginwindow.restartinitiated") == 0) {
                g_restartReceived = 1;
            }
        }
    }
}

// Run loop context structure
typedef struct {
    uint32_t shutdownToken;
    uint32_t restartToken;
    CFMachPortRef shutdownPort;
    CFMachPortRef restartPort;
    CFRunLoopSourceRef shutdownSource;
    CFRunLoopSourceRef restartSource;
} NotifyContext;

// Stop run loop when shutdown is detected
void stopRunLoop(void *info) {
    CFRunLoopStop(CFRunLoopGetCurrent());
}

int waitForShutdownDarwin() {
    NotifyContext ctx;
    ctx.shutdownToken = 0;
    ctx.restartToken = 0;

    // Register for shutdown notification
    int status = notify_register_mach_port(
        "com.apple.system.loginwindow.shutdownInitiated",
        &ctx.shutdownToken,
        MACH_PORT_NULL,
        NOTIFY_REUSE);
    if (status != NOTIFY_STATUS_OK) {
        return -1;
    }

    // Register for restart notification
    status = notify_register_mach_port(
        "com.apple.system.loginwindow.restartinitiated",
        &ctx.restartToken,
        MACH_PORT_NULL,
        NOTIFY_REUSE);
    if (status != NOTIFY_STATUS_OK) {
        notify_cancel(ctx.shutdownToken);
        return -1;
    }

    // Create mach ports for notifications
    ctx.shutdownPort = CFMachPortCreateWithPort(
        kCFAllocatorDefault,
        ctx.shutdownToken,
        notifyCallback,
        CFAllocatorDefault,
        NULL);

    ctx.restartPort = CFMachPortCreateWithPort(
        kCFAllocatorDefault,
        ctx.restartToken,
        notifyCallback,
        CFAllocatorDefault,
        NULL);

    if (!ctx.shutdownPort || !ctx.restartPort) {
        if (ctx.shutdownPort) CFRelease(ctx.shutdownPort);
        if (ctx.restartPort) CFRelease(ctx.restartPort);
        notify_cancel(ctx.shutdownToken);
        notify_cancel(ctx.restartToken);
        return -1;
    }

    // Create run loop sources
    ctx.shutdownSource = CFMachPortCreateRunLoopSource(
        kCFAllocatorDefault, ctx.shutdownPort, 0);
    ctx.restartSource = CFMachPortCreateRunLoopSource(
        kCFAllocatorDefault, ctx.restartPort, 0);

    if (!ctx.shutdownSource || !ctx.restartSource) {
        CFRelease(ctx.shutdownPort);
        CFRelease(ctx.restartPort);
        if (ctx.shutdownSource) CFRelease(ctx.shutdownSource);
        if (ctx.restartSource) CFRelease(ctx.restartSource);
        notify_cancel(ctx.shutdownToken);
        notify_cancel(ctx.restartToken);
        return -1;
    }

    // Add sources to current run loop
    CFRunLoopAddSource(CFRunLoopGetCurrent(), ctx.shutdownSource, kCFRunLoopDefaultMode);
    CFRunLoopAddSource(CFRunLoopGetCurrent(), ctx.restartSource, kCFRunLoopDefaultMode);

    // Add observer to stop run loop when shutdown is detected
    CFRunLoopObserverRef observer = CFRunLoopObserverCreate(
        kCFAllocatorDefault,
        kCFRunLoopAllActivities,
        true,
        0,
        stopRunLoop,
        NULL);
    if (observer) {
        CFRunLoopAddObserver(CFRunLoopGetCurrent(), observer, kCFRunLoopDefaultMode);
        CFRelease(observer);
    }

    // Run the run loop - this will block until shutdown is detected
    CFRunLoopRun();

    // Cleanup
    CFRunLoopRemoveSource(CFRunLoopGetCurrent(), ctx.shutdownSource, kCFRunLoopDefaultMode);
    CFRunLoopRemoveSource(CFRunLoopGetCurrent(), ctx.restartSource, kCFRunLoopDefaultMode);
    CFRelease(ctx.shutdownSource);
    CFRelease(ctx.restartSource);
    CFRelease(ctx.shutdownPort);
    CFRelease(ctx.restartPort);
    notify_cancel(ctx.shutdownToken);
    notify_cancel(ctx.restartToken);

    // Return 0 for shutdown, 1 for restart
    if (g_restartReceived) {
        return 1;
    }
    return 0;
}
*/
import "C"

import "fmt"

// WaitForShutdown blocks until the system is shutting down (not rebooting).
// It uses Darwin notifications to listen for loginwindow shutdown/restart events.
// Returns nil for shutdown, error for restart.
func WaitForShutdown() error {
	result := C.waitForShutdownDarwin()
	if result == -1 {
		return fmt.Errorf("failed to register for shutdown notifications")
	}
	// result == 0 means shutdown, result == 1 means restart
	// We only want to stop workstations on shutdown, not restart
	if result == 1 {
		return fmt.Errorf("system is restarting, not shutting down")
	}
	return nil
}
