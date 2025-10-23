package config

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestLoadConfigFromConfigMapSuccess(t *testing.T) {
	g := NewWithT(t)

	// Create a valid ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyDomain:           "example.com",
			ConfigKeyGatewayClassName: "cilium",
			ConfigKeyWebhookURL:       "https://webhook.example.com/kibaship",
			ConfigKeyACMEEmail:        "admin@example.com",
			ConfigKeyACMEEnv:          "production",
		},
	}

	// Create fake clientset with the ConfigMap
	fakeClientset := fake.NewSimpleClientset(configMap)

	// Create a mock function to return our fake clientset
	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	// Test loading configuration
	config, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(config).NotTo(BeNil())
	g.Expect(config.Domain).To(Equal("example.com"))
	g.Expect(config.GatewayClassName).To(Equal("cilium"))
	g.Expect(config.WebhookURL).To(Equal("https://webhook.example.com/kibaship"))
	g.Expect(config.ACMEEmail).To(Equal("admin@example.com"))
	g.Expect(config.ACMEEnv).To(Equal("production"))
}

func TestLoadConfigFromConfigMapMissingDomain(t *testing.T) {
	g := NewWithT(t)

	// Create ConfigMap missing domain
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyGatewayClassName: "cilium",
			ConfigKeyWebhookURL:       "https://webhook.example.com/kibaship",
			ConfigKeyACMEEmail:        "admin@example.com",
		},
	}

	fakeClientset := fake.NewSimpleClientset(configMap)

	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	_, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("missing required key ingress.domain"))
}

func TestLoadConfigFromConfigMapMissingACMEEmail(t *testing.T) {
	g := NewWithT(t)

	// Create ConfigMap missing ACME email
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyDomain:           "example.com",
			ConfigKeyGatewayClassName: "cilium",
			ConfigKeyWebhookURL:       "https://webhook.example.com/kibaship",
		},
	}

	fakeClientset := fake.NewSimpleClientset(configMap)

	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	_, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("missing required key certs.email"))
}

func TestLoadConfigFromConfigMapMissingGatewayClassName(t *testing.T) {
	g := NewWithT(t)

	// Create ConfigMap missing gateway class name
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyDomain:     "example.com",
			ConfigKeyWebhookURL: "https://webhook.example.com/kibaship",
			ConfigKeyACMEEmail:  "admin@example.com",
		},
	}

	fakeClientset := fake.NewSimpleClientset(configMap)

	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	_, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("missing required key ingress.gateway_classname"))
}

func TestLoadConfigFromConfigMapMissingWebhookURL(t *testing.T) {
	g := NewWithT(t)

	// Create ConfigMap missing webhook URL
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyDomain:           "example.com",
			ConfigKeyGatewayClassName: "cilium",
			ConfigKeyACMEEmail:        "admin@example.com",
		},
	}

	fakeClientset := fake.NewSimpleClientset(configMap)

	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	_, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("missing required key webhooks.url"))
}

func TestLoadConfigFromConfigMapEmptyValues(t *testing.T) {
	g := NewWithT(t)

	// Create ConfigMap with empty values
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyDomain:           "",
			ConfigKeyGatewayClassName: "",
			ConfigKeyWebhookURL:       "",
			ConfigKeyACMEEmail:        "",
		},
	}

	fakeClientset := fake.NewSimpleClientset(configMap)

	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	_, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("missing required key"))
}

func TestLoadConfigFromConfigMapDefaultACMEEnv(t *testing.T) {
	g := NewWithT(t)

	// Create ConfigMap without ACME env (should default to production)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyDomain:           "example.com",
			ConfigKeyGatewayClassName: "cilium",
			ConfigKeyWebhookURL:       "https://webhook.example.com/kibaship",
			ConfigKeyACMEEmail:        "admin@example.com",
			// ConfigKeyACMEEnv is intentionally omitted
		},
	}

	fakeClientset := fake.NewSimpleClientset(configMap)

	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	config, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(config.ACMEEnv).To(Equal("production")) // Should default to production
}

func TestLoadConfigFromConfigMapStagingACMEEnv(t *testing.T) {
	g := NewWithT(t)

	// Create ConfigMap with staging ACME env
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyDomain:           "example.com",
			ConfigKeyGatewayClassName: "cilium",
			ConfigKeyWebhookURL:       "https://webhook.example.com/kibaship",
			ConfigKeyACMEEmail:        "admin@example.com",
			ConfigKeyACMEEnv:          "staging",
		},
	}

	fakeClientset := fake.NewSimpleClientset(configMap)

	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	config, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(config.ACMEEnv).To(Equal("staging"))
}

func TestLoadConfigFromConfigMapInvalidACMEEnv(t *testing.T) {
	g := NewWithT(t)

	// Create ConfigMap with invalid ACME env
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OperatorConfigMapName,
			Namespace: OperatorNamespace,
		},
		Data: map[string]string{
			ConfigKeyDomain:           "example.com",
			ConfigKeyGatewayClassName: "cilium",
			ConfigKeyWebhookURL:       "https://webhook.example.com/kibaship",
			ConfigKeyACMEEmail:        "admin@example.com",
			ConfigKeyACMEEnv:          "invalid",
		},
	}

	fakeClientset := fake.NewSimpleClientset(configMap)

	originalNewForConfig := newForConfigFunc
	defer func() { newForConfigFunc = originalNewForConfig }()
	newForConfigFunc = func(*rest.Config) (kubernetesInterface, error) {
		return fakeClientset, nil
	}

	_, err := LoadConfigFromConfigMap(context.Background(), &rest.Config{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid value for certs.env: invalid"))
	g.Expect(err.Error()).To(ContainSubstring("must be 'production' or 'staging'"))
}
