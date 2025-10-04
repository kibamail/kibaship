package clusters

import (
	"fmt"
	"os/exec"
)

const (
	// TektonVersion is the version of Tekton Pipelines
	TektonVersion = "v1.4.0"
	// TektonNamespace is the namespace where Tekton is installed
	TektonNamespace = "tekton-pipelines"
	// TektonResolversNamespace is the namespace for Tekton resolvers
	TektonResolversNamespace = "tekton-pipelines-resolvers"
)

// InstallTekton installs Tekton Pipelines using the custom CRD files
func InstallTekton(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Install all Tekton CRD files in order
	for i, crdFile := range TektonCRDFiles {
		if err := applyCRDFile(crdFile.URL, contextName); err != nil {
			return fmt.Errorf("failed to install Tekton CRD %d (%s): %w", i+1, crdFile.Name, err)
		}
	}

	// Wait for Tekton CRDs to be established
	for _, crdName := range TektonCRDNames {
		if err := waitForCRD(crdName, contextName); err != nil {
			return fmt.Errorf("failed waiting for Tekton CRD %s: %w", crdName, err)
		}
	}

	// Wait for Tekton deployments to be ready
	if err := waitForTektonDeployments(contextName); err != nil {
		return fmt.Errorf("tekton deployments not ready: %w", err)
	}

	return nil
}

// waitForTektonDeployments waits for all Tekton deployments to be ready
func waitForTektonDeployments(contextName string) error {
	deployments := []struct {
		name      string
		namespace string
	}{
		{"tekton-pipelines-controller", TektonNamespace},
		{"tekton-events-controller", TektonNamespace},
		{"tekton-pipelines-webhook", TektonNamespace},
		{"tekton-pipelines-remote-resolvers", TektonResolversNamespace},
	}

	for _, deployment := range deployments {
		if err := waitForDeployment(deployment.name, deployment.namespace, contextName); err != nil {
			return fmt.Errorf("deployment %s in namespace %s not ready: %w", deployment.name, deployment.namespace, err)
		}
	}

	return nil
}

// VerifyTekton checks if Tekton is properly installed and running
func VerifyTekton(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Check if Tekton namespaces exist
	namespaces := []string{TektonNamespace, TektonResolversNamespace}
	for _, namespace := range namespaces {
		cmd := exec.Command("kubectl", "get", "namespace", namespace, "--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tekton namespace %s not found: %w", namespace, err)
		}
	}

	// Check if Tekton deployments exist and are ready
	deployments := []struct {
		name      string
		namespace string
	}{
		{"tekton-pipelines-controller", TektonNamespace},
		{"tekton-events-controller", TektonNamespace},
		{"tekton-pipelines-webhook", TektonNamespace},
		{"tekton-pipelines-remote-resolvers", TektonResolversNamespace},
	}

	for _, deployment := range deployments {
		cmd := exec.Command("kubectl", "get", "deployment", deployment.name, "-n", deployment.namespace,
			"--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tekton deployment %s not found in namespace %s: %w", deployment.name, deployment.namespace, err)
		}
	}

	// Check if Tekton CRDs are installed
	for _, crd := range TektonCRDNames {
		cmd := exec.Command("kubectl", "get", "crd", crd, "--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tekton CRD %s not found: %w", crd, err)
		}
	}

	return nil
}

// GetTektonStatus returns the status of Tekton components
func GetTektonStatus(clusterName string) (map[string]string, error) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	status := make(map[string]string)

	deployments := []struct {
		name      string
		namespace string
		key       string
	}{
		{"tekton-pipelines-controller", TektonNamespace, "controller"},
		{"tekton-events-controller", TektonNamespace, "events"},
		{"tekton-pipelines-webhook", TektonNamespace, "webhook"},
		{"tekton-pipelines-remote-resolvers", TektonResolversNamespace, "resolvers"},
	}

	for _, deployment := range deployments {
		cmd := exec.Command("kubectl", "get", "deployment", deployment.name, "-n", deployment.namespace,
			"--context", contextName, "-o", "jsonpath={.status.readyReplicas}/{.status.replicas}")
		if output, err := cmd.Output(); err == nil {
			status[deployment.key] = string(output)
		} else {
			status[deployment.key] = StatusUnknown
		}
	}

	return status, nil
}

// IsTektonInstalled checks if Tekton is already installed
func IsTektonInstalled(clusterName string) bool {
	err := VerifyTekton(clusterName)
	return err == nil
}

// applyCRDFile applies a CRD file from a URL
func applyCRDFile(url, contextName string) error {
	cmd := exec.Command("kubectl", "apply", "-f", url, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply CRD from %s: %w", url, err)
	}
	return nil
}

// InstallValkey installs Valkey Operator using the custom CRD files
func InstallValkey(clusterName string, printProgress, printInfo func(string)) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Install all Valkey CRD files in order
	for i, crdFile := range ValkeyCRDFiles {
		if err := applyCRDFile(crdFile.URL, contextName); err != nil {
			return fmt.Errorf("failed to install Valkey CRD %d (%s): %w", i+1, crdFile.Name, err)
		}
	}

	// Wait for Valkey CRDs to be established
	for _, crdName := range ValkeyCRDNames {
		if err := waitForCRD(crdName, contextName); err != nil {
			return fmt.Errorf("failed waiting for Valkey CRD %s: %w", crdName, err)
		}
	}

	// Monitor Valkey operator pods for 2 minutes with 25-second intervals
	MonitorComponentInstallation(clusterName, "valkey-operator-system", "Valkey Operator", printProgress, printInfo)

	return nil
}

// VerifyValkey checks if Valkey Operator is properly installed and running
func VerifyValkey(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Check if Valkey namespace exists
	cmd := exec.Command("kubectl", "get", "namespace", "valkey-operator-system", "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("valkey namespace not found: %w", err)
	}

	// Check if Valkey operator deployment exists and is ready
	cmd = exec.Command("kubectl", "get", "deployment", "valkey-operator-controller-manager",
		"-n", "valkey-operator-system", "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("valkey operator deployment not found: %w", err)
	}

	// Check if Valkey CRDs are installed
	for _, crd := range ValkeyCRDNames {
		cmd := exec.Command("kubectl", "get", "crd", crd, "--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("valkey CRD %s not found: %w", crd, err)
		}
	}

	return nil
}

// GetValkeyStatus returns the status of Valkey Operator components
func GetValkeyStatus(clusterName string) (map[string]string, error) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	status := make(map[string]string)

	cmd := exec.Command("kubectl", "get", "deployment", "valkey-operator-controller-manager",
		"-n", "valkey-operator-system",
		"--context", contextName, "-o", "jsonpath={.status.readyReplicas}/{.status.replicas}")
	if output, err := cmd.Output(); err == nil {
		status["operator"] = string(output)
	} else {
		status["operator"] = StatusUnknown
	}

	return status, nil
}

// IsValkeyInstalled checks if Valkey Operator is already installed
func IsValkeyInstalled(clusterName string) bool {
	err := VerifyValkey(clusterName)
	return err == nil
}
