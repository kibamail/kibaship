package clusters

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// CertManagerVersion is the version of cert-manager to install
	CertManagerVersion = "v1.18.2"
	// CertManagerNamespace is the namespace where cert-manager is installed
	CertManagerNamespace = "cert-manager"
	// CertManagerHelmRepo is the Helm repository for cert-manager
	CertManagerHelmRepo = "https://charts.jetstack.io"
)

// CertManagerConfig holds the configuration for cert-manager installation
type CertManagerConfig struct {
	Version           string
	ReplicaCount      int
	WebhookReplicas   int
	CRDsEnabled       bool
	PrometheusEnabled bool
}

// DefaultCertManagerConfig returns the default cert-manager configuration
func DefaultCertManagerConfig() CertManagerConfig {
	return CertManagerConfig{
		Version:           CertManagerVersion,
		ReplicaCount:      3,
		WebhookReplicas:   2,
		CRDsEnabled:       true,
		PrometheusEnabled: true,
	}
}

// InstallCertManager installs cert-manager using Helm with the specified configuration
func InstallCertManager(clusterName string, config CertManagerConfig, printProgress, printInfo func(string)) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Add cert-manager Helm repository
	if err := addCertManagerHelmRepo(); err != nil {
		return fmt.Errorf("failed to add cert-manager Helm repo: %w", err)
	}

	// Update Helm repositories
	if err := updateHelmRepos(); err != nil {
		return fmt.Errorf("failed to update Helm repos: %w", err)
	}

	// Install cert-manager with Helm
	if err := installCertManagerWithHelm(contextName, config); err != nil {
		return fmt.Errorf("failed to install cert-manager: %w", err)
	}

	// Monitor cert-manager pods for 2 minutes with 25-second intervals
	MonitorComponentInstallation(clusterName, CertManagerNamespace, "cert-manager", printProgress, printInfo)

	// Verify cert-manager webhook is functional
	if err := verifyCertManagerWebhook(contextName); err != nil {
		return fmt.Errorf("cert-manager webhook verification failed: %w", err)
	}

	return nil
}

// addCertManagerHelmRepo adds the cert-manager Helm repository
func addCertManagerHelmRepo() error {
	cmd := exec.Command("helm", "repo", "add", "jetstack", CertManagerHelmRepo)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add jetstack Helm repo: %w", err)
	}
	return nil
}

// updateHelmRepos updates all Helm repositories
func updateHelmRepos() error {
	cmd := exec.Command("helm", "repo", "update")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update Helm repos: %w", err)
	}
	return nil
}

// installCertManagerWithHelm installs cert-manager using Helm
func installCertManagerWithHelm(contextName string, config CertManagerConfig) error {
	// Build Helm install command with all required settings
	helmArgs := []string{
		"upgrade", "--install", "cert-manager", "jetstack/cert-manager",
		"--namespace", CertManagerNamespace, "--create-namespace",
		"--version", config.Version,
		"--kube-context", contextName,
		"--set", fmt.Sprintf("replicaCount=%d", config.ReplicaCount),
		"--set", fmt.Sprintf("crds.enabled=%t", config.CRDsEnabled),
		"--set", fmt.Sprintf("prometheus.enabled=%t", config.PrometheusEnabled),
		"--set", fmt.Sprintf("webhook.replicaCount=%d", config.WebhookReplicas),
		"--wait",
		"--timeout=10m",
	}

	cmd := exec.Command("helm", helmArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install cert-manager with Helm: %w", err)
	}

	return nil
}

// waitForCertManagerDeployments waits for all cert-manager deployments to be ready
func waitForCertManagerDeployments(contextName string) error {
	deployments := []string{
		"cert-manager",
		"cert-manager-cainjector",
		"cert-manager-webhook",
	}

	for _, deployment := range deployments {
		if err := waitForDeployment(deployment, CertManagerNamespace, contextName); err != nil {
			return fmt.Errorf("deployment %s not ready: %w", deployment, err)
		}
	}

	return nil
}

// verifyCertManagerWebhook verifies that the cert-manager webhook is functional
func verifyCertManagerWebhook(contextName string) error {
	// Test webhook by creating a test ClusterIssuer
	testIssuerYAML := `apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: test-selfsigned-issuer
spec:
  selfSigned: {}`

	// Apply test issuer
	cmd := exec.Command("kubectl", "apply", "-f", "-", "--context", contextName)
	cmd.Stdin = strings.NewReader(testIssuerYAML)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create test ClusterIssuer: %w", err)
	}

	// Wait for issuer to be ready
	cmd = exec.Command("kubectl", "wait", "--for", "condition=Ready", "clusterissuer", "test-selfsigned-issuer",
		"--timeout=60s", "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("test ClusterIssuer not ready: %w", err)
	}

	// Clean up test issuer
	cmd = exec.Command("kubectl", "delete", "clusterissuer", "test-selfsigned-issuer", "--context", contextName)
	if err := cmd.Run(); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to clean up test ClusterIssuer: %v\n", err)
	}

	return nil
}

// VerifyCertManager checks if cert-manager is properly installed and running
func VerifyCertManager(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Check if cert-manager namespace exists
	cmd := exec.Command("kubectl", "get", "namespace", CertManagerNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cert-manager namespace not found: %w", err)
	}

	// Check if cert-manager deployments exist and are ready
	deployments := []string{"cert-manager", "cert-manager-cainjector", "cert-manager-webhook"}
	for _, deployment := range deployments {
		cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", CertManagerNamespace, "--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cert-manager deployment %s not found: %w", deployment, err)
		}
	}

	// Check if cert-manager CRDs are installed
	crds := []string{"certificates.cert-manager.io", "clusterissuers.cert-manager.io", "issuers.cert-manager.io"}
	for _, crd := range crds {
		cmd := exec.Command("kubectl", "get", "crd", crd, "--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cert-manager CRD %s not found: %w", crd, err)
		}
	}

	return nil
}

// GetCertManagerStatus returns the status of cert-manager components
func GetCertManagerStatus(clusterName string) (map[string]string, error) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	status := make(map[string]string)

	deployments := []string{"cert-manager", "cert-manager-cainjector", "cert-manager-webhook"}

	for _, deployment := range deployments {
		cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", CertManagerNamespace,
			"--context", contextName, "-o", "jsonpath={.status.readyReplicas}/{.status.replicas}")
		if output, err := cmd.Output(); err == nil {
			status[deployment] = strings.TrimSpace(string(output))
		} else {
			status[deployment] = StatusUnknown
		}
	}

	return status, nil
}

// IsCertManagerInstalled checks if cert-manager is already installed
func IsCertManagerInstalled(clusterName string) bool {
	err := VerifyCertManager(clusterName)
	return err == nil
}

// UninstallCertManager removes cert-manager from the cluster
func UninstallCertManager(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Uninstall cert-manager using Helm
	cmd := exec.Command("helm", "uninstall", "cert-manager", "-n", CertManagerNamespace, "--kube-context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall cert-manager: %w", err)
	}

	// Delete namespace
	cmd = exec.Command("kubectl", "delete", "namespace", CertManagerNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		// Log warning but don't fail - namespace might have finalizers
		fmt.Printf("Warning: failed to delete cert-manager namespace: %v\n", err)
	}

	return nil
}
