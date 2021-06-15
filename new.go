package k8sOVirtCredentialsMonitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func New(
	connectionConfig ConnectionConfig,
	secretConfig OVirtSecretConfig,
	callbacks Callbacks,
	logger Logger,
) (OVirtCredentialMonitor, error) {
	if err := secretConfig.Validate(); err != nil {
		return nil, fmt.Errorf("secret configuration validation failed (%w)", err)
	}

	if callbacks.OnCredentialChange == nil {
		return nil, fmt.Errorf("the OnCredentialChange option is required for the callbacks")
	}

	cli, err := kubernetes.NewForConfig(connectionConfig.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client (%w)", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	secret, err := cli.CoreV1().Secrets(secretConfig.Namespace).Get(ctx, secretConfig.Name, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch secret with the name %s in namespaces %s (%w)",
			secretConfig.Name,
			secretConfig.Namespace,
			err,
		)
	}

	conn, err := buildConnection(secret)
	if err != nil {
		return nil, err
	}

	return &oVirtCredentialMonitor{
		cli:          cli,
		secretConfig: secretConfig,
		callbacks:    callbacks,
		secret:       secret,
		lock:         &sync.Mutex{},
		connection:   conn,
		logger:       logger,
	}, nil
}
