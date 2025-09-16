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

package streaming

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// secretManager implements SecretManager
type secretManager struct {
	client KubernetesClient
	config *Config
}

// NewSecretManager creates a new secret manager
func NewSecretManager(kubeClient KubernetesClient, config *Config) SecretManager {
	return &secretManager{
		client: kubeClient,
		config: config,
	}
}

// GetValkeyPassword retrieves the password from the Valkey secret
func (s *secretManager) GetValkeyPassword(ctx context.Context) (string, error) {
	log := logf.FromContext(ctx).WithName("secret-manager")

	// Fetch secret from Kubernetes
	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Name:      s.config.ValkeySecretName,
		Namespace: s.config.Namespace,
	}

	log.Info("Fetching Valkey secret", "secret", secretKey)
	err := s.client.Get(ctx, secretKey, secret)
	if err != nil {
		return "", fmt.Errorf("failed to get Valkey secret %s: %w", secretKey, err)
	}

	// Extract password from secret - the secret contains a simple string password
	password, err := s.extractPassword(secret)
	if err != nil {
		return "", fmt.Errorf("failed to extract password from secret %s: %w", secretKey, err)
	}

	log.Info("Successfully retrieved Valkey password")
	return password, nil
}

// extractPassword extracts the password from the secret
// The Valkey secret contains a simple string password as a single field
func (s *secretManager) extractPassword(secret *corev1.Secret) (string, error) {
	if secret.Data == nil {
		return "", fmt.Errorf("secret data is nil")
	}

	if len(secret.Data) == 0 {
		return "", fmt.Errorf("secret data is empty")
	}

	if len(secret.Data) != 1 {
		return "", fmt.Errorf("expected exactly one field in secret, got %d fields", len(secret.Data))
	}

	// Get the single field from the secret - we don't care about the key name
	for _, passwordBytes := range secret.Data {
		password := string(passwordBytes)
		if password == "" {
			return "", fmt.Errorf("secret contains empty password")
		}
		return password, nil
	}

	return "", fmt.Errorf("failed to extract password from secret")
}
