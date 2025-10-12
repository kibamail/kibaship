package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kibamail/kibaship/test/utils"
)

var _ = Describe("Project Reconciliation", func() {
	var (
		k8sClient   kubernetes.Interface
		ctx         context.Context
		projectUUID string
		projectName string
	)

	BeforeEach(func() {
		ctx = context.Background()
		client, err := getKubernetesClient()
		Expect(err).NotTo(HaveOccurred())
		k8sClient = client

		// Generate UUID for this test run
		projectUUID = uuid.New().String()
		projectName = fmt.Sprintf("project-%s", projectUUID)
	})

	AfterEach(func() {
		// Resources will be cleaned up by cluster deletion at the start of next test run
	})

	Context("When creating a new Project", func() {
		It("should successfully reconcile with all controller side effects", func() {
			By("Applying the test project with inline manifest")
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

			By("Verifying Project resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "Project should be created successfully")

			By("Verifying Project has finalizer added")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", finalizersPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.Contains(string(output), "platform.kibaship.com/project-finalizer")
			}, "30s", "2s").Should(BeTrue(), "Project should have finalizer")

			By("Verifying Project status becomes Ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", statusPhasePath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should become Ready")

			By("Verifying Project namespace is created")
			expectedNamespace := projectName
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
				HaveKeyWithValue("app.kubernetes.io/managed-by", "kibaship"),
				HaveKeyWithValue("platform.kibaship.com/project-name", projectName),
				HaveKeyWithValue("platform.kibaship.com/uuid", projectUUID),
				HaveKeyWithValue("platform.kibaship.com/workspace-uuid", workspaceUUIDConst),
			), "Namespace should have correct labels")

			By("Verifying service account is created")
			serviceAccountName := fmt.Sprintf("%s-sa", projectName)
			Eventually(func() error {
				_, err := k8sClient.CoreV1().ServiceAccounts(expectedNamespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
				return err
			}, "1m", "5s").Should(Succeed(), "Service account should be created")

			By("Verifying admin role is created with full permissions")
			roleName := fmt.Sprintf("%s-admin-role", projectName)
			Eventually(func() []rbacv1.PolicyRule {
				role, err := k8sClient.RbacV1().Roles(expectedNamespace).Get(ctx, roleName, metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return role.Rules
			}, "1m", "5s").Should(ContainElement(rbacv1.PolicyRule{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}), "Role should have full permissions")

			By("Verifying role binding connects service account to role")
			roleBindingName := fmt.Sprintf("%s-admin-binding", projectName)
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
			tektonRoleBindingName := fmt.Sprintf("%s-tekton-tasks-reader-binding", projectName)
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
					"app.kubernetes.io/managed-by":       "kibaship",
					"platform.kibaship.com/project-name": projectName,
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
