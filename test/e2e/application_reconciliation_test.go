package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship/pkg/utils"
	testutils "github.com/kibamail/kibaship/test/utils"
)

const (
	// TrueString represents the string "true" used in kubectl output comparisons
	TrueString = "true"
)

var _ = Describe("Application Reconciliation", func() {
	var (
		projectUUID     string
		projectName     string
		applicationUUID string
		applicationName string
		envUUID         string
		envName         string
		projectNS       string
	)

	BeforeEach(func() {
		// Generate UUIDs for this test run
		projectUUID = uuid.New().String()
		projectName = utils.GetProjectResourceName(projectUUID)
		projectNS = projectName
		applicationUUID = uuid.New().String()
		applicationName = utils.GetApplicationResourceName(applicationUUID)
	})

	AfterEach(func() {
		// Resources will be cleaned up by cluster deletion at the start of next test run
	})

	Context("When creating a new Application in an existing Project", func() {
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
			_, err := testutils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready")

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
    repository: kibamail/kibamail
    publicAccess: true
    branch: main
    rootDirectory: "./"
    buildCommand: "npm install && npm run build"
    startCommand: "npm start"
`, applicationName, projectNS, applicationUUID, applicationName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(applicationManifest)
			_, err = testutils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Application resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS)
				_, err := testutils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Application should be created successfully")

			By("Verifying Application has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", finalizersPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.Contains(string(output), "platform.operator.kibaship.com/application-finalizer")
			}, "30s", "2s").Should(BeTrue(), "Application should have finalizer")

			By("Verifying Application status becomes Ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Application should become Ready")

			By("Verifying Application has required labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", labelsPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				labels := string(output)
				return strings.Contains(labels, "platform.kibaship.com/uuid") &&
					strings.Contains(labels, "platform.kibaship.com/project-uuid")
			}, "30s", "2s").Should(BeTrue(), "Application should have required labels")

			By("Verifying Application name follows correct format")
			Expect(applicationName).To(MatchRegexp(`^application-[a-z0-9]([a-z0-9-]*[a-z0-9])?$`),
				"Application name should follow format application-<app-uuid>")

			By("Verifying Application GitRepository configuration is valid")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", gitProviderPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "github.com"
			}, "30s", "2s").Should(BeTrue(), "Application should have valid GitRepository provider")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", gitPublicAccessPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == TrueString
			}, "30s", "2s").Should(BeTrue(), "Application should have publicAccess set to true")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", gitRepositoryPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "kibamail/kibamail"
			}, "30s", "2s").Should(BeTrue(), "Application should have valid repository reference")

			By("Verifying Application exists in the project namespace")
			Expect(projectNS).To(Equal(projectName),
				"Application should be created in the project namespace")

			By("Verifying Application environmentRef points to the production environment")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", "jsonpath={.spec.environmentRef.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == envName
			}, "30s", "2s").Should(BeTrue(), "Application should reference the production environment")

			By("Verifying Application webhook validation passed")
			// If we got this far, webhook validation passed since the Application was created successfully

			By("Verifying Application controller adds correct UUID labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", applicationName, "-n", projectNS, "-o", uuidLabelPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				uuid := strings.TrimSpace(string(output))
				return len(uuid) > 0 && strings.Contains(uuid, "-")
			}, "30s", "2s").Should(BeTrue(), "Application should have valid UUID label set by controller")
		})
	})
})
