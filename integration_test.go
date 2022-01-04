package k8sovirtcredentialsmonitor_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	ovirtclient "github.com/ovirt/go-ovirt-client"
	log "github.com/ovirt/go-ovirt-client-log/v2"
	"github.com/ovirt/k8sovirtcredentialsmonitor"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func TestUpdatingSecretShouldTriggerConnectionUpdate(t *testing.T) {
	t.Logf("Attempting to obtain Kubernetes credentials from ~/.kube/config ...")
	kubeConfig, kubernetesClient := setupKubernetesConnection(t)

	serverVersion, err := kubernetesClient.ServerVersion()
	if err != nil {
		t.Skipf("cannot run test, the Kubernetes service from the config file in ~/.kube/config")
	}
	t.Logf("Kubernetes server version is %s.", serverVersion.String())

	ns := "default"

	testSecret := createTestSecret(t, kubernetesClient, ns)
	t.Cleanup(
		func() {
			removeTestSecret(t, kubernetesClient, ns, testSecret)
		},
	)

	url := make(chan string)
	expectedURL := "https://example.com/ovirt-engine/api"
	running := make(chan struct{})
	updateDone := make(chan struct{})
	updateError := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onCredentialsChange := func(connection ovirtclient.ClientWithLegacySupport) {
		t.Logf("Credentials change hook called.")
		url <- connection.GetURL()
	}
	onMonitorRunning := func() {
		t.Logf("Monitor running hook called.")
		running <- struct{}{}
	}
	setupMonitor(t, kubeConfig, testSecret, ns, onCredentialsChange, onMonitorRunning, ctx)

	t.Logf("Waiting for monitor to enter running state...")
	select {
	case <-running:
		t.Logf("Monitor is now running.")
	case <-time.After(time.Minute):
		t.Fatalf("timeout while waiting for running signal")
	}

	go func() {
		t.Logf("Changing test secret to URL %s...", expectedURL)
		secret := testSecret
		secret.Data["ovirt_url"] = []byte(expectedURL)
		_, err := kubernetesClient.CoreV1().Secrets(ns).Update(context.Background(), secret, v1.UpdateOptions{})
		if err != nil {
			updateError <- err
		}
		close(updateError)
		updateDone <- struct{}{}
	}()

	checkUpdateResults(t, url, expectedURL, updateError, updateDone)
}

func checkUpdateResults(
	t *testing.T,
	url chan string,
	expectedURL string,
	updateError chan error,
	updateDone chan struct{},
) {
	t.Logf("Waiting for credentials change...")
	select {
	case foundURL := <-url:
		t.Logf("Received changed URL %s.", foundURL)
		if foundURL != expectedURL {
			t.Fatalf("unexpected oVirt engine URL after update: %s", foundURL)
		}
	case <-time.After(time.Minute):
		t.Fatalf("timeout while waiting for updated signal")
	}

	if err, ok := <-updateError; ok {
		t.Fatalf("Failed to update secret (%v).", err)
	}

	t.Logf("Waiting for update done signal...")
	select {
	case <-updateDone:
		t.Logf("Update is complete.")
	case <-time.After(time.Minute):
		t.Fatalf("Timeout while waiting for update done signal.")
	}
}

func setupMonitor(
	t *testing.T,
	kubeConfig *rest.Config,
	testSecret *corev1.Secret,
	ns string,
	onCredentialsChange func(connection ovirtclient.ClientWithLegacySupport),
	onMonitorRunning func(),
	ctx context.Context,
) {
	t.Logf("Setting up credentials monitor...")
	logger := log.NewTestLogger(t)

	monitor, err := k8sovirtcredentialsmonitor.New(
		k8sovirtcredentialsmonitor.ConnectionConfig{
			Config: kubeConfig,
		},
		k8sovirtcredentialsmonitor.OVirtSecretConfig{
			Name:      testSecret.Name,
			Namespace: ns,
		},
		k8sovirtcredentialsmonitor.Callbacks{
			OnCredentialsChange: onCredentialsChange,
			OnMonitorRunning:    onMonitorRunning,
			OnCredentialsValidate: func(support ovirtclient.ClientWithLegacySupport) error {
				// Don't verify credentials because we don't have a working engine.
				return nil
			},
		},
		logger,
	)
	if err != nil {
		t.Fatalf("failed to instantiate monitor (%v)", err)
	}
	go monitor.Run(ctx)
}

func removeTestSecret(t *testing.T, kubernetesClient *kubernetes.Clientset, ns string, createResponse *corev1.Secret) {
	err := kubernetesClient.CoreV1().Secrets(ns).Delete(
		context.Background(),
		createResponse.Name,
		v1.DeleteOptions{},
	)
	if err != nil {
		t.Fatalf("failed to delete test secret %s after test (%v)", createResponse.Name, err)
	}
}

func createTestSecret(t *testing.T, kubernetesClient *kubernetes.Clientset, ns string) *corev1.Secret {
	url := "https://localhost/ovirt-engine/api"
	t.Logf("Creating test secret with URL %s...", url)
	createResponse, err := kubernetesClient.CoreV1().Secrets(ns).Create(
		context.Background(), &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    ns,
			},

			Data: map[string][]byte{
				"ovirt_url":      []byte(url),
				"ovirt_username": []byte("admin@internal"),
				"ovirt_password": []byte("asdfasdf"),
				"ovirt_insecure": []byte("true"),
			},
			Type: "generic",
		}, v1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("failed to create test secret (%v)", err)
	}
	return createResponse
}

func setupKubernetesConnection(t *testing.T) (*rest.Config, *kubernetes.Clientset) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		t.Fatalf("failed to read Kubeconfig file (%v)", err)
	}
	kubernetesClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		t.Fatalf("failed to create Kubernetes client (%v)", err)
	}
	return kubeConfig, kubernetesClient
}
