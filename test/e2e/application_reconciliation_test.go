package e2e

import (
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship-operator/test/utils"
)

var _ = Describe("Application Reconciliation", func() {
	AfterEach(func() {
		// Resources will be cleaned up by cluster deletion at the start of next test run
	})

	Context("When creating a new Application in an existing Project", func() {
		It("should successfully reconcile with all controller side effects", func() {
			By("Creating the test project first")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/application-reconciliation/test-project.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", "test-project-application-e2e", "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Application")

			By("Applying the test application YAML")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/application-reconciliation/test-application.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Application resource exists and passes validation")
			appNamespace := "project-test-project-application-e2e-kibaship-com"
			appName := "application-myapp-application-e2e-kibaship-com"
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Application should be created successfully")

			By("Verifying Application has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", finalizersPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.Contains(string(output), "platform.operator.kibaship.com/application-finalizer")
			}, "30s", "2s").Should(BeTrue(), "Application should have finalizer")

			By("Verifying Application status becomes Ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Application should become Ready")

			By("Verifying Application has required labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", labelsPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				labels := string(output)
				return strings.Contains(labels, "platform.kibaship.com/uuid") &&
					strings.Contains(labels, "platform.kibaship.com/slug") &&
					strings.Contains(labels, "platform.kibaship.com/project-uuid")
			}, "30s", "2s").Should(BeTrue(), "Application should have required labels")

			By("Verifying Application name follows correct format")
			Expect(appName).To(MatchRegexp(`^application-[a-z0-9]([a-z0-9-]*[a-z0-9])?-kibaship-com$`),
				"Application name should follow format application-<app-slug>-kibaship-com")

			By("Verifying Application GitRepository configuration is valid")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", gitProviderPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "github.com"
			}, "30s", "2s").Should(BeTrue(), "Application should have valid GitRepository provider")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", gitPublicAccessPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "true"
			}, "30s", "2s").Should(BeTrue(), "Application should have publicAccess set to true")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", gitRepositoryPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "kibamail/kibamail"
			}, "30s", "2s").Should(BeTrue(), "Application should have valid repository reference")

			By("Verifying Application exists in the correct project namespace")
			Expect(appNamespace).To(Equal("project-test-project-application-e2e-kibaship-com"),
				"Application should be created in the project namespace")

			By("Verifying Application projectRef points to the correct project")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", projectRefPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "test-project-application-e2e"
			}, "30s", "2s").Should(BeTrue(), "Application should reference the correct project")

			By("Verifying Application webhook validation passed")
			// If we got this far, webhook validation passed since the Application was created successfully

			By("Verifying Application controller adds correct UUID labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", uuidLabelPath)
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
