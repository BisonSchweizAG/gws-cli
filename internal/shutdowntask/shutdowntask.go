package shutdowntask

// Register creates a scheduled task that runs "gws stop --all" on shutdown.
func Register() error {
	return registerShutdownTask()
}

// Unregister removes the scheduled task.
func Unregister() error {
	return unregisterShutdownTask()
}

// Status checks whether the scheduled task is registered.
func Status() error {
	return shutdownTaskStatus()
}
