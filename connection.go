package k8sOVirtCredentialsMonitor

import (
	ovirtsdk "github.com/ovirt/go-ovirt"
)

// OVirtConnection holds the created connection(s). This interface is added for extensibility.
type OVirtConnection interface {
	// GetSDK returns a direct connection from the oVirt Go SDK.
	GetSDK() *ovirtsdk.Connection
}

type oVirtConnection struct {
	conn *ovirtsdk.Connection
}

func (o *oVirtConnection) GetSDK() *ovirtsdk.Connection {
	return o.conn
}