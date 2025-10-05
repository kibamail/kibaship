package clusters

import (
	"fmt"
	"os/exec"
	"time"
)

const (
	// LonghornVersion is the version of Longhorn to install
	LonghornVersion = "v1.10.0"
	// LonghornNamespace is the namespace where Longhorn is installed
	LonghornNamespace = "longhorn-system"
	// LonghornManifestURL is the URL for Longhorn installation manifest
	LonghornManifestURL = "https://raw.githubusercontent.com/longhorn/longhorn/v1.10.0/deploy/longhorn.yaml"
)

// InstallLonghorn installs Longhorn storage system
func InstallLonghorn(clusterName string, printProgress, printInfo func(string)) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Install Longhorn using the official manifest
	cmd := exec.Command("kubectl", "apply", "-f", LonghornManifestURL, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Longhorn: %w", err)
	}

	// Wait for Longhorn namespace to be created
	if err := waitForNamespace(LonghornNamespace, contextName, 2*time.Minute); err != nil {
		return fmt.Errorf("longhorn namespace not created: %w", err)
	}

	// Monitor Longhorn pods for 2 minutes with 25-second intervals
	MonitorComponentInstallation(clusterName, LonghornNamespace, "Longhorn", printProgress, printInfo)

	return nil
}

// VerifyLonghorn checks if Longhorn is properly installed and running
func VerifyLonghorn(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Check if Longhorn namespace exists
	cmd := exec.Command("kubectl", "get", "namespace", LonghornNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("longhorn namespace not found: %w", err)
	}

	// Check if Longhorn deployments exist and are ready
	deployments := []string{"longhorn-ui", "longhorn-manager", "longhorn-driver-deployer"}
	for _, deployment := range deployments {
		cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", LonghornNamespace, "--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("longhorn deployment %s not found: %w", deployment, err)
		}
	}

	// Check if Longhorn manager DaemonSet exists
	cmd = exec.Command("kubectl", "get", "daemonset", "longhorn-manager", "-n",
		LonghornNamespace, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("longhorn-manager DaemonSet not found: %w", err)
	}

	// Check if Longhorn StorageClass exists
	cmd = exec.Command("kubectl", "get", "storageclass", "longhorn", "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("longhorn StorageClass not found: %w", err)
	}

	return nil
}

// GetLonghornStatus returns the status of Longhorn components
func GetLonghornStatus(clusterName string) (map[string]string, error) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	status := make(map[string]string)

	deployments := []string{"longhorn-ui", "longhorn-manager", "longhorn-driver-deployer"}

	for _, deployment := range deployments {
		cmd := exec.Command("kubectl", "get", "deployment", deployment, "-n", LonghornNamespace,
			"--context", contextName, "-o", "jsonpath={.status.readyReplicas}/{.status.replicas}")
		if output, err := cmd.Output(); err == nil {
			status[deployment] = string(output)
		} else {
			status[deployment] = StatusUnknown
		}
	}

	// Get DaemonSet status
	cmd := exec.Command("kubectl", "get", "daemonset", "longhorn-manager", "-n", LonghornNamespace,
		"--context", contextName, "-o", "jsonpath={.status.numberReady}/{.status.desiredNumberScheduled}")
	if output, err := cmd.Output(); err == nil {
		status["longhorn-manager-ds"] = string(output)
	} else {
		status["longhorn-manager-ds"] = StatusUnknown
	}

	return status, nil
}

// IsLonghornInstalled checks if Longhorn is already installed
func IsLonghornInstalled(clusterName string) bool {
	err := VerifyLonghorn(clusterName)
	return err == nil
}

// UninstallLonghorn removes Longhorn from the cluster
func UninstallLonghorn(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Delete Longhorn using the same manifest
	cmd := exec.Command("kubectl", "delete", "-f", LonghornManifestURL, "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall Longhorn: %w", err)
	}

	return nil
}

// waitForNamespace waits for a namespace to be created and ready
func waitForNamespace(namespace, contextName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		cmd := exec.Command("kubectl", "get", "namespace", namespace, "--context", contextName)
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("namespace %s not ready within %v", namespace, timeout)
}

// GetLonghornUI returns the command to access Longhorn UI
func GetLonghornUI(clusterName string) string {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	return fmt.Sprintf("kubectl port-forward -n %s svc/longhorn-frontend 8080:80 --context %s",
		LonghornNamespace, contextName)
}

// GetLonghornInfo returns useful information about Longhorn installation
func GetLonghornInfo() []string {
	return []string{
		"Longhorn provides distributed block storage for Kubernetes",
		"Default StorageClass: longhorn",
		"UI Access: Use port-forward to access the Longhorn UI",
		"Backup: Supports backup to S3, NFS, and other storage backends",
		"Snapshots: Supports volume snapshots and cloning",
	}
}
