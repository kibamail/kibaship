package config

import (
	"errors"
	"fmt"
	"os"
)

const (
	// EnvWebhookTargetURL is the env var the operator reads for the webhook endpoint.
	EnvWebhookTargetURL = "WEBHOOK_TARGET_URL"

	// WebhookSecretName is the name of the Secret created in the operator namespace
	// that holds the HMAC signing key for webhook payloads.
	WebhookSecretName = "kibaship-webhook-signing"

	// WebhookSecretKey is the key name inside the Secret data map.
	WebhookSecretKey = "secret"
)

// RequireWebhookTargetURL reads the webhook target from env and returns an error
// if it is not set or empty. Use this at startup to enforce configuration.
func RequireWebhookTargetURL() (string, error) {
	url := os.Getenv(EnvWebhookTargetURL)
	if url == "" {
		return "", errors.New(fmt.Sprintf("%s must be set (e.g., http://webhook-receiver.kibaship-operator.svc.cluster.local:8080/webhook)", EnvWebhookTargetURL))
	}
	return url, nil
}
