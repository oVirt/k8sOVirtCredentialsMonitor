package k8sOVirtCredentialsMonitor

// Callbacks is a configuration structure that holds the possible callbacks for the monitor.
// The OnCredentialChange callback is required.
type Callbacks struct {
	// OnMonitorRunning is a callback that is called after the watch is set up for the secret.
	OnMonitorRunning func()
	// OnMonitorShuttingDown is a callback that is called before the watch is shut down.
	OnMonitorShuttingDown func()
	// OnCredentialChange is called when the oVirt credentials change.
	OnCredentialChange func(OVirtConnection)
}
