//go:build windows

package shutdowntask

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const shutdownTaskName = `\gws\GWS-StopOnShutdown`

// taskXMLTemplate is the Windows Task Scheduler XML template.
// It triggers on System event 1074 (User32) only for shutdown/power off, not restart.
const taskXMLTemplate = `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Stops all GWS workstations when Windows shuts down</Description>
  </RegistrationInfo>
  <Triggers>
    <EventTrigger>
      <Enabled>true</Enabled>
      <Subscription>&lt;QueryList&gt;&lt;Query Id="0" Path="System"&gt;&lt;Select Path="System"&gt;*[System[Provider[@Name='User32'] and EventID=1074]] and *[EventData[Data[@Name='param5']='power off' or Data[@Name='param5']='shutdown']]&lt;/Select&gt;&lt;/Query&gt;&lt;/QueryList&gt;</Subscription>
    </EventTrigger>
  </Triggers>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>false</StartWhenAvailable>
    <Enabled>true</Enabled>
    <ExecutionTimeLimit>PT1M</ExecutionTimeLimit>
  </Settings>
  <Actions>
    <Exec>
      <Command>{{.Command}}</Command>
      <Arguments>{{.Arguments}}</Arguments>
    </Exec>
  </Actions>
</Task>`

type taskXMLData struct {
	Command   string
	Arguments string
}

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

	// Write the task XML to a temporary file.
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, shutdownTaskName+".xml")
	tmpl, err := template.New("task").Parse(taskXMLTemplate)
	if err != nil {
		return fmt.Errorf("parse task template: %w", err)
	}
	data := taskXMLData{
		Command:   execPath,
		Arguments: "stop --all",
	}
	if err := os.WriteFile(tmpFile, mustRender(tmpl, data), 0644); err != nil {
		return fmt.Errorf("write task XML: %w", err)
	}
	defer os.Remove(tmpFile)

	// Create the task from the XML file.
	createCmd := exec.Command("schtasks",
		"/create",
		"/tn", shutdownTaskName,
		"/xml", tmpFile,
	)
	output, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create shutdown task: %s: %w", strings.TrimSpace(string(output)), err)
	}
	fmt.Printf("Shutdown task %q registered successfully.\n", shutdownTaskName)
	return nil
}

// mustRender renders the template and panics on error (should never happen with static templates).
func mustRender(tmpl *template.Template, data any) []byte {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("render task template: %w", err))
	}
	return []byte(buf.String())
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
