package shutdowntask

// Register creates a scheduled task that runs "gws stop --all" on shutdown.
// If dumpXML is true, the rendered XML is printed to stderr before registration.
func Register(dumpXML bool) error {
	return registerShutdownTask(dumpXML)
}

// Unregister removes the scheduled task.
func Unregister() error {
	return unregisterShutdownTask()
}

// Status checks whether the scheduled task is registered.
func Status() error {
	return shutdownTaskStatus()
}

// DumpXML prints the raw task XML template to stdout.
func DumpXML() error {
	return dumpXMLTemplate()
}
