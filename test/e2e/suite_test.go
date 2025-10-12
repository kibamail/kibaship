package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kibamail/kibaship-operator/test/utils"
)

const (
	readyPhase             = "Ready"
	testProjectName        = "test-project-reconciliation-e2e"
	deploymentResourceType = "deployments.platform.operator.kibaship.com"
	finalizersPath         = "jsonpath={.metadata.finalizers[0]}"
	statusPhasePath        = "jsonpath={.status.phase}"
	labelsPath             = "jsonpath={.metadata.labels}"
	uuidLabelPath          = "jsonpath={.metadata.labels.platform\\.kibaship\\.com/uuid}"
	deploymentUUIDLabel    = "platform.kibaship.com/deployment-uuid=789e4567-e89b-12d3-a456-426614174003"
	gitProviderPath        = "jsonpath={.spec.gitRepository.provider}"
	gitPublicAccessPath    = "jsonpath={.spec.gitRepository.publicAccess}"
	gitRepositoryPath      = "jsonpath={.spec.gitRepository.repository}"
	projectRefPath         = "jsonpath={.spec.projectRef.name}"
	applicationRefPath     = "jsonpath={.spec.applicationRef.name}"
	gitCommitSHAPath       = "jsonpath={.spec.gitRepository.commitSHA}"
	gitBranchPath          = "jsonpath={.spec.gitRepository.branch}"
	// shared test constants
	workspaceUUIDConst = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	kindServiceAccount = "ServiceAccount"
	kindRole           = "Role"
)

var (
	projectImage             = "kibaship.com/kibaship-operator:v0.0.1"
	projectImageAPIServer    = "kibaship.com/kibaship-operator-apiserver:v0.0.1"
	projectImageCertWebhook  = "kibaship.com/kibaship-operator-cert-manager-webhook:v0.0.1"
	projectImageRailpackCLI  = "kibaship.com/kibaship-railpack-cli:v0.0.1"
	projectImageRailpackBld  = "kibaship.com/kibaship-railpack-build:v0.0.1"
	projectImageRegistryAuth = "kibaship.com/kibaship-operator-registry-auth:v0.0.1"
)

// getKubernetesClient creates a Kubernetes client using the current context
func getKubernetesClient() (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}
	return clientset, nil
}

// namespaceExists checks if a namespace exists in the cluster
func namespaceExists(namespace string) bool {
	cmd := exec.Command("kubectl", "get", "namespace", namespace)
	err := cmd.Run()
	return err == nil
}

// deploymentExists checks if a deployment exists in the specified namespace
func deploymentExists(namespace, deploymentName string) bool {
	cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", namespace)
	err := cmd.Run()
	return err == nil
}

// clusterExists checks if the kind cluster exists
func clusterExists(clusterName string) bool {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, cluster := range clusters {
		if strings.TrimSpace(cluster) == clusterName {
			return true
		}
	}
	return false
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting kibaship-operator integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	var err error
	var cmd *exec.Cmd

	// Check if cluster exists, create if not
	clusterName := os.Getenv("KIND_CLUSTER")
	if clusterName == "" {
		clusterName = "kibaship-operator"
	}

	if !clusterExists(clusterName) {
		By("creating kind cluster (cluster does not exist)")
		cmd = exec.Command("make", "kind-create")
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to create kind cluster")
	} else {
		By("using existing kind cluster: " + clusterName)
	}

	// Check if infrastructure is already installed by checking for tekton-pipelines namespace
	infrastructureInstalled := namespaceExists("tekton-pipelines")

	if !infrastructureInstalled {
		By("installing Gateway API CRDs")
		Expect(utils.InstallGatewayAPI()).To(Succeed(), "Failed to install Gateway API CRDs")

		By("installing Cilium (CNI) via Helm")
		Expect(utils.InstallCiliumHelm("1.18.0")).To(Succeed(), "Failed to install Cilium via Helm")

		By("configuring CoreDNS to use public resolvers for e2e")
		Expect(utils.ConfigureCoreDNSForwarders()).To(Succeed(), "Failed to configure CoreDNS forwarders")

		By("installing CertManager")
		Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")

		By("installing Prometheus Operator")
		Expect(utils.InstallPrometheusOperator()).To(Succeed(), "Failed to install Prometheus Operator")

		By("installing Tekton Pipelines")
		Expect(utils.InstallTektonPipelines()).To(Succeed(), "Failed to install Tekton Pipelines")

		By("installing shared BuildKit daemon for cluster-wide builds")
		Expect(utils.InstallBuildkitSharedDaemon()).To(Succeed(), "Failed to install BuildKit daemon")

		By("installing Valkey Operator")
		Expect(utils.InstallValkeyOperator()).To(Succeed(), "Failed to install Valkey Operator")

		By("installing MySQL Operator")
		Expect(utils.InstallMySQLOperator()).To(Succeed(), "Failed to install MySQL Operator")
	} else {
		By("skipping infrastructure installation (tekton-pipelines namespace exists)")
	}

	// Always create storage classes (needed for every test run)
	By("creating storage classes for test environment")
	Expect(utils.CreateStorageClasses()).To(Succeed(), "Failed to create storage classes")

	// Always build and load images (these need to be refreshed on every test run)
	By("building the manager(Operator) image")
	cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image")

	By("loading the manager(Operator) image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the manager(Operator) image into Kind")

	By("building the railpack-cli image")
	cmd = exec.Command("docker", "build", "-f", "build/railpack-cli/Dockerfile", "-t", projectImageRailpackCLI, "build/railpack-cli")
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the railpack-cli image")

	By("loading the railpack-cli image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImageRailpackCLI)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the railpack-cli image into Kind")

	By("building the railpack-build (buildctl client) image")
	cmd = exec.Command("docker", "build", "-f", "build/railpack-build/Dockerfile", "-t", projectImageRailpackBld, "build/railpack-build")
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the railpack-build image")

	By("loading the railpack-build image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImageRailpackBld)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the railpack-build image into Kind")

	By("applying Tekton custom tasks (git-clone, railpack-prepare, railpack-build) with local image overrides")
	Expect(os.Setenv("RAILPACK_CLI_IMG", projectImageRailpackCLI)).To(Succeed(), "failed to set RAILPACK_CLI_IMG env")
	Expect(os.Setenv("RAILPACK_BUILD_IMG", projectImageRailpackBld)).To(Succeed(), "failed to set RAILPACK_BUILD_IMG env")
	Expect(utils.ApplyTektonResources()).To(Succeed(), "Failed to apply Tekton custom tasks")

	By("building the registry-auth image")
	cmd = exec.Command("make", "docker-build-registry-auth", fmt.Sprintf("IMG_REGISTRY_AUTH=%s", projectImageRegistryAuth))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the registry-auth image")

	By("loading the registry-auth image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImageRegistryAuth)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the registry-auth image into Kind")

	// Check if operator and registry are already deployed
	operatorDeployed := namespaceExists("kibaship-operator")
	registryDeployed := deploymentExists("registry", "registry") && deploymentExists("registry", "registry-auth")

	if !operatorDeployed {
		By("deploying test webhook receiver in-cluster")
		Expect(utils.DeployWebhookReceiver()).To(Succeed(), "Failed to deploy test webhook receiver")

		By("setting WEBHOOK_TARGET_URL for operator deploy")
		target := "http://webhook-receiver.kibaship-operator.svc.cluster.local:8080/webhook"
		err = os.Setenv("WEBHOOK_TARGET_URL", target)
		Expect(err).NotTo(HaveOccurred(), "failed to set WEBHOOK_TARGET_URL")

		By("provisioning kibaship-operator")
		Expect(utils.ProvisionKibashipOperator()).To(Succeed(), "Failed to provision kibaship-operator")

		By("waiting for kibaship-operator to be ready")
		Expect(utils.WaitForKibashipOperator()).To(Succeed(), "Failed to wait for kibaship-operator")
	} else {
		By("operator namespace exists - restarting deployments to pick up new images")
		cmd = exec.Command("kubectl", "rollout", "restart", "deployment", "-n", "kibaship-operator")
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to restart operator deployments")

		By("waiting for operator deployments to complete rollout")
		cmd = exec.Command("kubectl", "rollout", "status", "deployment", "-n", "kibaship-operator", "--timeout=5m")
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to wait for operator rollout")
	}

	if !registryDeployed {
		By("provisioning Docker Registry v3.0.0")
		Expect(utils.ProvisionRegistry()).To(Succeed(), "Failed to provision Docker Registry")

		By("provisioning Registry Auth service")
		Expect(utils.ProvisionRegistryAuth()).To(Succeed(), "Failed to provision Registry Auth service")

		By("waiting for Docker Registry to be ready")
		Expect(utils.WaitForRegistry()).To(Succeed(), "Failed to wait for Docker Registry")

		By("waiting for Registry Auth service to be ready")
		Expect(utils.WaitForRegistryAuth()).To(Succeed(), "Failed to wait for Registry Auth service")

		By("verifying registry-auth pods are healthy")
		Expect(utils.VerifyRegistryAuthHealthy()).To(Succeed(), "registry-auth pods are not healthy")

		By("verifying registry pods are healthy")
		Expect(utils.VerifyRegistryHealthy()).To(Succeed(), "registry pods are not healthy")
	} else {
		By("registry namespace exists - restarting deployments to pick up new images")
		cmd = exec.Command("kubectl", "rollout", "restart", "deployment", "-n", "registry")
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to restart registry deployments")

		By("waiting for registry deployments to complete rollout")
		cmd = exec.Command("kubectl", "rollout", "status", "deployment", "-n", "registry", "--timeout=5m")
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to wait for registry rollout")
	}

	// Fix OrbStack DNS issue: add registry services to Kind node /etc/hosts
	// OrbStack's DNS (0.250.250.254) cannot resolve cluster-internal service names,
	// so containerd fails to pull images from the cluster registry.
	By("configuring Kind node /etc/hosts for registry DNS resolution (OrbStack workaround)")
	cmd = exec.Command("kubectl", "get", "svc", "-n", "registry", "registry", "-o", "jsonpath={.spec.clusterIP}")
	registryIP, err := cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to get registry service IP")

	cmd = exec.Command("kubectl", "get", "svc", "-n", "registry", "registry-auth", "-o", "jsonpath={.spec.clusterIP}")
	registryAuthIP, err := cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to get registry-auth service IP")

	clusterName = os.Getenv("KIND_CLUSTER")
	if clusterName == "" {
		clusterName = "kibaship-operator"
	}

	By("adding registry service DNS entries to Kind node /etc/hosts")
	cmd = exec.Command("docker", "exec", fmt.Sprintf("%s-control-plane", clusterName), "bash", "-c",
		fmt.Sprintf("grep -q 'registry.registry.svc.cluster.local' /etc/hosts || echo '%s registry.registry.svc.cluster.local' >> /etc/hosts", strings.TrimSpace(string(registryIP))))
	_, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to add registry DNS entry to Kind node")

	cmd = exec.Command("docker", "exec", fmt.Sprintf("%s-control-plane", clusterName), "bash", "-c",
		fmt.Sprintf("grep -q 'registry-auth.registry.svc.cluster.local' /etc/hosts || echo '%s registry-auth.registry.svc.cluster.local' >> /etc/hosts", strings.TrimSpace(string(registryAuthIP))))
	_, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to add registry-auth DNS entry to Kind node")

	By("installing registry CA certificate on Kind node")
	cmd = exec.Command("kubectl", "get", "secret", "-n", "registry", "registry-tls", "-o", "jsonpath={.data.ca\\.crt}")
	caCertB64, err := cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to get registry CA certificate")

	cmd = exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | base64 -d | docker exec -i %s-control-plane tee /usr/local/share/ca-certificates/registry-ca.crt", strings.TrimSpace(string(caCertB64)), clusterName))
	_, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to copy CA certificate to Kind node")

	By("updating CA certificates on Kind node")
	cmd = exec.Command("docker", "exec", fmt.Sprintf("%s-control-plane", clusterName), "update-ca-certificates")
	_, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to update CA certificates on Kind node")

	By("restarting containerd to pick up DNS and TLS fixes")
	cmd = exec.Command("docker", "exec", fmt.Sprintf("%s-control-plane", clusterName), "systemctl", "restart", "containerd")
	_, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to restart containerd on Kind node")

	By("re-applying Tekton custom tasks with local railpack image override (always applies)")
	Expect(utils.ApplyTektonResources()).To(Succeed(), "Failed to re-apply Tekton custom tasks after operator deploy")

	By("building the API server image")
	cmd = exec.Command("make", "build-apiserver", fmt.Sprintf("IMG_APISERVER=%s", projectImageAPIServer))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the API server image")

	By("loading the API server image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImageAPIServer)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the API server image into Kind")

	By("deploying API server into operator namespace (always redeploys)")
	Expect(utils.DeployAPIServer(projectImageAPIServer)).To(Succeed(), "Failed to deploy API server")

	By("building the cert-manager webhook image")
	cmd = exec.Command("make", "docker-build-cert-manager-webhook", fmt.Sprintf("IMG_CERT_MANAGER_WEBHOOK=%s", projectImageCertWebhook))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the cert-manager webhook image")

	By("loading the cert-manager webhook image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImageCertWebhook)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the cert-manager webhook image into Kind")

	By("deploying cert-manager webhook into operator namespace (always redeploys)")
	Expect(utils.DeployCertManagerWebhook(projectImageCertWebhook)).To(Succeed(), "Failed to deploy cert-manager webhook")
})

var _ = AfterSuite(func() {
	_, _ = fmt.Fprintf(GinkgoWriter, "Tests completed. Cluster will remain for debugging.\n")
})

// Helper function to check if all required labels are present
func hasRequiredLabels(actual, required map[string]string) bool {
	for key, value := range required {
		if actual[key] != value {
			return false
		}
	}
	return true
}
