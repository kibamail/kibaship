/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" // nolint:revive,staticcheck
)

// InstallGatewayAPI installs the required Gateway API CRDs and waits for them to be Established
func InstallGatewayAPI() error {
	urls := []string{
		"https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gatewayclasses.yaml",
		"https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gateways.yaml",
		"https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_httproutes.yaml",
		"https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_referencegrants.yaml",
		"https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_grpcroutes.yaml",
		"https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/experimental/gateway.networking.k8s.io_tlsroutes.yaml",
	}

	for _, url := range urls {
		cmd := exec.Command("kubectl", "apply", "-f", url)
		if _, err := Run(cmd); err != nil {
			return err
		}
	}

	crds := []string{
		"gatewayclasses.gateway.networking.k8s.io",
		"gateways.gateway.networking.k8s.io",
		"httproutes.gateway.networking.k8s.io",
		"referencegrants.gateway.networking.k8s.io",
		"grpcroutes.gateway.networking.k8s.io",
		"tlsroutes.gateway.networking.k8s.io",
	}

	for _, crd := range crds {
		cmd := exec.Command("kubectl", "wait", "--for", "condition=Established", "crd", crd, "--timeout=300s")
		if _, err := Run(cmd); err != nil {
			return err
		}
	}
	return nil
}

const (
	prometheusOperatorVersion = "v0.77.1"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"

	certmanagerVersion = "v1.18.2"
	certmanagerURLTmpl = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) (string, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)

	// Stream output in real-time to GinkgoWriter
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%q failed with error: %w", command, err)
	}

	return "", nil
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "apply", "--server-side", "-f", url)
	_, err := Run(cmd)
	return err
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// IsPrometheusCRDsInstalled checks if any Prometheus CRDs are installed
// by verifying the existence of key CRDs related to Prometheus.
func IsPrometheusCRDsInstalled() bool {
	// List of common Prometheus CRDs
	prometheusCRDs := []string{
		"prometheuses.monitoring.coreos.com",
		"prometheusrules.monitoring.coreos.com",
		"prometheusagents.monitoring.coreos.com",
	}

	cmd := exec.Command("kubectl", "get", "crds", "-o", "custom-columns=NAME:.metadata.name")
	output, err := Run(cmd)
	if err != nil {
		return false
	}
	crdList := GetNonEmptyLines(output)
	for _, crd := range prometheusCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}

	// Wait for all cert-manager components to be ready
	components := []string{
		"cert-manager",
		"cert-manager-webhook",
		"cert-manager-cainjector",
	}

	for _, component := range components {
		cmd = exec.Command("kubectl", "wait", "deployment.apps/"+component,
			"--for", "condition=Available",
			"--namespace", "cert-manager",
			"--timeout", "5m",
		)
		if _, err := Run(cmd); err != nil {
			return err
		}
	}

	// // Disable cert-manager webhook validation in test environment
	// // This prevents cert-manager validation webhook from interfering with test deployment
	// cmd = exec.Command("kubectl", "delete",
	// 	"validatingwebhookconfigurations.admissionregistration.k8s.io",
	// 	"cert-manager-webhook")
	// if _, err := Run(cmd); err != nil {
	// 	// Ignore errors if webhook doesn't exist
	// 	warnError(err)
	// }

	return nil
}

// IsCertManagerCRDsInstalled checks if any Cert Manager CRDs are installed
// by verifying the existence of key CRDs related to Cert Manager.
func IsCertManagerCRDsInstalled() bool {
	// List of common Cert Manager CRDs
	certManagerCRDs := []string{
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	}

	// Execute the kubectl command to get all CRDs
	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	// Check if any of the Cert Manager CRDs are present
	crdList := GetNonEmptyLines(output)
	for _, crd := range certManagerCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// LoadImageToKindClusterWithName loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, fmt.Errorf("failed to get current working directory: %w", err)
	}
	wd = strings.ReplaceAll(wd, "/test/e2e", "")
	return wd, nil
}

// UncommentCode searches for target in the file and remove the comment prefix
// of the target content. The target content may span multiple lines.
func UncommentCode(filename, target, prefix string) error {
	// false positive
	// nolint:gosec
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}
	strContent := string(content)

	idx := strings.Index(strContent, target)
	if idx < 0 {
		return fmt.Errorf("unable to find the code %q to be uncomment", target)
	}

	out := new(bytes.Buffer)
	_, err = out.Write(content[:idx])
	if err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewBufferString(target))
	if !scanner.Scan() {
		return nil
	}
	for {
		if _, err = out.WriteString(strings.TrimPrefix(scanner.Text(), prefix)); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
		// Avoid writing a newline in case the previous line was the last in target.
		if !scanner.Scan() {
			break
		}
		if _, err = out.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	if _, err = out.Write(content[idx+len(target):]); err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	// false positive
	// nolint:gosec
	if err = os.WriteFile(filename, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", filename, err)
	}

	return nil
}

// InstallTektonPipelines installs the full Tekton Pipelines operator
func InstallTektonPipelines() error {
	url := "https://storage.googleapis.com/tekton-releases/pipeline/previous/v1.4.0/release.yaml"
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}

	// Wait for Tekton Pipelines components to be ready (increased timeout for CI environments)
	components := []string{
		"tekton-pipelines-controller",
		"tekton-pipelines-webhook",
	}

	for _, component := range components {
		cmd = exec.Command("kubectl", "wait", "deployment.apps/"+component,
			"--for", "condition=Available",
			"--namespace", "tekton-pipelines",
			"--timeout", "15m",
		)
		if _, err := Run(cmd); err != nil {
			return err
		}
	}

	// Also wait for remote resolvers to be ready
	cmd = exec.Command("kubectl", "wait", "deployment.apps/tekton-pipelines-remote-resolvers",
		"--for", "condition=Available",
		"--namespace", "tekton-pipelines-resolvers",
		"--timeout", "15m",
	)
	if _, err := Run(cmd); err != nil {
		return err
	}

	return nil
}

// UninstallTektonPipelines uninstalls the full Tekton Pipelines operator
func UninstallTektonPipelines() {
	url := "https://storage.googleapis.com/tekton-releases/pipeline/previous/v1.4.0/release.yaml"
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// IsTektonPipelinesCRDsInstalled checks if any Tekton Pipelines CRDs are installed
func IsTektonPipelinesCRDsInstalled() bool {
	tektonCRDs := []string{
		"tasks.tekton.dev",
		"taskruns.tekton.dev",
		"pipelines.tekton.dev",
		"pipelineruns.tekton.dev",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	for _, crd := range tektonCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// InstallValkeyOperator installs the full Valkey operator
func InstallValkeyOperator() error {
	url := "https://github.com/hyperspike/valkey-operator/releases/download/v0.0.59/install.yaml"
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}

	// Wait for Valkey operator components to be ready (increased timeout for CI environments)
	cmd = exec.Command("kubectl", "wait", "deployment.apps/valkey-operator-controller-manager",
		"--for", "condition=Available",
		"--namespace", "valkey-operator-system",
		"--timeout", "5m",
	)
	if _, err := Run(cmd); err != nil {
		return err
	}

	return nil
}

// UninstallValkeyOperator uninstalls the full Valkey operator
func UninstallValkeyOperator() {
	url := "https://github.com/hyperspike/valkey-operator/releases/download/v0.0.59/install.yaml"
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// IsValkeyOperatorCRDsInstalled checks if any Valkey operator CRDs are installed
func IsValkeyOperatorCRDsInstalled() bool {
	valkeyCRDs := []string{
		"valkeys.hyperspike.io",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	for _, crd := range valkeyCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// CreateStorageReplicaStorageClass creates the storage-replica-1 storage class
// that duplicates the default 'standard' storage class for test environments
func CreateStorageReplicaStorageClass() error {
	storageClassYAML := `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: storage-replica-1
  annotations:
    description: "Test environment storage class that mirrors standard storage class"
provisioner: rancher.io/local-path
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: false`

	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(storageClassYAML)
	_, err := Run(cmd)
	return err
}

// CleanupStorageReplicaStorageClass removes the storage-replica-1 storage class
func CleanupStorageReplicaStorageClass() {
	cmd := exec.Command("kubectl", "delete", "storageclass", "storage-replica-1", "--ignore-not-found")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// DeployKibashipOperator deploys the kibaship-operator using make deploy
func DeployKibashipOperator() error {
	// First install the CRDs
	cmd := exec.Command("make", "install")
	if _, err := Run(cmd); err != nil {
		return err
	}

	// Wait for CRDs to be available and established
	crdNames := []string{
		"projects.platform.operator.kibaship.com",
		"applications.platform.operator.kibaship.com",
		"deployments.platform.operator.kibaship.com",
		"applicationdomains.platform.operator.kibaship.com",
	}

	for _, crdName := range crdNames {
		cmd = exec.Command("kubectl", "wait", "--for", "condition=Established", "crd", crdName, "--timeout=300s")
		if _, err := Run(cmd); err != nil {
			return err
		}
	}

	// Then deploy the operator
	cmd = exec.Command("make", "deploy", "IMG=kibaship.com/kibaship-operator:v0.0.1")
	if _, err := Run(cmd); err != nil {
		return err
	}

	// Use enhanced monitoring with longer timeout to account for Valkey initialization
	maxWaitTime := 8 * time.Minute // Increased from 4 minutes to accommodate Valkey startup
	return MonitorOperatorStartup("kibaship-operator", maxWaitTime)
}

// DeployAPIServer deploys the API server into the operator namespace and sets its image
func DeployAPIServer(image string) error {
	// Apply the api-server kustomization
	cmd := exec.Command("kubectl", "apply", "-k", "config/api-server")
	if _, err := Run(cmd); err != nil {
		return err
	}

	// Update the deployment image to the locally built tag
	cmd = exec.Command("kubectl", "-n", "kibaship-operator", "set", "image",
		"deployment/apiserver", fmt.Sprintf("apiserver=%s", image))
	if _, err := Run(cmd); err != nil {
		return err
	}

	// Wait for rollout to complete
	cmd = exec.Command("kubectl", "-n", "kibaship-operator", "rollout", "status",
		"deployment/apiserver", "--timeout=5m")
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

// UndeployKibashipOperator removes the kibaship-operator deployment
func UndeployKibashipOperator() {
	cmd := exec.Command("make", "undeploy")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// CheckOperatorHealthStatus provides detailed operator health check diagnostics
func CheckOperatorHealthStatus(namespace string) {
	_, _ = fmt.Fprintf(GinkgoWriter, "\n=== Detailed Operator Health Check ===\n")

	// Get pod logs with specific focus on health probe failures
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-o", "name")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Error getting pods: %v\n", err)
		return
	}

	podNames := GetNonEmptyLines(string(output))
	for _, podName := range podNames {
		if strings.Contains(podName, "controller-manager") {
			_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Pod Logs for %s ---\n", podName)

			// Get recent logs with timestamps
			cmd = exec.Command("kubectl", "logs", podName, "-n", namespace, "--timestamps", "--tail=50")
			if _, err := Run(cmd); err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Error getting logs for %s: %v\n", podName, err)
			}

			// Check health endpoints directly
			_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Health Endpoint Status ---\n")
			CheckOperatorHealthEndpoints(namespace, strings.TrimPrefix(podName, "pod/"))

			// Check Valkey dependency status
			_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Valkey Dependency Status ---\n")
			CheckValkeyStatus(namespace)
		}
	}
}

// CheckOperatorHealthEndpoints tests the health endpoints directly
func CheckOperatorHealthEndpoints(namespace, podName string) {
	healthEndpoints := []string{"/healthz", "/readyz"}

	for _, endpoint := range healthEndpoints {
		_, _ = fmt.Fprintf(GinkgoWriter, "Testing %s endpoint on pod %s:\n", endpoint, podName)

		// Use kubectl exec to curl the health endpoint from within the pod
		cmd := exec.Command("kubectl", "exec", podName, "-n", namespace, "--",
			"wget", "-qO-", "--timeout=5", fmt.Sprintf("http://localhost:8081%s", endpoint))
		output, err := cmd.CombinedOutput()

		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "  ❌ %s failed: %v\n", endpoint, err)
			_, _ = fmt.Fprintf(GinkgoWriter, "  Output: %s\n", string(output))
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "  ✅ %s success: %s\n", endpoint, strings.TrimSpace(string(output)))
		}
	}
}

// CheckValkeyStatus checks if Valkey cluster is ready and accessible
func CheckValkeyStatus(operatorNamespace string) {
	// First, show ALL pods in the operator namespace for full context
	_, _ = fmt.Fprintf(GinkgoWriter, "\n=== ALL PODS IN NAMESPACE %s ===\n", operatorNamespace)
	cmd := exec.Command("kubectl", "get", "pods", "-n", operatorNamespace, "-o", "wide", "--show-labels")
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "  Error listing pods in namespace %s: %v\n", operatorNamespace, err)
	}

	// Also show pod status details
	_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Pod Status Details ---\n")
	cmd = exec.Command("kubectl", "describe", "pods", "-n", operatorNamespace)
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "  Error describing pods: %v\n", err)
	}

	// Show all services (including Valkey services)
	_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Services in Namespace ---\n")
	cmd = exec.Command("kubectl", "get", "svc", "-n", operatorNamespace, "-o", "wide")
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "  Error listing services: %v\n", err)
	}

	// Check if Valkey CRD exists
	_, _ = fmt.Fprintf(GinkgoWriter, "\n=== VALKEY CLUSTER STATUS ===\n")
	_, _ = fmt.Fprintf(GinkgoWriter, "Checking Valkey CRDs:\n")
	cmd = exec.Command("kubectl", "get", "crd", "valkeys.hyperspike.io")
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "  ❌ Valkey CRD not found: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "  ✅ Valkey CRD exists\n")

	// List ALL Valkey resources in the namespace first
	_, _ = fmt.Fprintf(GinkgoWriter, "\nAll Valkey resources in namespace %s:\n", operatorNamespace)
	cmd = exec.Command("kubectl", "get", "valkey", "-n", operatorNamespace, "-o", "wide")
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "  No Valkey resources found or error: %v\n", err)
	}

	// Check if expected Valkey cluster resource exists
	expectedValkeyName := "kibaship-valkey-cluster-kibaship-com"
	_, _ = fmt.Fprintf(GinkgoWriter, "\nChecking expected Valkey cluster resource '%s':\n", expectedValkeyName)
	cmd = exec.Command("kubectl", "get", "valkey", expectedValkeyName, "-n", operatorNamespace, "-o", "yaml")
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "  ❌ Expected Valkey cluster resource not found: %v\n", err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "  ✅ Valkey cluster resource exists (YAML above shows full status)\n")
	}

	// Show recent events related to Valkey
	_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Recent Events (may show Valkey cluster status) ---\n")
	cmd = exec.Command("kubectl", "get", "events", "-n", operatorNamespace,
		"--sort-by=.metadata.creationTimestamp", "--field-selector=involvedObject.kind=Valkey")
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "  No Valkey-specific events or error: %v\n", err)
	}

	// Check all secrets (including Valkey secret)
	_, _ = fmt.Fprintf(GinkgoWriter, "\n--- All Secrets in Namespace ---\n")
	cmd = exec.Command("kubectl", "get", "secrets", "-n", operatorNamespace, "-o", "wide")
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "    Error listing secrets: %v\n", err)
	}

	// Check expected Valkey secret specifically
	expectedSecretName := "kibaship-valkey-cluster-kibaship-com"
	_, _ = fmt.Fprintf(GinkgoWriter, "\nChecking expected Valkey authentication secret '%s':\n", expectedSecretName)
	cmd = exec.Command("kubectl", "describe", "secret", expectedSecretName, "-n", operatorNamespace)
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "  ❌ Expected Valkey secret not found: %v\n", err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "  ✅ Valkey secret exists (details above)\n")
	}
}

// MonitorOperatorStartup monitors operator startup with enhanced logging
func MonitorOperatorStartup(namespace string, timeout time.Duration) error {
	_, _ = fmt.Fprintf(GinkgoWriter, "\n=== Enhanced Operator Startup Monitoring ===\n")
	_, _ = fmt.Fprintf(GinkgoWriter, "Monitoring namespace: %s\n", namespace)
	_, _ = fmt.Fprintf(GinkgoWriter, "Timeout: %v\n", timeout)

	startTime := time.Now()
	checkInterval := 10 * time.Second

	for time.Since(startTime) < timeout {
		elapsed := time.Since(startTime)
		_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Status Check (%.0fs elapsed) ---\n", elapsed.Seconds())

		// Check deployment readiness
		cmd := exec.Command("kubectl", "get", "deployment", "kibaship-operator-controller-manager",
			"-n", namespace, "-o", "jsonpath={.status.readyReplicas}")
		output, err := cmd.CombinedOutput()
		readyReplicas := strings.TrimSpace(string(output))

		if err == nil && readyReplicas == "1" {
			_, _ = fmt.Fprintf(GinkgoWriter, "✅ Operator is ready!\n")
			return nil
		}

		_, _ = fmt.Fprintf(GinkgoWriter, "Current ready replicas: %s\n", readyReplicas)

		// Show all pods in namespace every 30 seconds to track Valkey startup progress
		if int(elapsed.Seconds())%30 == 0 || elapsed > 30*time.Second {
			_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Namespace Pod Status (%.0fs elapsed) ---\n", elapsed.Seconds())
			cmd = exec.Command("kubectl", "get", "pods", "-n", namespace, "-o", "wide")
			if _, err := Run(cmd); err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Error getting namespace pods: %v\n", err)
			}

			// Show Valkey resources specifically every 30 seconds
			_, _ = fmt.Fprintf(GinkgoWriter, "\n--- Valkey Resources Status ---\n")
			cmd = exec.Command("kubectl", "get", "valkey", "-n", namespace, "-o", "wide")
			if _, err := Run(cmd); err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "No Valkey resources or error: %v\n", err)
			}
		}

		// Detailed health check every 60 seconds or if not ready after 1 minute
		if int(elapsed.Seconds())%60 == 0 || elapsed > time.Minute {
			CheckOperatorHealthStatus(namespace)
		}

		_, _ = fmt.Fprintf(GinkgoWriter, "⏳ Waiting %v before next check...\n", checkInterval)
		time.Sleep(checkInterval)
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "\n❌ Timeout reached - performing final diagnostic check\n")
	CheckOperatorHealthStatus(namespace)

	return fmt.Errorf("timeout waiting for operator to become ready after %v", timeout)
}

// DiagnoseKibashipOperator provides a complete diagnostic check for the kibaship operator
// This function can be called from any test to get detailed operator status
func DiagnoseKibashipOperator() {
	_, _ = fmt.Fprintf(GinkgoWriter, "\n🔍 KIBASHIP OPERATOR DIAGNOSTIC REPORT\n")
	_, _ = fmt.Fprintf(GinkgoWriter, "=====================================\n")

	// Check general operator status
	CheckOperatorHealthStatus("kibaship-operator")

	// Additional context: Check dependencies
	_, _ = fmt.Fprintf(GinkgoWriter, "\n=== DEPENDENCY STATUS ===\n")

	// Check all operators that kibaship depends on
	operators := []struct {
		name      string
		namespace string
	}{
		{"cert-manager", "cert-manager"},
		{"tekton-pipelines", "tekton-pipelines"},
		{"valkey-operator", "valkey-operator-system"},
	}

	for _, op := range operators {
		_, _ = fmt.Fprintf(GinkgoWriter, "\n--- %s Status ---\n", op.name)
		cmd := exec.Command("kubectl", "get", "deployment", "-n", op.namespace)
		if _, err := Run(cmd); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "❌ Error getting %s deployments: %v\n", op.name, err)
		}
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "\n🏁 DIAGNOSTIC REPORT COMPLETE\n")
}
