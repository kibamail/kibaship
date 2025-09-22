package e2e

import (
	"fmt"
	"os/exec"
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
)

var (
	projectImage            = "kibaship.com/kibaship-operator:v0.0.1"
	projectImageAPIServer   = "kibaship.com/kibaship-operator-apiserver:v0.0.1"
	projectImageCertWebhook = "kibaship.com/kibaship-operator-cert-manager-webhook:v0.0.1"
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

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting kibaship-operator integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	By("cleaning up any existing kind cluster")
	cmd := exec.Command("make", "kind-delete")
	_, _ = utils.Run(cmd) // Ignore errors if cluster doesn't exist

	By("creating fresh kind cluster")
	cmd = exec.Command("make", "kind-create")
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to create kind cluster")
	By("installing Gateway API CRDs")
	Expect(utils.InstallGatewayAPI()).To(Succeed(), "Failed to install Gateway API CRDs")

	By("installing Cilium (CNI) via Helm")
	Expect(utils.InstallCiliumHelm("1.18.0")).To(Succeed(), "Failed to install Cilium via Helm")

	By("building the manager(Operator) image")

	By("configuring CoreDNS to use public resolvers for e2e")
	Expect(utils.ConfigureCoreDNSForwarders()).To(Succeed(), "Failed to configure CoreDNS forwarders")

	cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image")

	By("loading the manager(Operator) image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the manager(Operator) image into Kind")

	By("installing CertManager")
	Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")

	By("installing Prometheus Operator")
	Expect(utils.InstallPrometheusOperator()).To(Succeed(), "Failed to install Prometheus Operator")

	By("installing Tekton Pipelines")
	Expect(utils.InstallTektonPipelines()).To(Succeed(), "Failed to install Tekton Pipelines")

	By("installing Valkey Operator")
	Expect(utils.InstallValkeyOperator()).To(Succeed(), "Failed to install Valkey Operator")

	By("creating storage-replica-1 storage class for test environment")
	Expect(utils.CreateStorageReplicaStorageClass()).To(Succeed(), "Failed to create storage-replica-1 storage class")

	By("deploying kibaship-operator")
	Expect(utils.DeployKibashipOperator()).To(Succeed(), "Failed to deploy kibaship-operator")

	By("building the API server image")
	cmd = exec.Command("make", "build-apiserver", fmt.Sprintf("IMG_APISERVER=%s", projectImageAPIServer))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the API server image")

	By("loading the API server image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImageAPIServer)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the API server image into Kind")

	By("deploying API server into operator namespace")
	Expect(utils.DeployAPIServer(projectImageAPIServer)).To(Succeed(), "Failed to deploy API server")

	By("building the cert-manager webhook image")
	cmd = exec.Command("make", "docker-build-cert-manager-webhook", fmt.Sprintf("IMG_CERT_MANAGER_WEBHOOK=%s", projectImageCertWebhook))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the cert-manager webhook image")

	By("loading the cert-manager webhook image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImageCertWebhook)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the cert-manager webhook image into Kind")

	By("deploying cert-manager webhook into operator namespace")
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
