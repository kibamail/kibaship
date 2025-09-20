package e2e

import (
	"context"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kibamail/kibaship-operator/test/utils"
)

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
		// Resources will be cleaned up by cluster deletion at the start of next test run
	})

	Context("When creating a new Project", func() {
		It("should successfully reconcile with all controller side effects", func() {
			By("Applying the test project YAML")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/project-reconciliation/test-project.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Project resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", "test-project-reconciliation-e2e")
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Project should be created successfully")

			By("Verifying Project has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", testProjectName, "-o", finalizersPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.Contains(string(output), "platform.kibaship.com/project-finalizer")
			}, "30s", "2s").Should(BeTrue(), "Project should have finalizer")

			By("Verifying Project status becomes Ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", testProjectName, "-o", statusPhasePath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should become Ready")

			By("Verifying Project namespace is created")
			expectedNamespace := "project-test-project-reconciliation-e2e-kibaship-com"
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
				HaveKeyWithValue("platform.kibaship.com/project-name", "test-project-reconciliation-e2e"),
				HaveKeyWithValue("platform.kibaship.com/uuid", "550e8400-e29b-41d4-a716-446655440001"),
				HaveKeyWithValue("platform.kibaship.com/workspace-uuid", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			), "Namespace should have correct labels")

			By("Verifying service account is created")
			serviceAccountName := "project-test-project-reconciliation-e2e-sa-kibaship-com"
			Eventually(func() error {
				_, err := k8sClient.CoreV1().ServiceAccounts(expectedNamespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
				return err
			}, "1m", "5s").Should(Succeed(), "Service account should be created")

			By("Verifying admin role is created with full permissions")
			roleName := "project-test-project-reconciliation-e2e-admin-role-kibaship-com"
			Eventually(func() []rbacv1.PolicyRule {
				role, err := k8sClient.RbacV1().Roles(expectedNamespace).Get(ctx, roleName, metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return role.Rules
			}, "1m", "5s").Should(ContainElement(rbacv1.PolicyRule{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}), "Role should have full permissions")

			By("Verifying role binding connects service account to role")
			roleBindingName := "project-test-project-reconciliation-e2e-admin-binding-kibaship-com"
			Eventually(func() bool {
				rb, err := k8sClient.RbacV1().RoleBindings(expectedNamespace).Get(ctx, roleBindingName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				// Check subject references service account
				hasCorrectSubject := false
				for _, subject := range rb.Subjects {
					if subject.Kind == "ServiceAccount" && subject.Name == serviceAccountName && subject.Namespace == expectedNamespace {
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
			tektonRoleBindingName := "project-test-project-reconciliation-e2e-tekton-tasks-reader-binding-kibaship-com"
			Eventually(func() bool {
				rb, err := k8sClient.RbacV1().RoleBindings("tekton-pipelines").Get(ctx, tektonRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				// Check subject references project service account
				hasCorrectSubject := false
				for _, subject := range rb.Subjects {
					if subject.Kind == "ServiceAccount" && subject.Name == serviceAccountName && subject.Namespace == expectedNamespace {
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
					"platform.kibaship.com/project-name": "test-project-reconciliation-e2e",
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
