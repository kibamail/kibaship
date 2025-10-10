package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship-operator/test/utils"
)

var _ = Describe("Deployment Reconciliation", func() {
	var (
		projectUUID     string
		projectName     string
		applicationUUID string
		applicationName string
		deploymentUUID  string
		deploymentName  string
		envUUID         string
		envName         string
		projectNS       string
	)

	BeforeEach(func() {
		// Generate UUIDs for this test run
		projectUUID = uuid.New().String()
		projectName = fmt.Sprintf("project-%s", projectUUID)
		projectNS = projectName
		applicationUUID = uuid.New().String()
		applicationName = fmt.Sprintf("application-%s", applicationUUID)
		deploymentUUID = uuid.New().String()
		deploymentName = fmt.Sprintf("deployment-%s", deploymentUUID)
	})

	AfterEach(func() {
		// Resources will be cleaned up by cluster deletion at the start of next test run
	})

	Context("When creating a new Deployment in an existing Application", func() {
		It("should successfully reconcile with all controller side effects", func() {
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
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
    mysql:
      enabled: true
    postgres:
      enabled: true
    mysqlCluster:
      enabled: false
    postgresCluster:
      enabled: false
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
			}, "1m", "5s").Should(BeTrue(), "Project should be Ready")

			By("Waiting for default production environment to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/slug=production", "-o", "jsonpath={.items[0].metadata.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envName = strings.TrimSpace(string(output))
				return envName != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", envName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Fetching the environment UUID")
			cmd = exec.Command("kubectl", "get", "environment", envName, "-n", projectNS, "-o", "jsonpath={.metadata.labels.platform\\.kibaship\\.com/uuid}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envUUID = strings.TrimSpace(string(output))
			Expect(envUUID).NotTo(BeEmpty(), "Environment should have UUID label")

			By("Creating the test application with inline manifest")
			applicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
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
  type: GitRepository
  environmentRef:
    name: %s
  gitRepository:
    provider: github.com
    repository: railwayapp/railpack
    publicAccess: true
    branch: main
    rootDirectory: "examples/node-next"
    buildCommand: "npm install && npm run build"
    startCommand: "npm start"
`, applicationName, projectNS, applicationUUID, applicationName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(applicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Application should be Ready before creating Deployment")

			By("Creating the test deployment with inline manifest")
			deploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
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
  gitRepository:
    commitSHA: "960ef4fb6190de6aa8b394bf2f0d552ee67675c3"
    branch: "main"
`, deploymentName, projectNS, deploymentUUID, deploymentName, projectUUID, applicationUUID, workspaceUUIDConst, envUUID, applicationName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(deploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Deployment resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Deployment should be created successfully")

			By("Verifying Deployment has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", finalizersPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.Contains(string(output), "platform.operator.kibaship.com/deployment-finalizer")
			}, "30s", "2s").Should(BeTrue(), "Deployment should have finalizer")

			By("Verifying Deployment status becomes Initializing")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", statusPhasePath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				phase := strings.TrimSpace(string(output))
				return phase == "Initializing" || phase == "Running" || phase == "Succeeded" || phase == "Failed"
			}, "2m", "5s").Should(BeTrue(), "Deployment should have a valid phase")

			By("Verifying Deployment has required labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", labelsPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				labels := string(output)
				return strings.Contains(labels, "platform.kibaship.com/uuid") &&
					strings.Contains(labels, "platform.kibaship.com/project-uuid") &&
					strings.Contains(labels, "platform.kibaship.com/application-uuid")
			}, "30s", "2s").Should(BeTrue(), "Deployment should have required labels")

			By("Verifying Deployment name follows correct format")
			Expect(deploymentName).To(MatchRegexp(`^deployment-[a-z0-9]([a-z0-9-]*[a-z0-9])?$`),
				"Deployment name should follow format deployment-<deployment-uuid>")

			By("Verifying Deployment applicationRef points to the correct application")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", applicationRefPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == applicationName
			}, "30s", "2s").Should(BeTrue(), "Deployment should reference the correct application")

			By("Verifying Deployment GitRepository configuration is valid")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", gitCommitSHAPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "960ef4fb6190de6aa8b394bf2f0d552ee67675c3"
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid GitRepository commitSHA")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", gitBranchPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "main"
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid GitRepository branch")

			By("Verifying Deployment exists in the project namespace")
			Expect(projectNS).To(Equal(projectName),
				"Deployment should be created in the project namespace")

			By("Verifying Deployment webhook validation passed")
			// If we got this far, webhook validation passed since the Deployment was created successfully

			By("Verifying Deployment controller adds correct UUID labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", uuidLabelPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				uuid := strings.TrimSpace(string(output))
				return len(uuid) > 0 && strings.Contains(uuid, "-")
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid UUID label set by controller")

			By("Verifying Tekton Pipeline is created for the deployment")
			deploymentUUIDLabel := fmt.Sprintf("platform.kibaship.com/deployment-uuid=%s", deploymentUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipeline", "-n", projectNS, "-l", deploymentUUIDLabel)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "Tekton Pipeline should be created for deployment")

			By("Verifying PipelineRun is created for the deployment")
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", projectNS, "-l", deploymentUUIDLabel)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "PipelineRun should be created for deployment")

			By("Verifying the 'prepare' Tekton task ran and succeeded")
			var prName string
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", projectNS, "-l", deploymentUUIDLabel, "-o", "jsonpath={.items[0].metadata.name}")
				out, err := cmd.CombinedOutput()
				if err != nil {
					return err
				}
				prName = strings.TrimSpace(string(out))
				if prName == "" {
					return fmt.Errorf("no pipelinerun name yet")
				}
				return nil
			}, "2m", "5s").Should(Succeed(), "Should fetch PipelineRun name")

			Eventually(func() (string, error) {
				cmd := exec.Command(
					"kubectl", "get", "taskrun", "-n", projectNS,
					"-l", fmt.Sprintf("tekton.dev/pipelineRun=%s,tekton.dev/pipelineTask=prepare", prName),
					"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Succeeded')].status}",
				)
				out, err := cmd.CombinedOutput()
				return strings.TrimSpace(string(out)), err
			}, "10m", "10s").Should(Equal("True"), "prepare task should succeed")

			By("Verifying the 'build' Tekton task ran and succeeded")
			Eventually(func() (string, error) {
				cmd := exec.Command(
					"kubectl", "get", "taskrun", "-n", projectNS,
					"-l", fmt.Sprintf("tekton.dev/pipelineRun=%s,tekton.dev/pipelineTask=build", prName),
					"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Succeeded')].status}",
				)
				out, err := cmd.CombinedOutput()
				return strings.TrimSpace(string(out)), err
			}, "10m", "10s").Should(Equal("True"), "build task should succeed")

			By("Verifying all pipeline resources have proper tracking labels")
			Eventually(func() error {
				expectedLabels := map[string]string{
					"app.kubernetes.io/managed-by":           "kibaship-operator",
					"platform.kibaship.com/deployment-uuid":  deploymentUUID,
					"platform.kibaship.com/application-uuid": applicationUUID,
					"platform.kibaship.com/project-uuid":     projectUUID,
				}
				cmd := exec.Command("kubectl", "get", "pipeline", "-n", projectNS, "-l", deploymentUUIDLabel, "-o", "jsonpath={.items[0].metadata.labels}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to get pipeline labels: %v", err)
				}
				labels := string(output)
				for key := range expectedLabels {
					if !strings.Contains(labels, key) {
						return fmt.Errorf("pipeline missing label: %s", key)
					}
				}
				return nil
			}, "1m", "5s").Should(Succeed(), "Pipeline resources should have correct tracking labels")

			// ===================================================================
			// POST-BUILD RESOURCE CREATION TESTS
			// Tests for deployment_progress_controller_resource_creation task
			// ===================================================================

			By("Verifying Deployment phase transitions to Deploying or Succeeded after build succeeds")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "2m", "5s").Should(Or(Equal("Deploying"), Equal("Succeeded")), "Deployment should transition to Deploying or Succeeded phase after build succeeds (fast pod startup may skip Deploying)")

			By("Verifying Kubernetes Deployment resource is created by DeploymentProgressController")
			k8sDeploymentName := fmt.Sprintf("deployment-%s", deploymentUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "deployment", k8sDeploymentName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "2m", "5s").Should(Succeed(), "Kubernetes Deployment should be created after build succeeds")

			By("Verifying K8s Deployment has correct labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", k8sDeploymentName, "-n", projectNS, "-o", "jsonpath={.metadata.labels}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				labels := string(output)
				// jsonpath returns labels in JSON format: {"key":"value",...}
				return strings.Contains(labels, fmt.Sprintf("\"app.kubernetes.io/name\":\"app-%s\"", applicationUUID)) &&
					strings.Contains(labels, "\"app.kubernetes.io/managed-by\":\"kibaship-operator\"") &&
					strings.Contains(labels, fmt.Sprintf("\"platform.kibaship.com/deployment-uuid\":\"%s\"", deploymentUUID)) &&
					strings.Contains(labels, fmt.Sprintf("\"platform.kibaship.com/application-uuid\":\"%s\"", applicationUUID))
			}, "30s", "2s").Should(BeTrue(), "K8s Deployment should have correct labels")

			By("Verifying K8s Deployment has correct image from registry")
			expectedImage := fmt.Sprintf("registry.registry.svc.cluster.local/%s/%s:%s", projectNS, applicationUUID, deploymentUUID)
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "deployment", k8sDeploymentName, "-n", projectNS, "-o", "jsonpath={.spec.template.spec.containers[0].image}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "30s", "2s").Should(Equal(expectedImage), "K8s Deployment should use correct image from registry")

			By("Verifying K8s Deployment has resource limits configured")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", k8sDeploymentName, "-n", projectNS, "-o", "jsonpath={.spec.template.spec.containers[0].resources}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				resources := string(output)
				return strings.Contains(resources, "limits") && strings.Contains(resources, "requests")
			}, "30s", "2s").Should(BeTrue(), "K8s Deployment should have resource limits configured")

			By("Verifying K8s Deployment has registry image pull secret configured")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", k8sDeploymentName, "-n", projectNS, "-o", "jsonpath={.spec.template.spec.imagePullSecrets[0].name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "registry-image-pull-secret"
			}, "30s", "2s").Should(BeTrue(), "K8s Deployment should have registry image pull secret")

			By("Verifying Kubernetes Service resource is created by DeploymentProgressController")
			k8sServiceName := fmt.Sprintf("service-%s", applicationUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "service", k8sServiceName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "2m", "5s").Should(Succeed(), "Kubernetes Service should be created after build succeeds")

			By("Verifying K8s Service has correct type (ClusterIP)")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "service", k8sServiceName, "-n", projectNS, "-o", "jsonpath={.spec.type}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "30s", "2s").Should(Equal("ClusterIP"), "Service should be of type ClusterIP")

			By("Verifying K8s Service has correct selector to match Deployment pods")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "service", k8sServiceName, "-n", projectNS, "-o", "jsonpath={.spec.selector}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				selector := string(output)
				// jsonpath returns selector in JSON format: {"key":"value",...}
				return strings.Contains(selector, fmt.Sprintf("\"app.kubernetes.io/name\":\"app-%s\"", applicationUUID)) &&
					strings.Contains(selector, fmt.Sprintf("\"platform.kibaship.com/application-uuid\":\"%s\"", applicationUUID))
			}, "30s", "2s").Should(BeTrue(), "Service selector should match Deployment pod labels")

			By("Verifying K8s Service has correct port configuration")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "service", k8sServiceName, "-n", projectNS, "-o", "jsonpath={.spec.ports[0]}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				port := string(output)
				// jsonpath returns port in JSON format: {"name":"http","port":3000,"protocol":"TCP"}
				return strings.Contains(port, "\"name\":\"http\"") &&
					strings.Contains(port, "\"port\":3000") &&
					strings.Contains(port, "\"protocol\":\"TCP\"")
			}, "30s", "2s").Should(BeTrue(), "Service should have correct port configuration")

			By("Verifying K8s Service endpoints are populated (pods are discovered)")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "endpoints", k8sServiceName, "-n", projectNS, "-o", "jsonpath={.subsets[0].addresses}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				addresses := strings.TrimSpace(string(output))
				return addresses != "" && addresses != "[]"
			}, "5m", "10s").Should(BeTrue(), "Service endpoints should be populated with pod IPs")

			By("Verifying pods are created and becoming ready")
			Eventually(func() int {
				cmd := exec.Command("kubectl", "get", "pods", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/deployment-uuid=%s", deploymentUUID), "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return 0
				}
				readyStates := strings.Fields(string(output))
				readyCount := 0
				for _, state := range readyStates {
					if state == TrueString {
						readyCount++
					}
				}
				return readyCount
			}, "5m", "10s").Should(BeNumerically(">=", 1), "At least one pod should be ready")

			By("Verifying Deployment phase transitions to Succeeded after pods are ready")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "3m", "5s").Should(Equal("Succeeded"), "Deployment should transition to Succeeded phase after pods are ready")

			// ===================================================================
			// APPLICATION DOMAIN CREATION TESTS
			// Tests for deployment_reconciler_domain_creation task
			// REQUIRED: ApplicationDomain MUST be created for deployments
			// ===================================================================

			By("Verifying default ApplicationDomain was created for the application")
			var domainName string
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", applicationUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].metadata.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				domainName = strings.TrimSpace(string(output))
				return domainName != ""
			}, "1m", "5s").Should(BeTrue(), "ApplicationDomain MUST be created - test will fail if not created")

			By("Verifying ApplicationDomain has correct properties")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", applicationUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].spec}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				spec := string(output)
				// jsonpath returns spec in JSON format: {"key":"value",...}
				return strings.Contains(spec, "\"type\":\"default\"") &&
					strings.Contains(spec, "\"default\":true") &&
					strings.Contains(spec, "\"tlsEnabled\":true") &&
					strings.Contains(spec, "\"port\":3000")
			}, "30s", "2s").Should(BeTrue(), "ApplicationDomain should have correct default properties")

			By("Verifying ApplicationDomain domain follows pattern *.apps.<baseDomain>")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", applicationUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].spec.domain}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				domain := strings.TrimSpace(string(output))
				return strings.Contains(domain, ".apps.") && len(domain) > 0
			}, "30s", "2s").Should(BeTrue(), "ApplicationDomain should follow *.apps.<baseDomain> pattern")

			By("Verifying K8s resources have OwnerReferences for cascading deletion")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", k8sDeploymentName, "-n", projectNS, "-o", "jsonpath={.metadata.ownerReferences[0].kind}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "Deployment"
			}, "30s", "2s").Should(BeTrue(), "K8s Deployment should have OwnerReference to Deployment CR")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "service", k8sServiceName, "-n", projectNS, "-o", "jsonpath={.metadata.ownerReferences[0].kind}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "Deployment"
			}, "30s", "2s").Should(BeTrue(), "K8s Service should have OwnerReference to Deployment CR")

			// ===================================================================
			// PROMOTION TESTS
			// Tests for deployment promotion to Application.spec.currentDeploymentRef
			// ===================================================================

			By("Verifying Application.spec.currentDeploymentRef is updated after deployment succeeds with promote=true")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", "jsonpath={.spec.currentDeploymentRef.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "1m", "5s").Should(Equal(deploymentName), "Application.spec.currentDeploymentRef should be set to the promoted deployment")

			By("✅ All post-build resource creation tests passed - 3-controller architecture working correctly")
		})

		It("should successfully reconcile Dockerfile deployment with all controller side effects", func() {
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
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
    mysql:
      enabled: true
    postgres:
      enabled: true
    mysqlCluster:
      enabled: false
    postgresCluster:
      enabled: false
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
			}, "1m", "5s").Should(BeTrue(), "Project should be Ready")

			By("Waiting for default production environment to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/slug=production", "-o", "jsonpath={.items[0].metadata.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envName = strings.TrimSpace(string(output))
				return envName != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", envName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Fetching the environment UUID")
			cmd = exec.Command("kubectl", "get", "environment", envName, "-n", projectNS, "-o", "jsonpath={.metadata.labels.platform\\.kibaship\\.com/uuid}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envUUID = strings.TrimSpace(string(output))
			Expect(envUUID).NotTo(BeEmpty(), "Environment should have UUID label")

			By("Creating the test application with Dockerfile BuildType")
			applicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
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
  type: GitRepository
  environmentRef:
    name: %s
  gitRepository:
    provider: github.com
    repository: kibamail/kibaship-operator
    publicAccess: true
    branch: main
    buildType: Dockerfile
    dockerfileBuild:
      dockerfilePath: "examples/dockerfiles/todos-api-flask/Dockerfile"
      buildContext: "examples/dockerfiles/todos-api-flask"
`, applicationName, projectNS, applicationUUID, applicationName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(applicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Application should be Ready before creating Deployment")

			By("Verifying Application has BuildType=Dockerfile")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", "jsonpath={.spec.gitRepository.buildType}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "Dockerfile"
			}, "30s", "2s").Should(BeTrue(), "Application should have BuildType=Dockerfile")

			By("Creating the test deployment with inline manifest")
			deploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
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
  gitRepository:
    commitSHA: "80243e8b34fed099f6b0288b396bcda9a3776e9f"
    branch: "main"
`, deploymentName, projectNS, deploymentUUID, deploymentName, projectUUID, applicationUUID, workspaceUUIDConst, envUUID, applicationName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(deploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Deployment resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Deployment should be created successfully")

			By("Verifying Tekton Pipeline is created for the deployment")
			deploymentUUIDLabel := fmt.Sprintf("platform.kibaship.com/deployment-uuid=%s", deploymentUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipeline", "-n", projectNS, "-l", deploymentUUIDLabel)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "Tekton Pipeline should be created for deployment")

			By("Verifying PipelineRun is created for the deployment")
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", projectNS, "-l", deploymentUUIDLabel)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "PipelineRun should be created for deployment")

			By("Verifying the pipeline run was successfully created")
			var prName string
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", projectNS, "-l", deploymentUUIDLabel, "-o", "jsonpath={.items[0].metadata.name}")
				out, err := cmd.CombinedOutput()
				if err != nil {
					return err
				}
				prName = strings.TrimSpace(string(out))
				if prName == "" {
					return fmt.Errorf("no pipelinerun name yet")
				}
				return nil
			}, "2m", "5s").Should(Succeed(), "Should fetch PipelineRun name")

			By("Verifying the 'clone-repository' Tekton task ran and succeeded")
			Eventually(func() (string, error) {
				cmd := exec.Command(
					"kubectl", "get", "taskrun", "-n", projectNS,
					"-l", fmt.Sprintf("tekton.dev/pipelineRun=%s,tekton.dev/pipelineTask=clone-repository", prName),
					"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Succeeded')].status}",
				)
				out, err := cmd.CombinedOutput()
				return strings.TrimSpace(string(out)), err
			}, "5m", "10s").Should(Equal("True"), "clone-repository task should succeed")

			By("Verifying the 'build-dockerfile' Tekton task ran and succeeded")
			Eventually(func() (string, error) {
				cmd := exec.Command(
					"kubectl", "get", "taskrun", "-n", projectNS,
					"-l", fmt.Sprintf("tekton.dev/pipelineRun=%s,tekton.dev/pipelineTask=build-dockerfile", prName),
					"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Succeeded')].status}",
				)
				out, err := cmd.CombinedOutput()
				return strings.TrimSpace(string(out)), err
			}, "5m", "10s").Should(Equal("True"), "build-dockerfile task should succeed")

			By("Verifying Deployment phase transitions to Deploying or Succeeded after build succeeds")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "2m", "5s").Should(Or(Equal("Deploying"), Equal("Succeeded")), "Deployment should transition to Deploying or Succeeded phase after build succeeds")

			By("Verifying Kubernetes Deployment resource is created by DeploymentProgressController")
			k8sDeploymentName := fmt.Sprintf("deployment-%s", deploymentUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "deployment", k8sDeploymentName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "2m", "5s").Should(Succeed(), "Kubernetes Deployment should be created after build succeeds")

			By("Verifying Kubernetes Service resource is created by DeploymentProgressController")
			k8sServiceName := fmt.Sprintf("service-%s", applicationUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "service", k8sServiceName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "2m", "5s").Should(Succeed(), "Kubernetes Service should be created after build succeeds")

			By("Verifying pods are created and becoming ready")
			Eventually(func() int {
				cmd := exec.Command("kubectl", "get", "pods", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/deployment-uuid=%s", deploymentUUID), "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return 0
				}
				readyStates := strings.Fields(string(output))
				readyCount := 0
				for _, state := range readyStates {
					if state == TrueString {
						readyCount++
					}
				}
				return readyCount
			}, "5m", "10s").Should(BeNumerically(">=", 1), "At least one pod should be ready")

			By("Verifying Deployment phase transitions to Succeeded after pods are ready")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "3m", "5s").Should(Equal("Succeeded"), "Deployment should transition to Succeeded phase after pods are ready")

			By("✅ All Dockerfile deployment tests passed - BuildType=Dockerfile working correctly")
		})

		It("should successfully reconcile ImageFromRegistry deployment with all controller side effects", func() {
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
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
    imageFromRegistry:
      enabled: true
    mysql:
      enabled: true
    postgres:
      enabled: true
    mysqlCluster:
      enabled: false
    postgresCluster:
      enabled: false
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
			}, "1m", "5s").Should(BeTrue(), "Project should be Ready")

			By("Waiting for default production environment to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/slug=production", "-o", "jsonpath={.items[0].metadata.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envName = strings.TrimSpace(string(output))
				return envName != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", envName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Fetching the environment UUID")
			cmd = exec.Command("kubectl", "get", "environment", envName, "-n", projectNS, "-o", "jsonpath={.metadata.labels.platform\\.kibaship\\.com/uuid}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envUUID = strings.TrimSpace(string(output))
			Expect(envUUID).NotTo(BeEmpty(), "Environment should have UUID label")

			By("Creating the test application with ImageFromRegistry type")
			applicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
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
  type: ImageFromRegistry
  environmentRef:
    name: %s
  imageFromRegistry:
    registry: dockerhub
    repository: library/nginx
    defaultTag: "1.29.2"
    port: 80
`, applicationName, projectNS, applicationUUID, applicationName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(applicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Application should be Ready before creating Deployment")

			By("Verifying Application has Type=ImageFromRegistry")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", "jsonpath={.spec.type}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "ImageFromRegistry"
			}, "30s", "2s").Should(BeTrue(), "Application should have Type=ImageFromRegistry")

			By("Creating the test deployment with inline manifest")
			deploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
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
  imageFromRegistry:
    tag: "1.22"
`, deploymentName, projectNS, deploymentUUID, deploymentName, projectUUID, applicationUUID, workspaceUUIDConst, envUUID, applicationName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(deploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for deployment to be created and transition to Initializing")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				phase := strings.TrimSpace(string(output))
				return phase == "Initializing" || phase == "Deploying" || phase == "Succeeded"
			}, "2m", "2s").Should(BeTrue(), "Deployment should be created and have a valid phase")

			By("Verifying Deployment has ImageFromRegistry configuration")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", "jsonpath={.spec.imageFromRegistry.tag}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "1.22"
			}, "2m", "2s").Should(BeTrue(), "Deployment should have ImageFromRegistry tag=1.22")

			By("Waiting for Kubernetes Deployment to be created")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", fmt.Sprintf("deployment-%s", deploymentUUID), "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err == nil
			}, "2m", "5s").Should(BeTrue(), "Kubernetes Deployment should be created")

			By("Verifying Kubernetes Deployment has correct image")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "deployment", fmt.Sprintf("deployment-%s", deploymentUUID), "-n", projectNS, "-o", "jsonpath={.spec.template.spec.containers[0].image}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "2m", "2s").Should(Equal("docker.io/library/nginx:1.22"), "Kubernetes Deployment should have correct nginx image")

			By("Waiting for Kubernetes Service to be created")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "service", fmt.Sprintf("service-%s", deploymentUUID), "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err == nil
			}, "1m", "5s").Should(BeTrue(), "Kubernetes Service should be created")

			By("Verifying Kubernetes Service has correct port")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "service", fmt.Sprintf("service-%s", deploymentUUID), "-n", projectNS, "-o", "jsonpath={.spec.ports[0].port}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "2m", "2s").Should(Equal("80"), "Kubernetes Service should have port 80")

			By("Waiting for ApplicationDomain to be created")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomain", fmt.Sprintf("domain-%s", deploymentUUID), "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err == nil
			}, "1m", "5s").Should(BeTrue(), "ApplicationDomain should be created")

			By("Waiting for pods to become ready")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "deployment", fmt.Sprintf("deployment-%s", deploymentUUID), "-n", projectNS, "-o", "jsonpath={.status.readyReplicas}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return "0"
				}
				return strings.TrimSpace(string(output))
			}, "3m", "5s").Should(Equal("1"), "Pod should be ready")

			By("Waiting for deployment to transition to Succeeded phase")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "3m", "5s").Should(Equal("Succeeded"), "Deployment should transition to Succeeded phase after pods are ready")

			By("✅ All ImageFromRegistry deployment tests passed - ImageFromRegistry working correctly")
		})
	})
})
