/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// SecretName is the name of the secret containing the API key
	SecretName = "api-server-api-key-kibaship-com"
	// SecretKey is the key within the secret data
	SecretKey = "api-key"
	// MaxRetryDuration is the maximum time to retry fetching the secret
	MaxRetryDuration = 5 * time.Minute
	// RetryInterval is the interval between retry attempts
	RetryInterval = 15 * time.Second
)

// SecretManager handles retrieving API keys from Kubernetes secrets
type SecretManager struct {
	client    kubernetes.Interface
	namespace string
}

// NewSecretManager creates a new secret manager
func NewSecretManager(namespace string) (*SecretManager, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &SecretManager{
		client:    clientset,
		namespace: namespace,
	}, nil
}

// NewSecretManagerWithClient creates a new secret manager with a custom client (for testing)
func NewSecretManagerWithClient(client kubernetes.Interface, namespace string) *SecretManager {
	return &SecretManager{
		client:    client,
		namespace: namespace,
	}
}

// GetAPIKeyWithRetry retrieves the API key from the secret with retry logic
func (s *SecretManager) GetAPIKeyWithRetry(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, MaxRetryDuration)
	defer cancel()

	ticker := time.NewTicker(RetryInterval)
	defer ticker.Stop()

	// Try immediately first
	if apiKey, err := s.getAPIKey(ctx); err == nil {
		return apiKey, nil
	} else {
		log.Printf("Failed to get API key, will retry: %v", err)
	}

	// Then retry with intervals
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for secret %s in namespace %s: %w", SecretName, s.namespace, ctx.Err())
		case <-ticker.C:
			if apiKey, err := s.getAPIKey(ctx); err == nil {
				log.Printf("Successfully retrieved API key from secret %s", SecretName)
				return apiKey, nil
			} else {
				log.Printf("Retrying to get API key: %v", err)
			}
		}
	}
}

// getAPIKey retrieves the API key from the secret (single attempt)
func (s *SecretManager) getAPIKey(ctx context.Context) (string, error) {
	secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, SecretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", SecretName, err)
	}

	apiKey, exists := secret.Data[SecretKey]
	if !exists {
		return "", fmt.Errorf("secret %s does not contain key %s", SecretName, SecretKey)
	}

	if len(apiKey) == 0 {
		return "", fmt.Errorf("API key in secret %s is empty", SecretName)
	}

	return string(apiKey), nil
}
