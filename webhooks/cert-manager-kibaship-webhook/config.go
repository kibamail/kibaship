package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

// WebhookConfig is the user-provided config embedded in Issuer/ClusterIssuer
// under spec.acme.solvers[].dns01.webhook.config and passed to the solver
// via ChallengeRequest.Config. Keep fields minimal; defaults applied in code.
type WebhookConfig struct {
	Valkey  ValkeyConfig       `json:"valkey"`
	UIHints map[string]string  `json:"uiHints,omitempty"`
}

type ValkeyConfig struct {
	// Addr is the seed address for the Valkey/Redis Cluster, e.g. "valkey.namespace.svc:6379"
	Addr string `json:"addr,omitempty"`
	// Username is optional (ACL); most deployments use only password.
	Username string `json:"username,omitempty"`
	// PasswordSecretRef points to a Secret containing the password under a key.
	PasswordSecretRef *SecretRef `json:"passwordSecretRef,omitempty"`
	// TLS enables TLS to the Valkey endpoint (optional, default false).
	TLS bool `json:"tls,omitempty"`
	// Optional timeouts as strings (e.g., "2s", "500ms").
	ConnectTimeout string `json:"connectTimeout,omitempty"`
	ReadTimeout    string `json:"readTimeout,omitempty"`
	WriteTimeout   string `json:"writeTimeout,omitempty"`
}

// SecretRef identifies a Secret key. Namespace is optional; if empty, the
// caller should default to the Challenge namespace or configured default.
type SecretRef struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	Namespace string `json:"namespace,omitempty"`
}

// parseConfig parses the raw JSON configuration passed by cert-manager.
// raw may be nil/empty, in which case sensible defaults are returned.
func parseConfig(raw []byte) (WebhookConfig, error) {
	cfg := WebhookConfig{}
	if len(raw) == 0 || string(raw) == "null" {
		// No config provided; return zero-value with empty Valkey address.
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return WebhookConfig{}, fmt.Errorf("invalid webhook config: %w", err)
	}
	return cfg, validateConfig(cfg)
}

func validateConfig(cfg WebhookConfig) error {
	// For now, config is optional; we'll later allow addressing via labels only.
	if cfg.Valkey.PasswordSecretRef != nil {
		if cfg.Valkey.PasswordSecretRef.Name == "" || cfg.Valkey.PasswordSecretRef.Key == "" {
			return errors.New("valkey.passwordSecretRef.name and .key are required when passwordSecretRef is set")
		}
	}
	return nil
}

