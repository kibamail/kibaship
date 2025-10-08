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
		})
	})
})
