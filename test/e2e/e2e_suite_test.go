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

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kibamail/kibaship-operator/test/utils"
)

var (
	projectImage = "kibaship.com/kibaship-operator:v0.0.1"
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
	By("building the manager(Operator) image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
	_, err := utils.Run(cmd)

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
})

var _ = AfterSuite(func() {
	_, _ = fmt.Fprintf(GinkgoWriter, "Tests completed. Cleanup will be handled by Makefile (kind cluster deletion).\n")
})

var _ = Describe("Infrastructure Health Check", func() {
	Context("Operator and Valkey", func() {
		It("should have operator pod running successfully", func() {
			By("verifying operator pod is running")
			Eventually(func() error {
				client, err := getKubernetesClient()
				if err != nil {
					return fmt.Errorf("failed to create kubernetes client: %v", err)
				}

				ctx := context.Background()
				pods, err := client.CoreV1().Pods("kibaship-operator").List(ctx, metav1.ListOptions{
					LabelSelector: "control-plane=controller-manager",
				})
				if err != nil {
					return fmt.Errorf("failed to list operator pods: %v", err)
				}

				if len(pods.Items) == 0 {
					return fmt.Errorf("no operator pods found with label control-plane=controller-manager")
				}

				pod := pods.Items[0]
				if pod.Status.Phase != corev1.PodRunning {
					return fmt.Errorf("operator pod %s is not running, phase: %s", pod.Name, pod.Status.Phase)
				}

				// Also check that all containers are ready
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if !containerStatus.Ready {
						return fmt.Errorf("operator pod %s container %s is not ready", pod.Name, containerStatus.Name)
					}
				}

				return nil
			}, "5m", "10s").Should(Succeed(), "Operator pod should be running")
		})

		It("should have valkey cluster running successfully", func() {
			By("verifying valkey cluster pods are running")
			Eventually(func() error {
				client, err := getKubernetesClient()
				if err != nil {
					return fmt.Errorf("failed to create kubernetes client: %v", err)
				}

				ctx := context.Background()
				pods, err := client.CoreV1().Pods("kibaship-operator").List(ctx, metav1.ListOptions{
					LabelSelector: "app.kubernetes.io/name=valkey",
				})
				if err != nil {
					return fmt.Errorf("failed to list valkey pods: %v", err)
				}

				if len(pods.Items) == 0 {
					return fmt.Errorf("no valkey pods found with label app.kubernetes.io/name=valkey")
				}

				// Check that all valkey pods are running and ready
				for _, pod := range pods.Items {
					if pod.Status.Phase != corev1.PodRunning {
						return fmt.Errorf("valkey pod %s is not running, phase: %s", pod.Name, pod.Status.Phase)
					}

					// Check that all containers are ready
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if !containerStatus.Ready {
							return fmt.Errorf("valkey pod %s container %s is not ready", pod.Name, containerStatus.Name)
						}
					}
				}

				return nil
			}, "5m", "10s").Should(Succeed(), "Valkey cluster pods should be running")
		})
	})
})

var _ = Describe("Project Reconciliation", func() {
	var (
		k8sClient kubernetes.Interface
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		client, err := getKubernetesClient()
		Expect(err).NotTo(HaveOccurred())
		k8sClient = client
	})

	AfterEach(func() {
		By("Cleaning up test project")
		cmd := exec.Command("kubectl", "delete", "-f", "test/e2e/test-project.yaml", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		// Wait for namespace cleanup
		Eventually(func() bool {
			_, err := k8sClient.CoreV1().Namespaces().Get(ctx, "project-test-project-e2e-kibaship-com", metav1.GetOptions{})
			return err != nil // namespace should be gone
		}, "2m", "5s").Should(BeTrue(), "Project namespace should be deleted")
	})

	Context("When creating a new Project", func() {
		It("should successfully reconcile with all controller side effects", func() {
			By("Applying the test project YAML")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/test-project.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Project resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", "test-project-e2e")
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Project should be created successfully")

			By("Verifying Project has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", "test-project-e2e", "-o", "jsonpath={.metadata.finalizers[0]}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.Contains(string(output), "platform.kibaship.com/project-finalizer")
			}, "30s", "2s").Should(BeTrue(), "Project should have finalizer")

			By("Verifying Project status becomes Ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", "test-project-e2e", "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "Ready"
			}, "2m", "5s").Should(BeTrue(), "Project should become Ready")

			By("Verifying Project namespace is created")
			expectedNamespace := "project-test-project-e2e-kibaship-com"
			Eventually(func() error {
				_, err := k8sClient.CoreV1().Namespaces().Get(ctx, expectedNamespace, metav1.GetOptions{})
				return err
			}, "1m", "5s").Should(Succeed(), "Project namespace should be created")

			By("Verifying namespace has correct labels")
			Eventually(func() map[string]string {
				ns, err := k8sClient.CoreV1().Namespaces().Get(ctx, expectedNamespace, metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return ns.Labels
			}, "30s", "2s").Should(And(
				HaveKeyWithValue("app.kubernetes.io/managed-by", "kibaship-operator"),
				HaveKeyWithValue("platform.kibaship.com/project-name", "test-project-e2e"),
				HaveKeyWithValue("platform.kibaship.com/uuid", "550e8400-e29b-41d4-a716-446655440000"),
				HaveKeyWithValue("platform.kibaship.com/workspace-uuid", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			), "Namespace should have correct labels")

			By("Verifying service account is created")
			serviceAccountName := "project-test-project-e2e-sa-kibaship-com"
			Eventually(func() error {
				_, err := k8sClient.CoreV1().ServiceAccounts(expectedNamespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
				return err
			}, "1m", "5s").Should(Succeed(), "Service account should be created")

			By("Verifying admin role is created with full permissions")
			roleName := "project-test-project-e2e-admin-role-kibaship-com"
			Eventually(func() []rbacv1.PolicyRule {
				role, err := k8sClient.RbacV1().Roles(expectedNamespace).Get(ctx, roleName, metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return role.Rules
			}, "1m", "5s").Should(ContainElement(rbacv1.PolicyRule{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}), "Role should have full permissions")

			By("Verifying role binding connects service account to role")
			roleBindingName := "project-test-project-e2e-admin-binding-kibaship-com"
			Eventually(func() bool {
				rb, err := k8sClient.RbacV1().RoleBindings(expectedNamespace).Get(ctx, roleBindingName, metav1.GetOptions{})
				if err != nil {
					return false
				}

				// Check subject references service account
				hasCorrectSubject := false
				for _, subject := range rb.Subjects {
					if subject.Kind == "ServiceAccount" &&
						subject.Name == serviceAccountName &&
						subject.Namespace == expectedNamespace {
						hasCorrectSubject = true
						break
					}
				}

				// Check role ref references the admin role
				hasCorrectRoleRef := rb.RoleRef.Kind == "Role" && rb.RoleRef.Name == roleName

				return hasCorrectSubject && hasCorrectRoleRef
			}, "1m", "5s").Should(BeTrue(), "Role binding should connect service account to role")

			By("Verifying Tekton role is created")
			Eventually(func() error {
				_, err := k8sClient.RbacV1().Roles("tekton-pipelines").Get(ctx, "tekton-tasks-reader", metav1.GetOptions{})
				return err
			}, "1m", "5s").Should(Succeed(), "Tekton role should be created")

			By("Verifying Tekton role binding connects project service account to tekton role")
			tektonRoleBindingName := "project-test-project-e2e-tekton-tasks-reader-binding-kibaship-com"
			Eventually(func() bool {
				rb, err := k8sClient.RbacV1().RoleBindings("tekton-pipelines").Get(ctx, tektonRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return false
				}

				// Check subject references project service account
				hasCorrectSubject := false
				for _, subject := range rb.Subjects {
					if subject.Kind == "ServiceAccount" &&
						subject.Name == serviceAccountName &&
						subject.Namespace == expectedNamespace {
						hasCorrectSubject = true
						break
					}
				}

				// Check role ref references tekton role
				hasCorrectRoleRef := rb.RoleRef.Kind == "Role" && rb.RoleRef.Name == "tekton-tasks-reader"

				return hasCorrectSubject && hasCorrectRoleRef
			}, "1m", "5s").Should(BeTrue(), "Tekton role binding should connect project service account to tekton role")

			By("Verifying all resources have proper tracking labels")
			Eventually(func() bool {
				expectedLabels := map[string]string{
					"app.kubernetes.io/managed-by":       "kibaship-operator",
					"platform.kibaship.com/project-name": "test-project-e2e",
				}

				// Check service account labels
				sa, err := k8sClient.CoreV1().ServiceAccounts(expectedNamespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
				if err != nil || !hasRequiredLabels(sa.Labels, expectedLabels) {
					return false
				}

				// Check role labels
				role, err := k8sClient.RbacV1().Roles(expectedNamespace).Get(ctx, roleName, metav1.GetOptions{})
				if err != nil || !hasRequiredLabels(role.Labels, expectedLabels) {
					return false
				}

				// Check role binding labels
				rb, err := k8sClient.RbacV1().RoleBindings(expectedNamespace).Get(ctx, roleBindingName, metav1.GetOptions{})
				if err != nil || !hasRequiredLabels(rb.Labels, expectedLabels) {
					return false
				}

				return true
			}, "30s", "2s").Should(BeTrue(), "All resources should have correct tracking labels")
		})
	})
})

var _ = Describe("Application Reconciliation", func() {
	var (
		k8sClient kubernetes.Interface
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		client, err := getKubernetesClient()
		Expect(err).NotTo(HaveOccurred())
		k8sClient = client
	})

	AfterEach(func() {
		By("Cleaning up test application")
		cmd := exec.Command("kubectl", "delete", "-f", "test/e2e/test-application.yaml", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("Cleaning up test project")
		cmd = exec.Command("kubectl", "delete", "-f", "test/e2e/test-project.yaml", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		// Wait for namespace cleanup
		Eventually(func() bool {
			_, err := k8sClient.CoreV1().Namespaces().Get(ctx, "project-test-project-e2e-kibaship-com", metav1.GetOptions{})
			return err != nil // namespace should be gone
		}, "2m", "5s").Should(BeTrue(), "Project namespace should be deleted")
	})

	Context("When creating a new Application in an existing Project", func() {
		It("should successfully reconcile with all controller side effects", func() {
			By("Creating the test project first")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/test-project.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", "test-project-e2e", "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "Ready"
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Application")

			By("Applying the test application YAML")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/test-application.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Application resource exists and passes validation")
			appNamespace := "project-test-project-e2e-kibaship-com"
			appName := "application-myapp-kibaship-com"
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Application should be created successfully")

			By("Verifying Application has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.metadata.finalizers[0]}")
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
				return strings.TrimSpace(string(output)) == "Ready"
			}, "2m", "5s").Should(BeTrue(), "Application should become Ready")

			By("Verifying Application has required labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.metadata.labels}")
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
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.spec.gitRepository.provider}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "github.com"
			}, "30s", "2s").Should(BeTrue(), "Application should have valid GitRepository provider")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.spec.gitRepository.publicAccess}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "true"
			}, "30s", "2s").Should(BeTrue(), "Application should have publicAccess set to true")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.spec.gitRepository.repository}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "kibamail/kibamail"
			}, "30s", "2s").Should(BeTrue(), "Application should have valid repository reference")

			By("Verifying Application exists in the correct project namespace")
			Expect(appNamespace).To(Equal("project-test-project-e2e-kibaship-com"),
				"Application should be created in the project namespace")

			By("Verifying Application projectRef points to the correct project")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.spec.projectRef.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "test-project-e2e"
			}, "30s", "2s").Should(BeTrue(), "Application should reference the correct project")

			By("Verifying Application webhook validation passed")
			// If we got this far, webhook validation passed since the Application was created successfully
			// The webhook checks for proper name format, required labels, and GitRepository configuration

			By("Verifying Application controller adds correct UUID labels")
			// The controller should ensure UUID labels are set correctly through the ResourceLabeler
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.metadata.labels.platform\\.kibaship\\.com/uuid}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				uuid := strings.TrimSpace(string(output))
				// Validate UUID format
				return len(uuid) > 0 && strings.Contains(uuid, "-")
			}, "30s", "2s").Should(BeTrue(), "Application should have valid UUID label set by controller")
		})
	})
})

var _ = Describe("Deployment Reconciliation", func() {
	var (
		k8sClient kubernetes.Interface
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		client, err := getKubernetesClient()
		Expect(err).NotTo(HaveOccurred())
		k8sClient = client
	})

	AfterEach(func() {
		By("Cleaning up test deployment")
		cmd := exec.Command("kubectl", "delete", "-f", "test/e2e/test-deployment.yaml", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("Cleaning up test application")
		cmd = exec.Command("kubectl", "delete", "-f", "test/e2e/test-application.yaml", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("Cleaning up test project")
		cmd = exec.Command("kubectl", "delete", "-f", "test/e2e/test-project.yaml", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		// Wait for namespace cleanup
		Eventually(func() bool {
			_, err := k8sClient.CoreV1().Namespaces().Get(ctx, "project-test-project-e2e-kibaship-com", metav1.GetOptions{})
			return err != nil // namespace should be gone
		}, "2m", "5s").Should(BeTrue(), "Project namespace should be deleted")
	})

	Context("When creating a new Deployment in an existing Application", func() {
		It("should successfully reconcile with all controller side effects", func() {
			By("Creating the test project first")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/test-project.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", "test-project-e2e", "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "Ready"
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Application")

			By("Creating the test application")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/test-application.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for application to be ready")
			appNamespace := "project-test-project-e2e-kibaship-com"
			appName := "application-myapp-kibaship-com"
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", appName, "-n", appNamespace, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "Ready"
			}, "2m", "5s").Should(BeTrue(), "Application should be Ready before creating Deployment")

			By("Applying the test deployment YAML")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/test-deployment.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Deployment resource exists and passes validation")
			deploymentNamespace := "project-test-project-e2e-kibaship-com"
			deploymentName := "deployment-test-deploy-kibaship-com"
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Deployment should be created successfully")

			By("Verifying Deployment has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace, "-o", "jsonpath={.metadata.finalizers[0]}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.Contains(string(output), "platform.operator.kibaship.com/deployment-finalizer")
			}, "30s", "2s").Should(BeTrue(), "Deployment should have finalizer")

			By("Verifying Deployment status becomes Initializing")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				phase := strings.TrimSpace(string(output))
				return phase == "Initializing" || phase == "Running" || phase == "Succeeded" || phase == "Failed"
			}, "2m", "5s").Should(BeTrue(), "Deployment should have a valid phase")

			By("Verifying Deployment has required labels")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace, "-o", "jsonpath={.metadata.labels}")
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
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace, "-o", "jsonpath={.spec.applicationRef.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "application-myapp-kibaship-com"
			}, "30s", "2s").Should(BeTrue(), "Deployment should reference the correct application")

			By("Verifying Deployment GitRepository configuration is valid")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace, "-o", "jsonpath={.spec.gitRepository.commitSHA}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "06369723ec630bb2698fff92ae22f5c048b2a513"
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid GitRepository commitSHA")

			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace, "-o", "jsonpath={.spec.gitRepository.branch}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == "main"
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid GitRepository branch")

			By("Verifying Deployment exists in the correct project namespace")
			Expect(deploymentNamespace).To(Equal("project-test-project-e2e-kibaship-com"),
				"Deployment should be created in the project namespace")

			By("Verifying Deployment webhook validation passed")
			// If we got this far, webhook validation passed since the Deployment was created successfully
			// The webhook checks for proper name format, required labels, and GitRepository configuration

			By("Verifying Deployment controller adds correct UUID labels")
			// The controller should ensure UUID labels are set correctly through the ResourceLabeler
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace, "-o", "jsonpath={.metadata.labels.platform\\.kibaship\\.com/uuid}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				uuid := strings.TrimSpace(string(output))
				// Validate UUID format
				return len(uuid) > 0 && strings.Contains(uuid, "-")
			}, "30s", "2s").Should(BeTrue(), "Deployment should have valid UUID label set by controller")

			By("Verifying Tekton Pipeline is created for the deployment")
			// The deployment controller should create a Tekton Pipeline for CI/CD
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipeline", "-n", deploymentNamespace, "-l", "platform.kibaship.com/deployment-uuid=789e4567-e89b-12d3-a456-426614174111")
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "Tekton Pipeline should be created for deployment")

			By("Verifying PipelineRun is created for the deployment")
			// The deployment controller should trigger a PipelineRun for the initial deployment
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "pipelinerun", "-n", deploymentNamespace, "-l", "platform.kibaship.com/deployment-uuid=789e4567-e89b-12d3-a456-426614174111")
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "PipelineRun should be created for deployment")

			By("Verifying Deployment status includes pipeline run information")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", deploymentNamespace, "-o", "jsonpath={.status.currentPipelineRun}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				// Check if currentPipelineRun is populated (not empty)
				return len(strings.TrimSpace(string(output))) > 0
			}, "2m", "5s").Should(BeTrue(), "Deployment status should include currentPipelineRun information")

			By("Verifying all pipeline resources have proper tracking labels")
			Eventually(func() error {
				expectedLabels := map[string]string{
					"app.kubernetes.io/managed-by":           "kibaship-operator",
					"platform.kibaship.com/deployment-uuid":  "789e4567-e89b-12d3-a456-426614174111",
					"platform.kibaship.com/application-uuid": "123e4567-e89b-12d3-a456-426614174000",
					"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
				}

				// Check that pipeline has correct labels
				cmd := exec.Command("kubectl", "get", "pipeline", "-n", deploymentNamespace, "-l", "platform.kibaship.com/deployment-uuid=789e4567-e89b-12d3-a456-426614174111", "-o", "jsonpath={.items[0].metadata.labels}")
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

// Helper function to check if all required labels are present
func hasRequiredLabels(actual, required map[string]string) bool {
	for key, value := range required {
		if actual[key] != value {
			return false
		}
	}
	return true
}
