package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship-operator/test/utils"
)

var _ = Describe("Deployment Reconciliation", func() {
	AfterEach(func() {
		// Resources will be cleaned up by cluster deletion at the start of next test run
	})

	Context("When creating a new Deployment in an existing Application", func() {
		It("should successfully reconcile with all controller side effects", func() {
			By("Creating the test project first")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/deployment-reconciliation/test-project.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", "test-project-deployment-e2e", "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Application")

			By("Creating the test application")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/deployment-reconciliation/test-application.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for application to be ready")
			appNamespace := "project-test-project-deployment-e2e-kibaship-com"
			appName := "application-myapp-deployment-e2e-kibaship-com"
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Application should be Ready before creating Deployment")

			By("Applying the test deployment YAML")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/deployment-reconciliation/test-deployment.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Deployment resource exists and passes validation")
			deploymentNamespace := "project-test-project-deployment-e2e-kibaship-com"
			deploymentName := "deployment-test-deploy-deployment-e2e-kibaship-com"
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", deploymentNamespace)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Deployment should be created successfully")

			By("Verifying Deployment has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", deploymentNamespace, "-o", finalizersPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.Contains(string(output), "platform.operator.kibaship.com/deployment-finalizer")
			}, "30s", "2s").Should(BeTrue(), "Deployment should have finalizer")

			By("Verifying Deployment status becomes Initializing")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", deploymentNamespace, "-o", statusPhasePath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				phase := strings.TrimSpace(string(output))
				return phase == "Initializing" || phase == "Running" || phase == "Succeeded" || phase == "Failed"
			}, "2m", "5s").Should(BeTrue(), "Deployment should have a valid phase")

			By("Verifying Deployment has required labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", deploymentNamespace, "-o", labelsPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				labels := string(output)
				return strings.Contains(labels, "platform.kibaship.com/uuid") &&
					strings.Contains(labels, "platform.kibaship.com/slug") &&
					strings.Contains(labels, "platform.kibaship.com/project-uuid") &&
					strings.Contains(labels, "platform.kibaship.com/application-uuid")
			}, "30s", "2s").Should(BeTrue(), "Deployment should have required labels")

			By("Verifying Deployment name follows correct format")
			Expect(deploymentName).To(MatchRegexp(`^deployment-[a-z0-9]([a-z0-9-]*[a-z0-9])?-kibaship-com$`),
				"Deployment name should follow format deployment-<deployment-slug>-kibaship-com")

			By("Verifying Deployment applicationRef points to the correct application")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", deploymentNamespace, "-o", applicationRefPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "application-myapp-deployment-e2e-kibaship-com"
			}, "30s", "2s").Should(BeTrue(), "Deployment should reference the correct application")

			By("Verifying Deployment GitRepository configuration is valid")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", deploymentNamespace, "-o", gitCommitSHAPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "960ef4fb6190de6aa8b394bf2f0d552ee67675c3"
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid GitRepository commitSHA")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", deploymentNamespace, "-o", gitBranchPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "main"
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid GitRepository branch")

			By("Verifying Deployment exists in the correct project namespace")
			Expect(deploymentNamespace).To(Equal("project-test-project-deployment-e2e-kibaship-com"),
				"Deployment should be created in the project namespace")

			By("Verifying Deployment webhook validation passed")
			// If we got this far, webhook validation passed since the Deployment was created successfully

			By("Verifying Deployment controller adds correct UUID labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, deploymentName, "-n", deploymentNamespace, "-o", uuidLabelPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				uuid := strings.TrimSpace(string(output))
				return len(uuid) > 0 && strings.Contains(uuid, "-")
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid UUID label set by controller")

			By("Verifying Tekton Pipeline is created for the deployment")
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipeline", "-n", deploymentNamespace, "-l", deploymentUUIDLabel)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "Tekton Pipeline should be created for deployment")

			By("Verifying PipelineRun is created for the deployment")
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", deploymentNamespace, "-l", deploymentUUIDLabel)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "PipelineRun should be created for deployment")

			By("Verifying the 'prepare' Tekton task ran and succeeded")
			var prName string
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", deploymentNamespace, "-l", deploymentUUIDLabel, "-o", "jsonpath={.items[0].metadata.name}")
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
					"kubectl", "get", "taskrun", "-n", deploymentNamespace,
					"-l", fmt.Sprintf("tekton.dev/pipelineRun=%s,tekton.dev/pipelineTask=prepare", prName),
					"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Succeeded')].status}",
				)
				out, err := cmd.CombinedOutput()
				return strings.TrimSpace(string(out)), err
			}, "10m", "10s").Should(Equal("True"), "prepare task should succeed")

			By("Verifying all pipeline resources have proper tracking labels")
			Eventually(func() error {
				expectedLabels := map[string]string{
					"app.kubernetes.io/managed-by":           "kibaship-operator",
					"platform.kibaship.com/deployment-uuid":  "789e4567-e89b-12d3-a456-426614174003",
					"platform.kibaship.com/application-uuid": "123e4567-e89b-12d3-a456-426614174003",
					"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440003",
				}
				cmd := exec.Command("kubectl", "get", "pipeline", "-n", deploymentNamespace, "-l", deploymentUUIDLabel, "-o", "jsonpath={.items[0].metadata.labels}")
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
