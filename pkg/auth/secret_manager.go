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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// SecretName is the name of the secret containing the API key
	SecretName = "api-server-api-key-kibaship-com"
	// SecretKey is the key within the secret data
	SecretKey = "api-key"
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

// generateAPIKey generates a random 64-character API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 64 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateOrGetAPIKey creates the API key secret if it doesn't exist, or returns the existing one
func (s *SecretManager) CreateOrGetAPIKey(ctx context.Context) (string, error) {
	// First, try to get the existing secret
	if apiKey, err := s.getAPIKey(ctx); err == nil {
		log.Printf("Using existing API key from secret %s", SecretName)
		return apiKey, nil
	}

	// If the secret doesn't exist, create it
	log.Printf("API key secret %s not found, creating new one...", SecretName)

	// Generate a new API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create the secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app":       "kibaship-operator",
				"component": "api-server",
			},
		},
		Data: map[string][]byte{
			SecretKey: []byte(apiKey),
		},
		Type: corev1.SecretTypeOpaque,
	}

	_, err = s.client.CoreV1().Secrets(s.namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Secret was created concurrently, try to get it
			log.Printf("Secret %s was created concurrently, retrieving it...", SecretName)
			return s.getAPIKey(ctx)
		}
		return "", fmt.Errorf("failed to create secret %s: %w", SecretName, err)
	}

	log.Printf("Successfully created API key secret %s", SecretName)
	return apiKey, nil
}
