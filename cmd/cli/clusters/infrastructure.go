package clusters

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Note: Gateway API CRDs are now defined in crds.go

// InstallGatewayAPI installs Gateway API CRDs v1.3.0 using custom Kibaship CRDs
func InstallGatewayAPI(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Install each CRD from the custom list
	for i, crdFile := range GatewayCRDFiles {
		cmd := exec.Command("kubectl", "apply", "-f", crdFile.URL, "--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install Gateway API CRD %d (%s): %w", i+1, crdFile.Name, err)
		}
	}

	// Wait for CRDs to be established
	for _, crdName := range GatewayCRDNames {
		if err := waitForCRD(crdName, contextName); err != nil {
			return fmt.Errorf("failed waiting for CRD %s: %w", crdName, err)
		}
	}

	return nil
}

// InstallCilium installs Cilium CNI via Helm with Gateway API support
func InstallCilium(clusterName, version string, printProgress, printInfo func(string)) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Add Cilium Helm repository
	cmd := exec.Command("helm", "repo", "add", "cilium", "https://helm.cilium.io")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add Cilium Helm repo: %w", err)
	}

	// Update Helm repositories
	cmd = exec.Command("helm", "repo", "update")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update Helm repos: %w", err)
	}

	// Build Helm install command with all required settings
	helmArgs := []string{
		"upgrade", "--install", "cilium", "cilium/cilium",
		"--namespace", "kube-system", "--create-namespace",
		"--version", version,
		"--kube-context", contextName,
		"--set", "kubeProxyReplacement=true",
		"--set", "tunnelProtocol=vxlan",
		"--set", "gatewayAPI.enabled=true",
		"--set", "gatewayAPI.hostNetwork.enabled=true",
		"--set", "gatewayAPI.enableAlpn=true",
		"--set", "gatewayAPI.hostNetwork.nodeLabelSelector=ingress.kibaship.com/ready=true",
		"--set", "gatewayAPI.enableProxyProtocol=true",
		"--set", "gatewayAPI.enableAppProtocol=true",
		"--set", "ipam.mode=kubernetes",
		"--set", "loadBalancer.mode=snat",
		"--set", fmt.Sprintf("k8sServiceHost=%s-control-plane", fullClusterName),
		"--set", "k8sServicePort=6443",
	}

	cmd = exec.Command("helm", helmArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Cilium: %w", err)
	}

	// Monitor Cilium nodes and pods for 2 minutes with 25-second intervals
	MonitorCiliumInstallation(clusterName, "Cilium", printProgress, printInfo)

	return nil
}

// LabelAllNodesForIngress adds the ingress label to all nodes
func LabelAllNodesForIngress(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Label all nodes for ingress
	cmd := exec.Command(
		"kubectl", "label", "nodes", "--all",
		"ingress.kibaship.com/ready=true", "--overwrite", "--context", contextName,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to label nodes for ingress: %w", err)
	}

	return nil
}

// waitForCRD waits for a CRD to be established
func waitForCRD(crdName, context string) error {
	cmd := exec.Command(
		"kubectl", "wait", "--for", "condition=Established",
		"crd", crdName, "--timeout=300s", "--context", context,
	)
	return cmd.Run()
}

// waitForDeployment waits for a Deployment to be ready
func waitForDeployment(name, namespace, context string) error {
	timeout := 10 * time.Minute
	cmd := exec.Command("kubectl", "-n", namespace, "rollout", "status", fmt.Sprintf("deploy/%s", name),
		fmt.Sprintf("--timeout=%s", timeout), "--context", context)
	return cmd.Run()
}

// VerifyGatewayAPI checks if Gateway API CRDs are properly installed
func VerifyGatewayAPI(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	for _, crdName := range GatewayCRDNames {
		cmd := exec.Command("kubectl", "get", "crd", crdName, "--context", contextName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gateway API CRD %s not found: %w", crdName, err)
		}
	}

	return nil
}

// VerifyCilium checks if Cilium is properly installed and running
func VerifyCilium(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	// Check if Cilium DaemonSet exists and is ready
	cmd := exec.Command("kubectl", "get", "ds", "cilium", "-n", "kube-system", "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cilium DaemonSet not found: %w", err)
	}

	// Check if Cilium operator deployment exists and is ready
	cmd = exec.Command("kubectl", "get", "deployment", "cilium-operator", "-n", "kube-system", "--context", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cilium operator deployment not found: %w", err)
	}

	return nil
}

// GetCiliumStatus returns the status of Cilium components
func GetCiliumStatus(clusterName string) (map[string]string, error) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	status := make(map[string]string)

	// Get DaemonSet status
	cmd := exec.Command("kubectl", "get", "ds", "cilium", "-n", "kube-system",
		"--context", contextName, "-o", "jsonpath={.status.numberReady}/{.status.desiredNumberScheduled}")
	if output, err := cmd.Output(); err == nil {
		status["daemonset"] = strings.TrimSpace(string(output))
	} else {
		status["daemonset"] = StatusUnknown
	}

	// Get operator deployment status
	cmd = exec.Command("kubectl", "get", "deployment", "cilium-operator", "-n", "kube-system",
		"--context", contextName, "-o", "jsonpath={.status.readyReplicas}/{.status.replicas}")
	if output, err := cmd.Output(); err == nil {
		status["operator"] = strings.TrimSpace(string(output))
	} else {
		status["operator"] = StatusUnknown
	}

	return status, nil
}
