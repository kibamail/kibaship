package clusters

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	// OperatorNamespace is the namespace where the operator runs
	OperatorNamespace = "kibaship-operator"
	// OperatorConfigMapName is the name of the ConfigMap in the operator namespace
	OperatorConfigMapName = "kibaship-operator-config"
	// OperatorInstallURLTemplate is the template for the operator install URL
	OperatorInstallURLTemplate = "https://raw.githubusercontent.com/kibamail/kibaship-operator/refs/tags/%s/dist/install.yaml"
)

// OperatorConfig holds the configuration needed by the Kibaship operator
type OperatorConfig struct {
	// Domain is the base domain for all application subdomains (required)
	Domain string
	// ACMEEmail is the email address for ACME certificate registration (optional)
	ACMEEmail string
	// WebhookURL is the URL where webhook notifications should be sent (required)
	WebhookURL string
}

// ValidateOperatorConfig validates the operator configuration
func ValidateOperatorConfig(config OperatorConfig) error {
	if config.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	if config.WebhookURL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	// Validate domain format - must be a valid DNS name
	domainRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
	if !domainRegex.MatchString(config.Domain) {
		return fmt.Errorf("invalid domain format: %s - domain must be a valid DNS name (lowercase, alphanumeric, hyphens, dots)", config.Domain)
	}

	// Validate webhook URL format (basic validation)
	if !strings.HasPrefix(config.WebhookURL, "http://") && !strings.HasPrefix(config.WebhookURL, "https://") {
		return fmt.Errorf("webhook URL must start with http:// or https://")
	}

	// Validate email format if provided
	if config.ACMEEmail != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(config.ACMEEmail) {
			return fmt.Errorf("invalid email format: %s", config.ACMEEmail)
		}
	}

	return nil
}

// CreateOperatorNamespace creates the kibaship-operator namespace if it doesn't exist
func CreateOperatorNamespace(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Create namespace YAML
	namespaceYAML := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    name: %s
    app.kubernetes.io/name: kibaship-operator
    app.kubernetes.io/component: operator-namespace`, OperatorNamespace, OperatorNamespace)

	// Apply namespace
	cmd := exec.Command("kubectl", "apply", "-f", "-", "--context", contextName)
	cmd.Stdin = strings.NewReader(namespaceYAML)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create operator namespace: %w", err)
	}

	return nil
}

// CreateOperatorConfigMap creates the required ConfigMap for the operator
func CreateOperatorConfigMap(clusterName string, config OperatorConfig) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Validate configuration first
	if err := ValidateOperatorConfig(config); err != nil {
		return fmt.Errorf("invalid operator configuration: %w", err)
	}

	// Create ConfigMap YAML
	configMapYAML := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: kibaship-operator
    app.kubernetes.io/component: operator-config
data:
  KIBASHIP_OPERATOR_DOMAIN: "%s"
  WEBHOOK_TARGET_URL: "%s"`, OperatorConfigMapName, OperatorNamespace, config.Domain, config.WebhookURL)

	// Add ACME email if provided
	if config.ACMEEmail != "" {
		configMapYAML += fmt.Sprintf(`
  KIBASHIP_ACME_EMAIL: "%s"`, config.ACMEEmail)
	}

	// Apply ConfigMap
	cmd := exec.Command("kubectl", "apply", "-f", "-", "--context", contextName)
	cmd.Stdin = strings.NewReader(configMapYAML)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create operator ConfigMap: %w", err)
	}

	return nil
}

// InstallOperatorConfiguration installs the operator namespace and ConfigMap
func InstallOperatorConfiguration(clusterName string, config OperatorConfig) error {
	// Create operator namespace first
	if err := CreateOperatorNamespace(clusterName); err != nil {
		return fmt.Errorf("failed to create operator namespace: %w", err)
	}

	// Create operator ConfigMap
	if err := CreateOperatorConfigMap(clusterName, config); err != nil {
		return fmt.Errorf("failed to create operator ConfigMap: %w", err)
	}

	return nil
}

// InstallKibashipOperator installs the Kibaship operator using the official install manifest
func InstallKibashipOperator(clusterName, version string, printProgress, printInfo func(string)) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Build the install URL using the provided version
	operatorVersion := version
	if operatorVersion == "dev" {
		// For development builds, check KIBASHIP_VERSION environment variable
		if envVersion := os.Getenv("KIBASHIP_VERSION"); envVersion != "" {
			operatorVersion = envVersion
		} else {
			// Fallback to default version if no env var is set
			operatorVersion = "v0.1.3"
		}
	}
	// Ensure version has 'v' prefix
	if !strings.HasPrefix(operatorVersion, "v") {
		operatorVersion = "v" + operatorVersion
	}

	installURL := fmt.Sprintf(OperatorInstallURLTemplate, operatorVersion)

	// Apply the operator install manifest
	cmd := exec.Command("kubectl", "apply", "-f", installURL, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Kibaship operator from %s: %w", installURL, err)
	}

	// Monitor operator pods for 2 minutes with 25-second intervals
	MonitorComponentInstallation(clusterName, OperatorNamespace, "Kibaship Operator", printProgress, printInfo)

	return nil
}

// waitForOperatorDeployment waits for the operator deployment to be ready
func waitForOperatorDeployment(contextName string) error {
	// The operator deployment is typically named "kibaship-operator-controller-manager"
	// We'll wait for it to be ready
	if err := waitForDeployment("kibaship-operator-controller-manager", OperatorNamespace, contextName); err != nil {
		return fmt.Errorf("operator controller manager deployment not ready: %w", err)
	}

	return nil
}

// VerifyOperatorConfiguration verifies that the operator configuration is properly installed
func VerifyOperatorConfiguration(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Check if operator namespace exists
	cmd := exec.Command("kubectl", "get", "namespace", OperatorNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("operator namespace not found: %w", err)
	}

	// Check if operator ConfigMap exists
	cmd = exec.Command("kubectl", "get", "configmap", OperatorConfigMapName, "-n", OperatorNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("operator ConfigMap not found: %w", err)
	}

	// Verify required keys exist in ConfigMap
	cmd = exec.Command("kubectl", "get", "configmap", OperatorConfigMapName, "-n", OperatorNamespace,
		"--context", contextName, "-o", "jsonpath={.data}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap data: %w", err)
	}

	configData := string(output)
	requiredKeys := []string{"KIBASHIP_OPERATOR_DOMAIN", "WEBHOOK_TARGET_URL"}
	for _, key := range requiredKeys {
		if !strings.Contains(configData, key) {
			return fmt.Errorf("required ConfigMap key %s not found", key)
		}
	}

	return nil
}

// VerifyKibashipOperator verifies that the Kibaship operator is properly installed and running
func VerifyKibashipOperator(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Check if operator deployment exists and is ready
	cmd := exec.Command("kubectl", "get", "deployment", "kibaship-operator-controller-manager",
		"-n", OperatorNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("operator deployment not found: %w", err)
	}

	// Check deployment readiness
	cmd = exec.Command("kubectl", "get", "deployment", "kibaship-operator-controller-manager",
		"-n", OperatorNamespace, "--context", contextName,
		"-o", "jsonpath={.status.readyReplicas}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get deployment status: %w", err)
	}

	if string(output) == "0" || string(output) == "" {
		return fmt.Errorf("operator deployment has no ready replicas")
	}

	return nil
}

// GetOperatorConfigurationStatus returns the status of operator configuration
func GetOperatorConfigurationStatus(clusterName string) (map[string]string, error) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	status := make(map[string]string)

	// Check namespace status
	cmd := exec.Command("kubectl", "get", "namespace", OperatorNamespace, "--context", contextName,
		"-o", "jsonpath={.status.phase}")
	if output, err := cmd.Output(); err == nil {
		status["namespace"] = string(output)
	} else {
		status["namespace"] = "NotFound"
	}

	// Check ConfigMap status
	cmd = exec.Command("kubectl", "get", "configmap", OperatorConfigMapName, "-n", OperatorNamespace,
		"--context", contextName, "-o", "jsonpath={.metadata.name}")
	if output, err := cmd.Output(); err == nil && string(output) == OperatorConfigMapName {
		status["configmap"] = "Ready"
	} else {
		status["configmap"] = "NotFound"
	}

	return status, nil
}

// GetKibashipOperatorStatus returns the status of the Kibaship operator deployment
func GetKibashipOperatorStatus(clusterName string) (map[string]string, error) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	status := make(map[string]string)

	// Check operator deployment status
	cmd := exec.Command("kubectl", "get", "deployment", "kibaship-operator-controller-manager",
		"-n", OperatorNamespace, "--context", contextName,
		"-o", "jsonpath={.status.readyReplicas}/{.status.replicas}")
	if output, err := cmd.Output(); err == nil {
		status["operator"] = string(output)
	} else {
		status["operator"] = StatusUnknown
	}

	return status, nil
}

// IsOperatorConfigurationInstalled checks if operator configuration is already installed
func IsOperatorConfigurationInstalled(clusterName string) bool {
	err := VerifyOperatorConfiguration(clusterName)
	return err == nil
}

// GetDefaultOperatorConfig returns a default operator configuration with placeholders
func GetDefaultOperatorConfig() OperatorConfig {
	return OperatorConfig{
		Domain:     "myapps.kibaship.com",
		ACMEEmail:  "acme@kibaship.com",
		WebhookURL: "https://webhook.example.com/kibaship",
	}
}

// GenerateOperatorConfigExample returns an example of operator configuration
func GenerateOperatorConfigExample() string {
	return `Example operator configuration:

Domain: myapps.kibaship.com
  - Base domain for all application subdomains
  - Applications will be accessible at <app-name>.<domain>
  - Must be a valid DNS name (lowercase, alphanumeric, hyphens, dots)

ACME Email: acme@kibaship.com (optional)
  - Email address for Let's Encrypt certificate registration
  - Used for certificate expiration notifications
  - Can be left empty for development clusters

Webhook URL: https://webhook.example.com/kibaship
  - URL where the operator will send webhook notifications
  - Must be accessible from the cluster
  - Used for project lifecycle events and notifications`
}

// UninstallOperatorConfiguration removes the operator configuration
func UninstallOperatorConfiguration(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Delete ConfigMap
	cmd := exec.Command("kubectl", "delete", "configmap", OperatorConfigMapName, "-n", OperatorNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		// Don't fail if ConfigMap doesn't exist
		fmt.Printf("Warning: failed to delete operator ConfigMap: %v\n", err)
	}

	// Delete namespace (this will also delete the ConfigMap if it still exists)
	cmd = exec.Command("kubectl", "delete", "namespace", OperatorNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		// Don't fail if namespace doesn't exist
		fmt.Printf("Warning: failed to delete operator namespace: %v\n", err)
	}

	return nil
}
