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

	"github.com/kibamail/kibaship/pkg/utils"
)

// Covers: POST/GET/DELETE application domains via API
var _ = Describe("API Server Application Domain CRUD", func() {
	It("creates a custom domain, fetches it, then deletes it and verifies CR removal", func() {
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

		httpClient := &http.Client{Timeout: 30 * time.Second}

		var domainCRName string

		By("creating a project via POST /v1/projects")
		workspaceUUID := workspaceUUIDConst
		projReqBody := map[string]any{
			"name":          "proj-for-domain-crud-e2e",
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
		}
		_ = json.NewDecoder(respProj.Body).Decode(&projResp)
		Expect(projResp.UUID).NotTo(BeEmpty())

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

		By("creating an application via POST /v1/environments/{uuid}/applications (GitRepository)")
		appReqBody := map[string]any{
			"name": "app-for-domain-crud-e2e",
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
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respApp.Body.Close() }()
		Expect(respApp.StatusCode).To(Equal(http.StatusCreated))
		var appResp struct {
			UUID string `json:"uuid"`
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respApp.Body).Decode(&appResp)
		Expect(appResp.Slug).NotTo(BeEmpty())

		By("creating a custom domain via POST /v1/applications/{uuid}/domains")
		domainReq := map[string]any{
			"domain":  "custom-e2e.example.com",
			"port":    3000,
			"type":    "custom",
			"default": false,
		}
		domainBytes, _ := json.Marshal(domainReq)
		reqDom, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:18080/v1/applications/%s/domains", appResp.UUID), bytes.NewReader(domainBytes))
		reqDom.Header.Set("Content-Type", "application/json")
		reqDom.Header.Set("Authorization", "Bearer "+apiKey)
		respDom, err := httpClient.Do(reqDom)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respDom.Body.Close() }()
		Expect(respDom.StatusCode).To(Equal(http.StatusCreated))
		var domResp struct {
			UUID string `json:"uuid"`
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respDom.Body).Decode(&domResp)
		Expect(domResp.UUID).NotTo(BeEmpty())
		Expect(domResp.UUID).NotTo(BeEmpty())

		By("GET /v1/domains/{uuid} returns the domain")
		reqGet, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:18080/v1/domains/%s", domResp.UUID), nil)
		reqGet.Header.Set("Authorization", "Bearer "+apiKey)
		respGet, err := httpClient.Do(reqGet)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respGet.Body.Close() }()
		Expect(respGet.StatusCode).To(Equal(http.StatusOK))

		By("verifying Certificate is created and status.certificateRef is set")
		domainCRName = utils.GetApplicationDomainResourceName(domResp.UUID)
		// Certificate should be named ad-<applicationdomain-name> in the certificates namespace
		certName := fmt.Sprintf("ad-%s", domainCRName)
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "-n", "certificates", "get", "certificate", certName)
			out, err := cmd.CombinedOutput()
			_ = out
			return err == nil
		}, "2m", "5s").Should(BeTrue(), "expected certificate to be created")

		// status.certificateRef should point to that certificate
		Eventually(func() (string, error) {
			cmd := exec.Command("kubectl", "-n", "default", "get", "applicationdomains.platform.operator.kibaship.com", domainCRName,
				"-o", "jsonpath={.status.certificateRef.name}:{.status.certificateRef.namespace}")
			out, err := cmd.CombinedOutput()
			return string(out), err
		}, "2m", "5s").Should(Equal(certName + ":certificates"))

		// Certificate should carry key labels propagated from the ApplicationDomain
		Eventually(func() (string, error) {
			cmd := exec.Command("kubectl", "-n", "certificates", "get", "certificate", certName,
				"-o", "jsonpath={.metadata.labels.platform\\.kibaship\\.com/uuid}")
			out, err := cmd.CombinedOutput()
			return strings.TrimSpace(string(out)), err
		}, "2m", "5s").ShouldNot(BeEmpty())
		Eventually(func() (string, error) {
			cmd := exec.Command("kubectl", "-n", "certificates", "get", "certificate", certName,
				"-o", "jsonpath={.metadata.labels.platform\\.kibaship\\.com/project-uuid}")
			out, err := cmd.CombinedOutput()
			return strings.TrimSpace(string(out)), err
		}, "2m", "5s").ShouldNot(BeEmpty())

		By("DELETE /v1/domains/{uuid} deletes the domain")
		reqDel, _ := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:18080/v1/domains/%s", domResp.UUID), nil)
		reqDel.Header.Set("Authorization", "Bearer "+apiKey)
		respDel, err := httpClient.Do(reqDel)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respDel.Body.Close() }()
		Expect(respDel.StatusCode).To(Equal(http.StatusNoContent))

		By("verifying ApplicationDomain CR for UUID is gone")
		// The CR name pattern for custom domains comes from service: domain-<uuid>
		domainCRName = utils.GetApplicationDomainResourceName(domResp.UUID)
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "-n", "default", "get", "applicationdomains.platform.operator.kibaship.com", domainCRName)
			_, err := cmd.CombinedOutput()
			return err != nil
		}, "2m", "5s").Should(BeTrue())
	})
})
