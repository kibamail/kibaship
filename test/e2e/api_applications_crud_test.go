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

// Covers: GET/PATCH/DELETE /applications/{slug} and GET /projects/{slug}/applications
var _ = Describe("API Server Application CRUD", func() {
	It("creates, updates, lists and deletes an application; verifies domain cleanup", func() {
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
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, "60s", "1s").Should(BeTrue(), "API server did not become ready via /readyz")

		httpClient := &http.Client{Timeout: 30 * time.Second}

		By("creating a project via POST /projects")
		workspaceUUID := workspaceUUIDConst
		projReqBody := map[string]any{
			"name":          "proj-for-app-crud-e2e",
			"workspaceUuid": workspaceUUID,
		}
		projBytes, _ := json.Marshal(projReqBody)
		reqProj, _ := http.NewRequest("POST", "http://127.0.0.1:18080/projects", bytes.NewReader(projBytes))
		reqProj.Header.Set("Content-Type", "application/json")
		reqProj.Header.Set("Authorization", "Bearer "+apiKey)
		respProj, err := httpClient.Do(reqProj)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respProj.Body.Close() }()
		Expect(respProj.StatusCode).To(Equal(http.StatusCreated))
		var projResp struct {
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respProj.Body).Decode(&projResp)
		Expect(projResp.Slug).NotTo(BeEmpty())

		By("creating an environment via POST /projects/{slug}/environments")
		envReqBody := map[string]any{
			"name":        "production",
			"description": "Production environment",
		}
		envBytes, _ := json.Marshal(envReqBody)
		reqEnv, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:18080/projects/%s/environments", projResp.Slug), bytes.NewReader(envBytes))
		reqEnv.Header.Set("Content-Type", "application/json")
		reqEnv.Header.Set("Authorization", "Bearer "+apiKey)
		respEnv, err := httpClient.Do(reqEnv)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respEnv.Body.Close() }()
		if respEnv.StatusCode != http.StatusCreated {
			bodyBytes, _ := io.ReadAll(respEnv.Body)
			_, _ = fmt.Fprintf(GinkgoWriter, "Environment creation failed. Status: %d, Body: %s\n", respEnv.StatusCode, string(bodyBytes))
		}
		Expect(respEnv.StatusCode).To(Equal(http.StatusCreated))
		var envResp struct {
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respEnv.Body).Decode(&envResp)
		Expect(envResp.Slug).NotTo(BeEmpty())

		By("creating an application via POST /environments/{slug}/applications (GitRepository)")
		appReqBody := map[string]any{
			"name": "app-crud-e2e",
			"type": "GitRepository",
			"gitRepository": map[string]any{
				"provider":      "github.com",
				"repository":    "kibamail/kibamail",
				"publicAccess":  true,
				"branch":        "main",
				"rootDirectory": "./",
			},
		}
		appBytes, _ := json.Marshal(appReqBody)
		reqApp, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:18080/environments/%s/applications", envResp.Slug), bytes.NewReader(appBytes))
		reqApp.Header.Set("Content-Type", "application/json")
		reqApp.Header.Set("Authorization", "Bearer "+apiKey)
		respApp, err := httpClient.Do(reqApp)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respApp.Body.Close() }()
		Expect(respApp.StatusCode).To(Equal(http.StatusCreated))
		var appResp struct {
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respApp.Body).Decode(&appResp)
		Expect(appResp.Slug).NotTo(BeEmpty())

		appCRName := fmt.Sprintf("application-%s-kibaship-com", appResp.Slug)

		By("GET /applications/{slug} returns application details")
		Eventually(func() int {
			reqGet, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:18080/application/%s", appResp.Slug), nil)
			reqGet.Header.Set("Authorization", "Bearer "+apiKey)
			respGet, err := httpClient.Do(reqGet)
			if err != nil {
				return 0
			}
			_, _ = io.Copy(io.Discard, respGet.Body)
			_ = respGet.Body.Close()
			return respGet.StatusCode
		}, "60s", "1s").Should(Equal(http.StatusOK))

		By("PATCH /applications/{slug} updates name")
		newName := "app-crud-e2e-updated"
		patchBody := map[string]any{"name": newName}
		patchBytes, _ := json.Marshal(patchBody)
		reqPatch, _ := http.NewRequest("PATCH", fmt.Sprintf("http://127.0.0.1:18080/application/%s", appResp.Slug), bytes.NewReader(patchBytes))
		reqPatch.Header.Set("Content-Type", "application/json")
		reqPatch.Header.Set("Authorization", "Bearer "+apiKey)
		respPatch, err := httpClient.Do(reqPatch)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respPatch.Body.Close() }()
		Expect(respPatch.StatusCode).To(Equal(http.StatusOK))
		var patched struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(respPatch.Body).Decode(&patched)
		Expect(patched.Name).To(Equal(newName))

		By("GET /environments/{slug}/applications lists the application")
		Eventually(func() bool {
			reqList, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:18080/environments/%s/applications", envResp.Slug), nil)
			reqList.Header.Set("Authorization", "Bearer "+apiKey)
			respList, err := httpClient.Do(reqList)
			if err != nil {
				return false
			}
			defer func() { _ = respList.Body.Close() }()
			if respList.StatusCode != http.StatusOK {
				_, _ = io.Copy(io.Discard, respList.Body)
				return false
			}
			var apps []struct {
				Slug string `json:"slug"`
			}
			_ = json.NewDecoder(respList.Body).Decode(&apps)
			for _, a := range apps {
				if a.Slug == appResp.Slug {
					return true
				}
			}
			return false
		}, "60s", "2s").Should(BeTrue(), "created application should be in project list")

		By("DELETE /applications/{slug} deletes the application")
		reqDel, _ := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:18080/application/%s", appResp.Slug), nil)
		reqDel.Header.Set("Authorization", "Bearer "+apiKey)
		respDel, err := httpClient.Do(reqDel)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respDel.Body.Close() }()
		Expect(respDel.StatusCode).To(Equal(http.StatusNoContent))

		By("verifying Application CR is gone")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "-n", "default", "get", "application", appCRName)
			_, err := cmd.CombinedOutput()
			return err != nil
		}, "2m", "5s").Should(BeTrue())

		By("verifying ApplicationDomain CRs for this application were cleaned up")
		// domains carry label platform.operator.kibaship.com/application=<application k8s name>
		Eventually(func() string {
			cmd := exec.Command("kubectl", "-n", "default", "get", "applicationdomains.platform.operator.kibaship.com",
				"-l", fmt.Sprintf("platform.operator.kibaship.com/application=%s", appCRName), "-o", "name")
			out, _ := cmd.CombinedOutput()
			return strings.TrimSpace(string(out))
		}, "2m", "5s").Should(Equal(""))
	})
})
