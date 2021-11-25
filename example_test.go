package k8sovirtcredentialsmonitor_test

import (
	"context"
	goLog "log"
	"os"
	"sync"

	ovirtclient "github.com/ovirt/go-ovirt-client"
	log "github.com/ovirt/go-ovirt-client-log/v2"
	"github.com/ovirt/k8sovirtcredentialsmonitor"
)

// Example showcases how to use the k8s credentials monitor. We are disabling the linter here so the example can
// be fit in a single function.
func Example() { //nolint:funlen
	// Set the namespace to monitor.
	var secretNamespace = "default"
	// Set the secret to monitor. We are disabling the linter here because the hard-coded secret name is just for show.
	var secretName = "ovirt-credentials" //nolint:gosec
	// This variable will hold connection.
	var connection ovirtclient.ClientWithLegacySupport
	// Set up logging. See go-ovirt-client-log for details.
	var logger = log.NewGoLogger(goLog.New(os.Stdout, "", 0))

	// We use the connLock to avoid race conditions.
	connLock := &sync.Mutex{}

	// This callback will be called when the credentials change.
	callbacks := k8sovirtcredentialsmonitor.Callbacks{
		OnCredentialsChange: func(
			conn ovirtclient.ClientWithLegacySupport,
		) {
			connLock.Lock()
			defer connLock.Unlock()

			// Save the connection.
			connection = conn
		},
		// Optional callbacks without extra parameters
		// - OnMonitorRunning - when the monitor has started
		// - OnMonitorShuttingDown - before the monitor is shutting down
		// - OnCredentialsValidate - allows you to validate credentials before use.
		//                           If no function is passed
		//                           k8sOVirtCredentialsMonitor.ValidateCredentials
		//                           will be used.
	}

	// Set up the credential monitor.
	monitor, err := k8sovirtcredentialsmonitor.New(
		k8sovirtcredentialsmonitor.ConnectionConfig{
			// add Kubernetes connection parameters here
		},
		k8sovirtcredentialsmonitor.OVirtSecretConfig{
			Name:      secretName,
			Namespace: secretNamespace,
		},
		callbacks,
		logger,
	)
	if err != nil {
		// Handle error
		panic(err)
	}

	// Set up the context for stopping the monitor.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the monitor in the background. This goroutine runs until the context
	// is canceled.
	go monitor.Run(ctx)

	// Set up initial connection. We do this in a lock to avoid race conditions.
	connLock.Lock()
	connection, err = monitor.GetConnection()
	connLock.Unlock()
	if err != nil {
		// The connection has failed. Handle error here.
		goLog.Fatalf("oVirt connection failed (%v)", err)
	}

	// Use sdkConnection here. The connection variable will be updated in the hooks above when the credentials change.
	print(connection.GetURL())
}
