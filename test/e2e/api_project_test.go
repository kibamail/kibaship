package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kibamail/kibaship-operator/test/utils"
)

var _ = Describe("API Server Project Creation", func() {
	It("creates a project via REST and operator reconciles it", func() {
		By("fetching API key from api-server secret")
		cmd := exec.Command("kubectl", "get", "secret", "api-server-api-key-kibaship-com", "-n", "kibaship-operator", "-o", "jsonpath={.data.api-key}")
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to get api key secret: %s", string(output)))
		encodedKey := strings.TrimSpace(string(output))
		decodedKeyBytes, err := base64.StdEncoding.DecodeString(encodedKey)
		Expect(err).NotTo(HaveOccurred(), "failed to base64 decode api key")
		apiKey := strings.TrimSpace(string(decodedKeyBytes))

		By("port-forwarding API service to localhost:18080")
		pfCmd := exec.Command("kubectl", "-n", "kibaship-operator", "port-forward", "svc/apiserver", "18080:80")
		Expect(pfCmd.Start()).To(Succeed(), "failed to start port-forward")
		defer func() { _ = pfCmd.Process.Kill() }()

		// Wait for readiness on /readyz
		Eventually(func() bool {
			resp, err := http.Get("http://127.0.0.1:18080/readyz")
			if err != nil {
				return false
			}
			io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, "60s", "1s").Should(BeTrue(), "API server did not become ready via /readyz")

		By("calling POST /projects to create a project")
		workspaceUUID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
		bodyMap := map[string]any{
			"name":          "test-project-api-e2e",
			"workspaceUuid": workspaceUUID,
		}
		reqBytes, _ := json.Marshal(bodyMap)
		httpReq, _ := http.NewRequest("POST", "http://127.0.0.1:18080/projects", bytes.NewReader(reqBytes))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		httpClient := &http.Client{Timeout: 30 * time.Second}
		httpResp, err := httpClient.Do(httpReq)
		Expect(err).NotTo(HaveOccurred(), "HTTP request to create project failed")
		defer httpResp.Body.Close()
		Expect(httpResp.StatusCode).To(Equal(http.StatusCreated))

		var respObj struct {
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(httpResp.Body).Decode(&respObj)
		Expect(respObj.Slug).NotTo(BeEmpty(), "API response should include slug")

		projectCRName := "project-" + respObj.Slug
		expectedNamespace := "project-" + projectCRName + "-kibaship-com"
		serviceAccountName := "project-" + projectCRName + "-sa-kibaship-com"
		roleName := "project-" + projectCRName + "-admin-role-kibaship-com"
		roleBindingName := "project-" + projectCRName + "-admin-binding-kibaship-com"
		tektonRoleBindingName := "project-" + projectCRName + "-tekton-tasks-reader-binding-kibaship-com"

		By("asserting Project CR is created")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "get", "project", projectCRName)
			_, err := utils.Run(cmd)
			return err == nil
		}, "30s", "2s").Should(BeTrue(), "Project CR should be created")

		By("asserting Project has finalizer")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "get", "project", projectCRName, "-o", finalizersPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return false
			}
			return strings.Contains(string(output), "platform.kibaship.com/project-finalizer")
		}, "1m", "2s").Should(BeTrue(), "Project should have finalizer")

		By("waiting for Project to become Ready")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "get", "project", projectCRName, "-o", statusPhasePath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return false
			}
			return strings.TrimSpace(string(output)) == readyPhase
		}, "3m", "5s").Should(BeTrue(), "Project should become Ready")

		// Use client-go for namespace and RBAC checks
		clientset, err := getKubernetesClient()
		Expect(err).NotTo(HaveOccurred())
		ctx := context.Background()

		By("verifying namespace is created")
		Eventually(func() error {
			_, err := clientset.CoreV1().Namespaces().Get(ctx, expectedNamespace, metav1.GetOptions{})
			return err
		}, "2m", "5s").Should(Succeed(), "Project namespace should be created")

		By("verifying namespace has required labels")
		Eventually(func() map[string]string {
			ns, err := clientset.CoreV1().Namespaces().Get(ctx, expectedNamespace, metav1.GetOptions{})
			if err != nil {
				return nil
			}
			return ns.Labels
		}, "30s", "2s").Should(And(
			HaveKeyWithValue("app.kubernetes.io/managed-by", "kibaship-operator"),
			HaveKeyWithValue("platform.kibaship.com/project-name", projectCRName),
			HaveKeyWithValue("platform.kibaship.com/workspace-uuid", workspaceUUID),
		), "Namespace should have correct labels")

		By("verifying service account is created")
		Eventually(func() error {
			_, err := clientset.CoreV1().ServiceAccounts(expectedNamespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
			return err
		}, "1m", "5s").Should(Succeed(), "Service account should be created")

		By("verifying admin role is created with full permissions")
		Eventually(func() []rbacv1.PolicyRule {
			role, err := clientset.RbacV1().Roles(expectedNamespace).Get(ctx, roleName, metav1.GetOptions{})
			if err != nil {
				return nil
			}
			return role.Rules
		}, "1m", "5s").Should(ContainElement(rbacv1.PolicyRule{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}))

		By("verifying role binding connects SA to role")
		Eventually(func() bool {
			rb, err := clientset.RbacV1().RoleBindings(expectedNamespace).Get(ctx, roleBindingName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			hasSubject := false
			for _, s := range rb.Subjects {
				if s.Kind == "ServiceAccount" && s.Name == serviceAccountName && s.Namespace == expectedNamespace {
					hasSubject = true
					break
				}
			}
			hasRoleRef := rb.RoleRef.Kind == "Role" && rb.RoleRef.Name == roleName
			return hasSubject && hasRoleRef
		}, "1m", "5s").Should(BeTrue(), "RoleBinding should connect SA to Role")

		By("verifying Tekton role binding exists in tekton-pipelines namespace")
		Eventually(func() error {
			_, err := clientset.RbacV1().RoleBindings("tekton-pipelines").Get(ctx, tektonRoleBindingName, metav1.GetOptions{})
			return err
		}, "1m", "5s").Should(Succeed(), "Tekton rolebinding should be created for project")
	})
})
