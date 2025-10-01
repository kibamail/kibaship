package registryauth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
)

// Validator handles credential validation and namespace access control
type Validator struct {
	k8sClient *K8sClient
	cache     *CredentialCache
}

// NewValidator creates a new credential validator
func NewValidator(k8sClient *K8sClient, cache *CredentialCache) *Validator {
	return &Validator{
		k8sClient: k8sClient,
		cache:     cache,
	}
}

// ValidateCredentials validates username and password against the namespace Secret
// Returns true if credentials are valid
func (v *Validator) ValidateCredentials(ctx context.Context, namespace, username, password string) bool {
	// Check cache first
	if cached, ok := v.cache.Get(namespace); ok {
		if cached.Username == username && subtle.ConstantTimeCompare([]byte(cached.Password), []byte(password)) == 1 {
			log.Printf("auth: cache hit for namespace=%s", namespace)
			return true
		}
	}

	// Cache miss or invalid - fetch from Kubernetes
	// Secret must be named <namespace>-registry-credentials
	secretName := fmt.Sprintf("%s-registry-credentials", namespace)
	creds, err := v.k8sClient.GetCredentials(ctx, namespace, secretName)
	if err != nil {
		log.Printf("auth: failed to get credentials secret %s/%s: %v", namespace, secretName, err)
		return false
	}

	// Validate credentials
	if creds.Username != username {
		log.Printf("auth: username mismatch for namespace=%s (expected=%s, got=%s)", namespace, creds.Username, username)
		return false
	}

	if subtle.ConstantTimeCompare([]byte(creds.Password), []byte(password)) != 1 {
		log.Printf("auth: password mismatch for namespace=%s", namespace)
		return false
	}

	// Valid credentials - cache them
	v.cache.Set(namespace, creds)
	log.Printf("auth: credentials validated for namespace=%s", namespace)
	return true
}
