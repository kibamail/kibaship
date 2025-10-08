package e2e

import (
	"bytes"
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
)

// Covers: GET/PATCH/DELETE /v1/projects/{uuid}
var _ = Describe("API Server Project CRUD", func() {
	It("performs GET, PATCH, and DELETE on a project and verifies operator cleanup", func() {
		By("fetching API key from api-server secret")
		cmd := exec.Command("kubectl", "get", "secret", "api-server-api-key", "-n", "kibaship-operator", "-o", "jsonpath={.data.api-key}")
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
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, "60s", "1s").Should(BeTrue(), "API server did not become ready via /readyz")

		httpClient := &http.Client{Timeout: 30 * time.Second}

		By("creating a project via POST /v1/projects")
		workspaceUUID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
		projReqBody := map[string]any{
			"name":          "proj-crud-e2e",
			"workspaceUuid": workspaceUUID,
		}
		projBytes, _ := json.Marshal(projReqBody)
		reqProj, _ := http.NewRequest("POST", "http://127.0.0.1:18080/v1/projects", bytes.NewReader(projBytes))
		reqProj.Header.Set("Content-Type", "application/json")
		reqProj.Header.Set("Authorization", "Bearer "+apiKey)
		respProj, err := httpClient.Do(reqProj)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respProj.Body.Close() }()
		Expect(respProj.StatusCode).To(Equal(http.StatusCreated))
		var projResp struct {
			UUID string `json:"uuid"`
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respProj.Body).Decode(&projResp)
		Expect(projResp.UUID).NotTo(BeEmpty())
		Expect(projResp.UUID).NotTo(BeEmpty())

		projectCRName := "project-" + projResp.UUID
		expectedNamespace := projectCRName
		serviceAccountName := projectCRName + "-sa"
		roleName := projectCRName + "-admin-role"
		roleBindingName := projectCRName + "-admin-binding"
		tektonRB := projectCRName + "-tekton-tasks-reader-binding"

		By("GET /v1/projects/{uuid} returns project details")
		Eventually(func() int {
			reqGet, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:18080/v1/projects/%s", projResp.UUID), nil)
			reqGet.Header.Set("Authorization", "Bearer "+apiKey)
			respGet, err := httpClient.Do(reqGet)
			if err != nil {
				return 0
			}
			_, _ = io.Copy(io.Discard, respGet.Body)
			_ = respGet.Body.Close()
			return respGet.StatusCode
		}, "60s", "1s").Should(Equal(http.StatusOK))

		By("PATCH /v1/projects/{uuid} updates description")
		newDesc := "updated from e2e"
		patchBody := map[string]any{"description": newDesc}
		patchBytes, _ := json.Marshal(patchBody)
		reqPatch, _ := http.NewRequest("PATCH", fmt.Sprintf("http://127.0.0.1:18080/v1/projects/%s", projResp.UUID), bytes.NewReader(patchBytes))
		reqPatch.Header.Set("Content-Type", "application/json")
		reqPatch.Header.Set("Authorization", "Bearer "+apiKey)
		respPatch, err := httpClient.Do(reqPatch)
		Expect(err).NotTo(HaveOccurred())
		bodyBytes, _ := io.ReadAll(respPatch.Body)
		_ = respPatch.Body.Close()
		Expect(respPatch.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("patch response: %s", string(bodyBytes)))
		var patched struct {
			Description string `json:"description"`
		}
		_ = json.Unmarshal(bodyBytes, &patched)
		Expect(patched.Description).To(Equal(newDesc))

		By("DELETE /v1/projects/{uuid} deletes the project")
		reqDel, _ := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:18080/v1/projects/%s", projResp.UUID), nil)
		reqDel.Header.Set("Authorization", "Bearer "+apiKey)
		respDel, err := httpClient.Do(reqDel)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respDel.Body.Close() }()
		Expect(respDel.StatusCode).To(Equal(http.StatusNoContent))

		By("verifying Project CR is gone")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "get", "project", projectCRName)
			_, err := cmd.CombinedOutput()
			return err != nil // not found
		}, "2m", "5s").Should(BeTrue())

		By("verifying namespace removed")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "get", "ns", expectedNamespace)
			_, err := cmd.CombinedOutput()
			return err != nil
		}, "2m", "5s").Should(BeTrue())

		By("verifying RBAC resources removed")
		Eventually(func() bool { // SA
			cmd := exec.Command("kubectl", "get", "sa", "-n", expectedNamespace, serviceAccountName)
			_, err := cmd.CombinedOutput()
			return err != nil
		}, "2m", "5s").Should(BeTrue())
		Eventually(func() bool { // Role
			cmd := exec.Command("kubectl", "get", "role", "-n", expectedNamespace, roleName)
			_, err := cmd.CombinedOutput()
			return err != nil
		}, "2m", "5s").Should(BeTrue())
		Eventually(func() bool { // RoleBinding in project ns
			cmd := exec.Command("kubectl", "get", "rolebinding", "-n", expectedNamespace, roleBindingName)
			_, err := cmd.CombinedOutput()
			return err != nil
		}, "2m", "5s").Should(BeTrue())
		Eventually(func() bool { // Tekton RoleBinding in tekton ns
			cmd := exec.Command("kubectl", "get", "rolebinding", "-n", "tekton-pipelines", tektonRB)
			_, err := cmd.CombinedOutput()
			return err != nil
		}, "2m", "5s").Should(BeTrue())
	})
})
