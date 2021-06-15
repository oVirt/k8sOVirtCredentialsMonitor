package k8sOVirtCredentialsMonitor_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/ovirt/k8sOVirtCredentialsMonitor"
)

func TestUpdatingSecretShouldTriggerConnectionUpdate(t *testing.T) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		t.Fatal(fmt.Errorf("failed to read Kubeconfig file (%w)", err))
	}
	kubernetesClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create Kubernetes client (%w)", err))
	}

	ns := "default"

	createResponse, err := kubernetesClient.CoreV1().Secrets(ns).Create(context.Background(), &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: "test-",
			Namespace: ns,
		},

		Data: map[string][]byte{
			"ovirt_url": []byte("https://localhost/ovirt-engine/api"),
			"ovirt_username": []byte("admin@internal"),
			"ovirt_password": []byte("asdfasdf"),
			"ovirt_insecure": []byte("true"),
		},
		Type: "generic",
	}, v1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create test secret (%w)", err))
	}
	defer func() {
		err := kubernetesClient.CoreV1().Secrets(ns).Delete(
			context.Background(),
			createResponse.Name,
			v1.DeleteOptions{},
		)
		if err != nil {
			t.Fatal(fmt.Errorf("failed to delete test secret %s after test (%w)", createResponse.Name, err))
		}
	}()

	url := ""
	running := make(chan struct{})
	updated := make(chan struct{})
	updateDone := make(chan struct{})
	var updateError error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor, err := k8sOVirtCredentialsMonitor.New(
		k8sOVirtCredentialsMonitor.ConnectionConfig{
			Config: kubeConfig,
		},
		k8sOVirtCredentialsMonitor.OVirtSecretConfig{
			Name: createResponse.Name,
			Namespace: ns,
		},
		k8sOVirtCredentialsMonitor.Callbacks{
			OnCredentialChange: func(connection k8sOVirtCredentialsMonitor.OVirtConnection) {
				url = connection.GetSDK().URL()
				updated <- struct{}{}
			},
			OnMonitorRunning: func() {
				running <- struct{}{}
			},
		},
	)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to instantiate monitor (%w)", err))
	}

	go monitor.Run(ctx)

	select {
	case <-running:
	case <-time.After(time.Minute):
		t.Fatal(fmt.Errorf("timeout while waiting for running signal"))
	}

	go func() {
		secret := createResponse
		secret.Data["ovirt_url"] = []byte("https://example.com/ovirt-engine/api")
		_, updateError = kubernetesClient.CoreV1().Secrets(ns).Update(context.Background(), secret, v1.UpdateOptions{})
		updateDone <- struct{}{}
	}()

	select {
	case <-updated:
	case <-time.After(time.Minute):
		t.Fatal(fmt.Errorf("timeout while waiting for updated signal"))
	}

	if url != "https://example.com/ovirt-engine/api" {
		t.Fatal(fmt.Errorf("unexpected oVirt engine URL after update: %s", url))
	}

	cancel()

	select {
	case <-updateDone:
	case <-time.After(time.Minute):
		t.Fatal(fmt.Errorf("timeout while waiting for updateDone signal"))
	}

	if updateError != nil {
		t.Fatal(fmt.Errorf("failed to update secret (%w)", updateError))
	}
}
