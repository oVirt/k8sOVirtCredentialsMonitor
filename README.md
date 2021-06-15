# K8S oVirt Credential Monitor

This library provides the ability to monitor a secret in Kubernetes and provide an updated oVirt SDK client to a hook function.

It also provides the ability to integrate with the [operator framework](https://sdk.operatorframework.io/).

## Basic usage

In order to use this library you will need to add it as a Go dependency:

```
go get github.com/oVirt/k8sOVirtCredentialsMonitor
```

Then you can set up the monitor:

```go
// This variable will hold the SDK connection
var sdkConnection *ovirtsdk.Connection
// We use the connLock to avoid race conditions
var connLock := &sync.Mutex{}

// This callback will be called when the credentials change.
callback := k8sOVirtCredentialsMonitor.Callbacks{
    OnCredentialChange: func (connection ovirt_credential_monitor.OVirtConnection) {
        connLock.Lock()
        defer connLock.Unlock()

        // Fetch the SDK connection.
        sdkConnection = connection.GetSDK()
    },
    // Optional callbacks without extra parameters
    // - OnMonitorRunning - when the monitor has started
    // - OnMonitorShuttingDown - before the monitor is shutting down
}

// Set up the credential monitor.
monitor, err := k8sOVirtCredentialsMonitor.New(
    k8sOVirtCredentialsMonitor.ConnectionConfig {
        // add Kubernetes connection parameters here
    },
    // Namespace to read the secret from. If empty this will be read 
    secretNamespace string,
    // The secret name must be provided.
    secretName string,
    // callback will be called when the connection updates.
    callbacks,
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
sdkConnection = monitor.GetConnection().GetSDK()
connLock.Unlock()

// Use sdkConnection here.
```

## Configuration format

The oVirt configuration format in secrets is as follows:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ovirt-credentials
type: generic
data:
  ovirt_url: <oVirt engine URL here, base64 encoded>
  ovirt_username: <oVirt username here, base64 encoded>
  ovirt_password: <oVirt password here, base64 encoded>
  ovirt_insecure: <true or false, base64 encoded; optional; not recommended>
  ovirt_ca_bundle: <CA certificate in PEM format, base64-encoded; optional if ovirt_insecure is true>
```
