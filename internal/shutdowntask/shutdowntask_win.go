//go:build windows

package shutdowntask

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed task.xml
var taskXMLFS embed.FS

const shutdownTaskName = `\gws\GWS-StopOnShutdown`

type taskXMLData struct {
	Command   string
	Arguments string
}

// registerShutdownTask creates a Windows scheduled task that runs "gws stop --all --now" on shutdown.
func registerShutdownTask(dumpXML bool) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve gws executable path: %w", err)
	}

	// Check if the task already exists.
	cmd := exec.Command("schtasks", "/query", "/tn", shutdownTaskName)
	if err := cmd.Run(); err == nil {
		fmt.Printf("Shutdown task %s is already registered.\n", shutdownTaskName)
		return nil
	}

	// Read the embedded task XML template.
	taskXML, err := taskXMLFS.ReadFile("task.xml")
	if err != nil {
		return fmt.Errorf("read task XML template: %w", err)
	}

	// Write the task XML to a temporary file.
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "GWS-StopOnShutdown.xml")
	tmpl, err := template.New("task").Parse(string(taskXML))
	if err != nil {
		return fmt.Errorf("parse task template: %w", err)
	}
	data := taskXMLData{
		Command:   execPath,
		Arguments: "stop --all --now",
	}
	renderedXML := mustRender(tmpl, data)
	if dumpXML {
		fmt.Fprintln(os.Stderr, string(renderedXML))
		return nil
	}
	if err := os.WriteFile(tmpFile, renderedXML, 0644); err != nil {
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
	fmt.Printf("Shutdown task %s registered successfully.\n", shutdownTaskName)
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
		fmt.Printf("Shutdown task %s is not registered.\n", shutdownTaskName)
		return nil
	}

	// Task exists, delete it.
	deleteCmd := exec.Command("schtasks", "/delete", "/tn", shutdownTaskName, "/f")
	output, err := deleteCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete shutdown task: %s: %w", strings.TrimSpace(string(output)), err)
	}
	fmt.Printf("Shutdown task %s unregistered successfully.\n", shutdownTaskName)
	return nil
}

// shutdownTaskStatus checks whether the Windows scheduled task is registered.
func shutdownTaskStatus() error {
	cmd := exec.Command("schtasks", "/query", "/tn", shutdownTaskName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Shutdown task %s is not registered.\n", shutdownTaskName)
		return nil
	}
	fmt.Printf("Shutdown task %s is registered.\n", shutdownTaskName)
	return nil
}

// dumpXMLTemplate prints the raw embedded task XML template to stdout.
func dumpXMLTemplate() error {
	taskXML, err := taskXMLFS.ReadFile("task.xml")
	if err != nil {
		return fmt.Errorf("read task XML template: %w", err)
	}
	fmt.Println(string(taskXML))
	return nil
}
