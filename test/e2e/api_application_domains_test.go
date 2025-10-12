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

var _ = Describe("API Server Application Domains Auto-Loading", func() {
	It("creates a GitRepository application and GET /v1/applications/{uuid} returns its default domain", func() {
		By("fetching API key from api-server secret")
		cmd := exec.Command("kubectl", "get", "secret", "api-server-api-key", "-n", "kibaship", "-o", "jsonpath={.data.api-key}")
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to get api key secret: %s", string(output)))
		encodedKey := strings.TrimSpace(string(output))
		decodedKeyBytes, err := base64.StdEncoding.DecodeString(encodedKey)
		Expect(err).NotTo(HaveOccurred(), "failed to base64 decode api key")
		apiKey := strings.TrimSpace(string(decodedKeyBytes))

		By("port-forwarding API service to localhost:18080")
		pfCmd := exec.Command("kubectl", "-n", "kibaship", "port-forward", "svc/apiserver", "18080:80")
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

		By("creating a project via POST /v1/projects to host the application")
		workspaceUUID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
		projReqBody := map[string]any{
			"name":          "test-project-app-domain-api-e2e",
			"workspaceUuid": workspaceUUID,
		}
		projBytes, _ := json.Marshal(projReqBody)
		reqProj, _ := http.NewRequest("POST", "http://127.0.0.1:18080/v1/projects", bytes.NewReader(projBytes))
		reqProj.Header.Set("Content-Type", "application/json")
		reqProj.Header.Set("Authorization", "Bearer "+apiKey)
		httpClient := &http.Client{Timeout: 30 * time.Second}
		respProj, err := httpClient.Do(reqProj)
		Expect(err).NotTo(HaveOccurred(), "HTTP request to create project failed")
		defer func() { _ = respProj.Body.Close() }()
		Expect(respProj.StatusCode).To(Equal(http.StatusCreated))
		var projResp struct {
			UUID string `json:"uuid"`
		}
		_ = json.NewDecoder(respProj.Body).Decode(&projResp)
		Expect(projResp.UUID).NotTo(BeEmpty(), "project UUID should be present")

		By("creating an environment via POST /v1/projects/{uuid}/environments")
		envReqBody := map[string]any{
			"name":        "production",
			"description": "Production environment",
		}
		envBytes, _ := json.Marshal(envReqBody)
		reqEnv, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:18080/v1/projects/%s/environments", projResp.UUID), bytes.NewReader(envBytes))
		reqEnv.Header.Set("Content-Type", "application/json")
		reqEnv.Header.Set("Authorization", "Bearer "+apiKey)
		respEnv, err := httpClient.Do(reqEnv)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respEnv.Body.Close() }()
		Expect(respEnv.StatusCode).To(Equal(http.StatusCreated))
		var envResp struct {
			UUID string `json:"uuid"`
		}
		_ = json.NewDecoder(respEnv.Body).Decode(&envResp)
		Expect(envResp.UUID).NotTo(BeEmpty())

		By("creating an application via POST /v1/environments/{uuid}/applications (type: GitRepository)")
		appReqBody := map[string]any{
			"name": "my-web-app-e2e",
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
		reqApp, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:18080/v1/environments/%s/applications", envResp.UUID), bytes.NewReader(appBytes))
		reqApp.Header.Set("Content-Type", "application/json")
		reqApp.Header.Set("Authorization", "Bearer "+apiKey)
		respApp, err := httpClient.Do(reqApp)
		Expect(err).NotTo(HaveOccurred(), "HTTP request to create application failed")
		defer func() { _ = respApp.Body.Close() }()
		Expect(respApp.StatusCode).To(Equal(http.StatusCreated))
		var appResp struct {
			UUID string `json:"uuid"`
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respApp.Body).Decode(&appResp)
		Expect(appResp.UUID).NotTo(BeEmpty(), "application slug should be present")
		Expect(appResp.UUID).NotTo(BeEmpty(), "application UUID should be present")

		appCRName := fmt.Sprintf("application-%s", appResp.UUID)
		By("waiting for Application CR to become Ready")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "-n", "default", "get", "application", appCRName, "-o", "jsonpath={.status.phase}")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return false
			}
			return strings.TrimSpace(string(output)) == readyPhase
		}, "4m", "5s").Should(BeTrue(), "Application should become Ready")

		By("eventually fetching GET /v1/applications/{uuid} and seeing the default domain auto-loaded")
		targetDomainSuffix := ".myapps.kibaship.com"
		Eventually(func() error {
			req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:18080/v1/applications/%s", appResp.UUID), nil)
			req.Header.Set("Authorization", "Bearer "+apiKey)
			resp, err := httpClient.Do(req)
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status: %d", resp.StatusCode)
			}
			var getResp struct {
				Slug    string `json:"slug"`
				Domains []struct {
					Domain  string `json:"domain"`
					Type    string `json:"type"`
					Default bool   `json:"default"`
				} `json:"domains"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
				return err
			}
			if getResp.Slug != appResp.Slug {
				return fmt.Errorf("slug mismatch: %s", getResp.Slug)
			}
			if len(getResp.Domains) == 0 {
				return fmt.Errorf("no domains returned")
			}
			for _, d := range getResp.Domains {
				if d.Default || strings.EqualFold(d.Type, "default") {
					if !strings.HasSuffix(d.Domain, targetDomainSuffix) {
						return fmt.Errorf("default domain %s does not end with %s", d.Domain, targetDomainSuffix)
					}
					return nil
				}
			}
			return fmt.Errorf("default domain not found")
		}, "4m", "5s").Should(Succeed())
	})
})
