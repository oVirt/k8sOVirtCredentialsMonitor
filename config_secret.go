package k8sOVirtCredentialsMonitor

import (
	"fmt"
)

type OVirtSecretConfig struct {
	// Name is the name of the secret to be read.
	Name string
	// Namespace is the optional namespace specification.
	Namespace string
}

func (o OVirtSecretConfig) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("the Name field for the oVirt secret config is required")
	}
	if o.Namespace == "" {
		return fmt.Errorf("the Namespace field for the oVirt secret config is required")
	}
	return nil
}