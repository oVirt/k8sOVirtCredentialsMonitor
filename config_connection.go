package k8sOVirtCredentialsMonitor

import (
	"k8s.io/client-go/rest"
)

// ConnectionConfig contains the configuration for the Kubernetes API.
type ConnectionConfig struct {
	*rest.Config
}
