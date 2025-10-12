package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship/internal/controller"
	"github.com/kibamail/kibaship/test/utils"
)

var _ = Describe("Valkey Deployment Reconciliation", func() {
	var (
		projectUUID                 string
		projectName                 string
		valkeyAppUUID               string
		valkeyAppName               string
		valkeyClusterAppUUID        string
		valkeyClusterAppName        string
		valkeyDeploymentUUID        string
		valkeyDeploymentName        string
		valkeyClusterDeploymentUUID string
		valkeyClusterDeploymentName string
		envUUID                     string
		envName                     string
		projectNS                   string
	)

	BeforeEach(func() {
		// Generate UUIDs for this test run
		projectUUID = uuid.New().String()
		projectName = fmt.Sprintf("project-%s", projectUUID)
		projectNS = projectName
		valkeyAppUUID = uuid.New().String()
		valkeyAppName = fmt.Sprintf("application-%s", valkeyAppUUID)
		valkeyClusterAppUUID = uuid.New().String()
		valkeyClusterAppName = fmt.Sprintf("application-%s", valkeyClusterAppUUID)
		valkeyDeploymentUUID = uuid.New().String()
		valkeyDeploymentName = fmt.Sprintf("deployment-%s", valkeyDeploymentUUID)
		valkeyClusterDeploymentUUID = uuid.New().String()
		valkeyClusterDeploymentName = fmt.Sprintf("deployment-%s", valkeyClusterDeploymentUUID)
	})

	AfterEach(func() {
		// Resources will be cleaned up by cluster deletion at the start of next test run
	})

	Context("When creating Valkey applications with deployments", func() {
		It("should successfully reconcile single Valkey deployments", func() {
			By("Creating the test project first with inline manifest")
			projectManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
spec:
  applicationTypes:
    valkey:
      enabled: true
    valkeyCluster:
      enabled: true
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
`, projectName, projectUUID, projectName, workspaceUUIDConst)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(projectManifest)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Applications")

			By("Waiting for default production environment to be ready and fetching its UUID")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.labels.platform\\.kibaship\\.com/uuid}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envUUID = strings.TrimSpace(string(output))
				return envUUID != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment UUID")

			By("Verifying the production environment is ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Getting the environment name for application reference")
			cmd = exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.name}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envName = strings.TrimSpace(string(output))
			Expect(envName).NotTo(BeEmpty(), "Environment should have a name")

			// ===================================================================
			// VALKEY SINGLE INSTANCE TESTS
			// ===================================================================

			By("Creating the Valkey application with inline manifest")
			valkeyApplicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  type: Valkey
  environmentRef:
    name: %s
  valkey:
    version: "7.2"
    database: 0
`, valkeyAppName, projectNS, valkeyAppUUID, valkeyAppName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(valkeyApplicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for Valkey application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", valkeyAppName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Valkey Application should be Ready before creating Deployment")

			By("Verifying Valkey ApplicationDomain was created with correct pattern")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", valkeyAppUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].spec.domain}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				domain := strings.TrimSpace(string(output))
				return strings.Contains(domain, ".valkey.") && len(domain) > 0
			}, "1m", "5s").Should(BeTrue(), "Valkey ApplicationDomain should follow *.valkey.<baseDomain> pattern")

			By("Verifying Valkey ApplicationDomain has correct port")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", valkeyAppUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].spec.port}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				port := strings.TrimSpace(string(output))
				return port == "6379"
			}, "30s", "2s").Should(BeTrue(), "Valkey ApplicationDomain should have port 6379")

			By("Creating the Valkey deployment with inline manifest")
			valkeyDeploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/application-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  applicationRef:
    name: %s
  promote: true
`, valkeyDeploymentName, projectNS, valkeyDeploymentUUID, valkeyDeploymentName, projectUUID, valkeyAppUUID, workspaceUUIDConst, envUUID, valkeyAppName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(valkeyDeploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Valkey Deployment resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, valkeyDeploymentName, "-n", projectNS)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Valkey Deployment should be created successfully")

			By("Verifying Valkey credentials secret is created")
			valkeySecretName := fmt.Sprintf("valkey-%s", valkeyDeploymentUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "secret", valkeySecretName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "Valkey credentials secret should be created")

			By("Verifying Valkey credentials secret has password")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "secret", valkeySecretName, "-n", projectNS, "-o", "jsonpath={.data.password}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				password := strings.TrimSpace(string(output))
				return len(password) > 0
			}, "30s", "2s").Should(BeTrue(), "Valkey secret should contain password")

			By("Verifying Valkey instance resource is created")
			valkeyInstanceName := fmt.Sprintf("valkey-%s", valkeyDeploymentUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "valkey", valkeyInstanceName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "30s", "5s").Should(Succeed(), "Valkey instance should be created")

			By("Verifying Valkey instance has correct configuration")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "valkey", valkeyInstanceName, "-n", projectNS, "-o", "jsonpath={.spec}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				spec := string(output)
				return strings.Contains(spec, "\"nodes\":1") &&
					strings.Contains(spec, "\"replicas\":0") &&
					strings.Contains(spec, "\"anonymousAuth\":false") &&
					strings.Contains(spec, "\"prometheus\":false")
			}, "30s", "2s").Should(BeTrue(), "Valkey instance should have correct single-instance configuration")

			By("Verifying Valkey pods are created and becoming ready")
			Eventually(func() int {
				cmd := exec.Command("kubectl", "get", "pods", "-n", projectNS, "-l", "app.kubernetes.io/managed-by=kibaship", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return 0
				}
				readyStates := strings.Fields(string(output))
				readyCount := 0
				for _, state := range readyStates {
					if state == controller.TrueString {
						readyCount++
					}
				}
				return readyCount
			}, "3m", "10s").Should(BeNumerically(">=", 1), "At least one Valkey pod should be ready")

			By("Verifying Valkey Deployment phase transitions to Succeeded")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, valkeyDeploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "4m", "5s").Should(Equal("Succeeded"), "Valkey Deployment should transition to Succeeded phase after Valkey is ready")
		})

		It("should successfully reconcile ValkeyCluster deployments", func() {
			By("Creating the test project first with inline manifest")
			projectManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
spec:
  applicationTypes:
    valkey:
      enabled: true
    valkeyCluster:
      enabled: true
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
`, projectName, projectUUID, projectName, workspaceUUIDConst)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(projectManifest)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Applications")

			By("Waiting for default production environment to be ready and fetching its UUID")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.labels.platform\\.kibaship\\.com/uuid}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envUUID = strings.TrimSpace(string(output))
				return envUUID != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment UUID")

			By("Verifying the production environment is ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Getting the environment name for application reference")
			cmd = exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.name}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envName = strings.TrimSpace(string(output))
			Expect(envName).NotTo(BeEmpty(), "Environment should have a name")

			By("Creating the ValkeyCluster application with inline manifest")
			valkeyClusterApplicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  type: ValkeyCluster
  environmentRef:
    name: %s
  valkeyCluster:
    version: "7.2"
    replicas: 6
    database: 0
`, valkeyClusterAppName, projectNS, valkeyClusterAppUUID, valkeyClusterAppName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(valkeyClusterApplicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for ValkeyCluster application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", valkeyClusterAppName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "ValkeyCluster Application should be Ready before creating Deployment")

			By("Verifying ValkeyCluster ApplicationDomain was created with correct pattern")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", valkeyClusterAppUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].spec.domain}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				domain := strings.TrimSpace(string(output))
				return strings.Contains(domain, ".valkey.") && len(domain) > 0
			}, "1m", "5s").Should(BeTrue(), "ValkeyCluster ApplicationDomain should follow *.valkey.<baseDomain> pattern")

			By("Creating the ValkeyCluster deployment with inline manifest")
			valkeyClusterDeploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/application-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  applicationRef:
    name: %s
  promote: true
`, valkeyClusterDeploymentName, projectNS, valkeyClusterDeploymentUUID, valkeyClusterDeploymentName, projectUUID, valkeyClusterAppUUID, workspaceUUIDConst, envUUID, valkeyClusterAppName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(valkeyClusterDeploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ValkeyCluster Deployment resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, valkeyClusterDeploymentName, "-n", projectNS)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "ValkeyCluster Deployment should be created successfully")

			By("Verifying ValkeyCluster credentials secret is created")
			valkeyClusterSecretName := fmt.Sprintf("valkey-%s", valkeyClusterDeploymentUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "secret", valkeyClusterSecretName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "ValkeyCluster credentials secret should be created")

			By("Verifying ValkeyCluster instance resource is created")
			valkeyClusterInstanceName := fmt.Sprintf("valkey-cluster-%s", valkeyClusterDeploymentUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "valkey", valkeyClusterInstanceName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "ValkeyCluster instance should be created")

			By("Verifying ValkeyCluster instance has correct cluster configuration")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "valkey", valkeyClusterInstanceName, "-n", projectNS, "-o", "jsonpath={.spec}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				spec := string(output)
				return strings.Contains(spec, "\"nodes\":3") &&
					strings.Contains(spec, "\"replicas\":1") &&
					strings.Contains(spec, "\"anonymousAuth\":false") &&
					strings.Contains(spec, "\"prometheus\":false")
			}, "30s", "2s").Should(BeTrue(), "ValkeyCluster instance should have correct cluster configuration")

			By("Verifying ValkeyCluster pods are created and becoming ready")
			Eventually(func() int {
				cmd := exec.Command("kubectl", "get", "pods", "-n", projectNS, "-l", "app.kubernetes.io/managed-by=kibaship", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return 0
				}
				readyStates := strings.Fields(string(output))
				readyCount := 0
				for _, state := range readyStates {
					if state == controller.TrueString {
						readyCount++
					}
				}
				return readyCount
			}, "10m", "15s").Should(BeNumerically(">=", 3), "At least 3 ValkeyCluster pods should be ready")

			By("Verifying ValkeyCluster Deployment phase transitions to Succeeded")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, valkeyClusterDeploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "3m", "5s").Should(Equal("Succeeded"), "ValkeyCluster Deployment should transition to Succeeded phase after cluster is ready")
		})

		It("should successfully handle Valkey deployment updates", func() {
			By("Creating the test project first with inline manifest")
			projectManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
spec:
  applicationTypes:
    valkey:
      enabled: true
    valkeyCluster:
      enabled: true
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
`, projectName, projectUUID, projectName, workspaceUUIDConst)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(projectManifest)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Applications")

			By("Waiting for default production environment to be ready and fetching its UUID")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.labels.platform\\.kibaship\\.com/uuid}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envUUID = strings.TrimSpace(string(output))
				return envUUID != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment UUID")

			By("Verifying the production environment is ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Getting the environment name for application reference")
			cmd = exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.name}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envName = strings.TrimSpace(string(output))
			Expect(envName).NotTo(BeEmpty(), "Environment should have a name")

			By("Creating the initial Valkey application with version 7.2")
			valkeyAppName := fmt.Sprintf("application-%s", valkeyAppUUID)
			valkeyApplicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  type: Valkey
  environmentRef:
    name: %s
  valkey:
    version: "7.2"
    database: 0
`, valkeyAppName, projectNS, valkeyAppUUID, valkeyAppName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(valkeyApplicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for Valkey application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", valkeyAppName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Valkey Application should be Ready before creating Deployment")

			By("Creating the first Valkey deployment")
			valkeyDeploymentName := fmt.Sprintf("deployment-%s", valkeyDeploymentUUID)
			valkeyDeploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/application-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  applicationRef:
    name: %s
  promote: true
`, valkeyDeploymentName, projectNS, valkeyDeploymentUUID, valkeyDeploymentName, projectUUID, valkeyAppUUID, workspaceUUIDConst, envUUID, valkeyAppName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(valkeyDeploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for first deployment to succeed")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, valkeyDeploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "6m", "5s").Should(Equal("Succeeded"), "First Valkey Deployment should succeed")

			valkeyInstanceName := fmt.Sprintf("valkey-%s", valkeyDeploymentUUID)

			By("Testing Valkey deployment update by changing version")
			valkeyUpdateManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  type: Valkey
  environmentRef:
    name: %s
  valkey:
    version: "7.2"
    database: 0
`, valkeyAppName, projectNS, valkeyAppUUID, valkeyAppName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(valkeyUpdateManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a second Valkey deployment to trigger update")
			valkeyDeployment2UUID := uuid.New().String()
			valkeyDeployment2Name := fmt.Sprintf("deployment-%s", valkeyDeployment2UUID)
			valkeyDeployment2Manifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/application-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  applicationRef:
    name: %s
  promote: true
`, valkeyDeployment2Name, projectNS, valkeyDeployment2UUID, valkeyDeployment2Name, projectUUID, valkeyAppUUID, workspaceUUIDConst, envUUID, valkeyAppName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(valkeyDeployment2Manifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Valkey instance is updated with new version")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "valkey", valkeyInstanceName, "-n", projectNS, "-o", "jsonpath={.spec.image}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				image := strings.TrimSpace(string(output))
				// Image field should be set when version is specified
				return image != "" && strings.Contains(image, "valkey/valkey:7.2")
			}, "2m", "5s").Should(BeTrue(), "Valkey instance should be updated with new version")

			By("Verifying second deployment also reaches Succeeded phase")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, valkeyDeployment2Name, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "3m", "5s").Should(Equal("Succeeded"), "Second Valkey Deployment should also reach Succeeded phase")
		})
	})
})
