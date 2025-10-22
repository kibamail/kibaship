package config

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// OperatorConfigMapName is the name of the ConfigMap in the operator namespace
	OperatorConfigMapName = "kibaship-config"

	// OperatorNamespace is the namespace where the operator runs
	OperatorNamespace = "kibaship"

	// ConfigMap keys
	ConfigKeyDomain           = "KIBASHIP_OPERATOR_DOMAIN"
	ConfigKeyACMEEmail        = "KIBASHIP_ACME_EMAIL"
	ConfigKeyWebhookURL       = "WEBHOOK_TARGET_URL"
	ConfigKeyGatewayClassName = "KIBASHIP_GATEWAY_CLASS_NAME"

	// WebhookSecretName is the name of the Secret created in the operator namespace
	// that holds the HMAC signing key for webhook payloads.
	WebhookSecretName = "kibaship-webhook-signing"

	// WebhookSecretKey is the key name inside the Secret data map.
	WebhookSecretKey = "secret"

	// Retry configuration
	maxRetries    = 10
	retryInterval = 5 * time.Second
)

// OperatorConfiguration holds the operator configuration loaded from ConfigMap
type OperatorConfiguration struct {
	Domain           string
	ACMEEmail        string
	WebhookURL       string
	GatewayClassName string
}

// LoadConfigFromConfigMap loads the operator configuration from a ConfigMap
// It will retry up to maxRetries times with retryInterval between attempts
func LoadConfigFromConfigMap(ctx context.Context, kubeConfig *rest.Config) (*OperatorConfiguration, error) {
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	var configMap *corev1.ConfigMap
	var lastErr error

	// Retry loop to wait for ConfigMap to exist
	for i := 0; i < maxRetries; i++ {
		configMap, err = clientset.CoreV1().ConfigMaps(OperatorNamespace).Get(
			ctx,
			OperatorConfigMapName,
			metav1.GetOptions{},
		)
		if err == nil {
			break
		}

		if apierrors.IsNotFound(err) {
			lastErr = fmt.Errorf(
				"ConfigMap %s/%s not found (attempt %d/%d): ensure the ConfigMap exists before starting the operator",
				OperatorNamespace, OperatorConfigMapName, i+1, maxRetries,
			)
			if i < maxRetries-1 {
				time.Sleep(retryInterval)
				continue
			}
		} else {
			return nil, fmt.Errorf("failed to get ConfigMap %s/%s: %w", OperatorNamespace, OperatorConfigMapName, err)
		}
	}

	if configMap == nil {
		return nil, lastErr
	}

	// Extract and validate required fields
	domain, ok := configMap.Data[ConfigKeyDomain]
	if !ok || domain == "" {
		return nil, fmt.Errorf("ConfigMap %s/%s is missing required key %s",
			OperatorNamespace, OperatorConfigMapName, ConfigKeyDomain)
	}

	webhookURL, ok := configMap.Data[ConfigKeyWebhookURL]
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("ConfigMap %s/%s is missing required key %s",
			OperatorNamespace, OperatorConfigMapName, ConfigKeyWebhookURL)
	}

	gatewayClassName, ok := configMap.Data[ConfigKeyGatewayClassName]
	if !ok || gatewayClassName == "" {
		return nil, fmt.Errorf("ConfigMap %s/%s is missing required key %s",
			OperatorNamespace, OperatorConfigMapName, ConfigKeyGatewayClassName)
	}

	// ACMEEmail is optional
	acmeEmail := configMap.Data[ConfigKeyACMEEmail]

	return &OperatorConfiguration{
		Domain:           domain,
		ACMEEmail:        acmeEmail,
		WebhookURL:       webhookURL,
		GatewayClassName: gatewayClassName,
	}, nil
}
