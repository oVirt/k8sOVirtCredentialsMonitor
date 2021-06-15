package k8sOVirtCredentialsMonitor

import (
	"context"
	"fmt"
	"sync"

	ovirtsdk "github.com/ovirt/go-ovirt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// OVirtCredentialMonitor is a utility to monitor a Kubernetes secret and call a callback function with a new
// connection whenever the credentials change.
type OVirtCredentialMonitor interface {
	// GetConnection returns the client from the latest credential update. It is recommended that this function should
	// only be called under a lock shared with the callback to avoid race conditions.
	GetConnection() OVirtConnection
	// Run runs in foreground until the context expires.
	Run(ctx context.Context)
}

type oVirtCredentialMonitor struct {
	cli          *kubernetes.Clientset
	secretConfig OVirtSecretConfig
	callbacks    Callbacks
	secret       *corev1.Secret
	lock         *sync.Mutex
	connection   OVirtConnection
}

func (o *oVirtCredentialMonitor) GetConnection() OVirtConnection {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.connection
}

func (o *oVirtCredentialMonitor) Run(ctx context.Context) {
	w, err := o.cli.CoreV1().Secrets(o.secretConfig.Namespace).Watch(
		ctx, v1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", o.secretConfig.Name),
			Watch:         true,
		},
	)
	if err != nil {
		panic(err)
	}
	defer func() {
		if o.callbacks.OnMonitorShuttingDown != nil {
			o.callbacks.OnMonitorShuttingDown()
		}
		w.Stop()
	}()
	if o.callbacks.OnMonitorRunning != nil {
		o.callbacks.OnMonitorRunning()
	}
	for {
		select {
		case result := <-w.ResultChan():
			if result.Type == watch.Modified {
				if secret, ok := result.Object.(*corev1.Secret); ok {
					o.sendCallback(secret)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (o *oVirtCredentialMonitor) sendCallback(secret *corev1.Secret) {
	o.lock.Lock()
	defer o.lock.Unlock()
	conn, err := buildConnection(secret)
	if err != nil {
		//TODO handle error properly
		panic(err)
	}
	o.connection = conn
	o.callbacks.OnCredentialChange(
		conn,
	)
}

func buildConnection(secret *corev1.Secret) (OVirtConnection, error) {
	data := secret.Data
	connectionBuilder := ovirtsdk.NewConnectionBuilder()
	if url, ok := data["ovirt_url"]; ok {
		connectionBuilder = connectionBuilder.URL(string(url))
	}
	if username, ok := data["ovirt_username"]; ok {
		connectionBuilder = connectionBuilder.Username(string(username))
	}
	if password, ok := data["ovirt_password"]; ok {
		connectionBuilder = connectionBuilder.Password(string(password))
	}
	if insecure, ok := data["ovirt_insecure"]; ok {
		if string(insecure) == "true" {
			connectionBuilder = connectionBuilder.Insecure(true)
		} else {
			connectionBuilder = connectionBuilder.Insecure(false)
		}
	}
	if bundle, ok := data["ovirt_ca_bundle"]; ok {
		connectionBuilder = connectionBuilder.CACert(bundle)
	}
	conn, err := connectionBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build oVirt connection object (%w)", err)
	}
	return &oVirtConnection{
		conn: conn,
	}, nil
}
