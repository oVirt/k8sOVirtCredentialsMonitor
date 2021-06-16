package k8sOVirtCredentialsMonitor_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/oVirt/k8sOVirtCredentialsMonitor"
)

func TestUpdatingSecretShouldTriggerConnectionUpdate(t *testing.T) {
	kubeConfig, kubernetesClient := setupKubernetesConnection(t)

	ns := "default"

	testSecret := createTestSecret(t, kubernetesClient, ns)
	defer removeTestSecret(t, kubernetesClient, ns, testSecret)

	url := make(chan string)
	expectedURL := "https://example.com/ovirt-engine/api"
	running := make(chan struct{})
	updateDone := make(chan struct{})
	updateError := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onCredentialsChange := func(connection k8sOVirtCredentialsMonitor.OVirtConnection) {
		url <- connection.GetSDK().URL()
	}
	onMonitorRunning := func() {
		running <- struct{}{}
	}
	setupMonitor(t, kubeConfig, testSecret, ns, onCredentialsChange, onMonitorRunning, ctx)

	select {
	case <-running:
	case <-time.After(time.Minute):
		t.Fatalf("timeout while waiting for running signal")
	}

	go func() {
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
	select {
	case foundURL := <-url:
		if foundURL != expectedURL {
			t.Fatalf("unexpected oVirt engine URL after update: %s", foundURL)
		}
	case <-time.After(time.Minute):
		t.Fatalf("timeout while waiting for updated signal")
	}

	if err, ok := <-updateError; ok {
		t.Fatalf("failed to update secret (%v)", err)
	}

	select {
	case <-updateDone:
	case <-time.After(time.Minute):
		t.Fatalf("timeout while waiting for updateDone signal")
	}
}

func setupMonitor(
	t *testing.T,
	kubeConfig *rest.Config,
	testSecret *corev1.Secret,
	ns string,
	onCredentialsChange func(connection k8sOVirtCredentialsMonitor.OVirtConnection),
	onMonitorRunning func(),
	ctx context.Context,
) {
	logger := k8sOVirtCredentialsMonitor.NewTestLogger(t)

	monitor, err := k8sOVirtCredentialsMonitor.New(
		k8sOVirtCredentialsMonitor.ConnectionConfig{
			Config: kubeConfig,
		},
		k8sOVirtCredentialsMonitor.OVirtSecretConfig{
			Name:      testSecret.Name,
			Namespace: ns,
		},
		k8sOVirtCredentialsMonitor.Callbacks{
			OnCredentialChange: onCredentialsChange,
			OnMonitorRunning:   onMonitorRunning,
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
	createResponse, err := kubernetesClient.CoreV1().Secrets(ns).Create(
		context.Background(), &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    ns,
			},

			Data: map[string][]byte{
				"ovirt_url":      []byte("https://localhost/ovirt-engine/api"),
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
