package registryauth

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// K8sClient wraps Kubernetes client for reading Secrets
type K8sClient struct {
	clientset kubernetes.Interface
}

// Credentials represents username and password
type Credentials struct {
	Username string
	Password string
}

// NewK8sClient creates a new Kubernetes client using in-cluster config
func NewK8sClient() (*K8sClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &K8sClient{clientset: clientset}, nil
}

// GetCredentials reads credentials from a Secret in the specified namespace
func (k *K8sClient) GetCredentials(ctx context.Context, namespace, secretName string) (*Credentials, error) {
	secret, err := k.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
	}

	username, ok := secret.Data["username"]
	if !ok || len(username) == 0 {
		return nil, fmt.Errorf("secret %s/%s missing 'username' key", namespace, secretName)
	}

	password, ok := secret.Data["password"]
	if !ok || len(password) == 0 {
		return nil, fmt.Errorf("secret %s/%s missing 'password' key", namespace, secretName)
	}

	return &Credentials{
		Username: string(username),
		Password: string(password),
	}, nil
}
