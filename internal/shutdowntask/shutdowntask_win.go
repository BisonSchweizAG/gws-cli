//go:build windows

package shutdowntask

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const shutdownTaskName = "GWS-StopOnShutdown"

// registerShutdownTask creates a Windows scheduled task that runs "gws stop --all" on shutdown.
func registerShutdownTask() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve gws executable path: %w", err)
	}

	// Check if the task already exists.
	cmd := exec.Command("schtasks", "/query", "/tn", shutdownTaskName)
	if err := cmd.Run(); err == nil {
		fmt.Printf("Shutdown task %q is already registered.\n", shutdownTaskName)
		return nil
	}

	// Task doesn't exist, create it.
	// Use ONEVENT trigger with XPath query for System log, event ID 1074 (shutdown/restart).
	createCmd := exec.Command("schtasks",
		"/create",
		"/tn", shutdownTaskName,
		"/tr", fmt.Sprintf(`"%s" stop --all`, execPath),
		"/sc", "ONEVENT",
		"/ec", "System",
		"/mo", `[System/EventID=1074]`,
	)
	output, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create shutdown task: %s: %w", strings.TrimSpace(string(output)), err)
	}
	fmt.Printf("Shutdown task %q registered successfully.\n", shutdownTaskName)
	return nil
}

// unregisterShutdownTask removes the Windows scheduled task.
func unregisterShutdownTask() error {
	// Check if the task exists.
	cmd := exec.Command("schtasks", "/query", "/tn", shutdownTaskName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Shutdown task %q is not registered.\n", shutdownTaskName)
		return nil
	}

	// Task exists, delete it.
	deleteCmd := exec.Command("schtasks", "/delete", "/tn", shutdownTaskName, "/f")
	output, err := deleteCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete shutdown task: %s: %w", strings.TrimSpace(string(output)), err)
	}
	fmt.Printf("Shutdown task %q unregistered successfully.\n", shutdownTaskName)
	return nil
}

// shutdownTaskStatus checks whether the Windows scheduled task is registered.
func shutdownTaskStatus() error {
	cmd := exec.Command("schtasks", "/query", "/tn", shutdownTaskName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Shutdown task %q is not registered.\n", shutdownTaskName)
		return nil
	}
	fmt.Printf("Shutdown task %q is registered.\n", shutdownTaskName)
	return nil
}
