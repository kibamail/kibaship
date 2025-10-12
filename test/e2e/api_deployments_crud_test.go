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

// Covers: POST/GET list deployments endpoints and Tekton reconciliation side-effects
var _ = Describe("API Server Deployment CRUD", func() {
	It("creates a deployment via API, lists and fetches it, and verifies Tekton resources + LatestDeployment", func() {
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

		By("creating a project via POST /v1/projects")
		workspaceUUID := workspaceUUIDConst
		projReqBody := map[string]any{
			"name":          "proj-for-deploy-crud-e2e",
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
			"name": "app-for-deploy-crud-e2e",
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
		}
		_ = json.NewDecoder(respApp.Body).Decode(&appResp)
		Expect(appResp.UUID).NotTo(BeEmpty())

		By("creating a deployment via POST /v1/applications/{uuid}/deployments")
		deployReqBody := map[string]any{
			"gitRepository": map[string]any{
				"commitSHA": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // dummy SHA
				"branch":    "main",
			},
		}
		deployBytes, _ := json.Marshal(deployReqBody)
		reqDep, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:18080/v1/applications/%s/deployments", appResp.UUID), bytes.NewReader(deployBytes))
		reqDep.Header.Set("Content-Type", "application/json")
		reqDep.Header.Set("Authorization", "Bearer "+apiKey)
		respDep, err := httpClient.Do(reqDep)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respDep.Body.Close() }()
		if respDep.StatusCode != http.StatusCreated {
			bodyBytes, _ := io.ReadAll(respDep.Body)
			_, _ = fmt.Fprintf(GinkgoWriter, "Deployment creation failed. Status: %d, Body: %s\n", respDep.StatusCode, string(bodyBytes))
		}
		Expect(respDep.StatusCode).To(Equal(http.StatusCreated))
		var depResp struct {
			UUID string `json:"uuid"`
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respDep.Body).Decode(&depResp)
		Expect(depResp.UUID).NotTo(BeEmpty())
		Expect(depResp.UUID).NotTo(BeEmpty())

		pipelineName := fmt.Sprintf("pipeline-%s", depResp.UUID)

		By("verifying Tekton Pipeline created for deployment")
		Eventually(func() bool {
			cmd := exec.Command("kubectl", "-n", "default", "get", "pipeline.tekton.dev", pipelineName)
			_, err := cmd.CombinedOutput()
			return err == nil
		}, "3m", "5s").Should(BeTrue())

		By("verifying Tekton PipelineRun created for deployment generation")
		Eventually(func() string {
			// Select by label tekton.dev/pipeline=<pipelineName>
			cmd := exec.Command("kubectl", "-n", "default", "get", "pipelineruns.tekton.dev", "-l", fmt.Sprintf("tekton.dev/pipeline=%s", pipelineName), "-o", "name")
			out, _ := cmd.CombinedOutput()
			return strings.TrimSpace(string(out))
		}, "4m", "5s").ShouldNot(Equal(""))

		By("GET /v1/applications/{uuid}/deployments lists the deployment")
		reqList, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:18080/v1/applications/%s/deployments", appResp.UUID), nil)
		reqList.Header.Set("Authorization", "Bearer "+apiKey)
		respList, err := httpClient.Do(reqList)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respList.Body.Close() }()
		Expect(respList.StatusCode).To(Equal(http.StatusOK))
		var listResp []struct {
			Slug string `json:"slug"`
		}
		_ = json.NewDecoder(respList.Body).Decode(&listResp)
		Expect(listResp).NotTo(BeEmpty())
		Expect(func() bool {
			for _, d := range listResp {
				if d.Slug == depResp.Slug {
					return true
				}
			}
			return false
		}()).To(BeTrue(), "created deployment should be listed")

		By("GET /v1/deployments/{uuid} returns the deployment")
		reqGet, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:18080/v1/deployments/%s", depResp.UUID), nil)
		reqGet.Header.Set("Authorization", "Bearer "+apiKey)
		respGet, err := httpClient.Do(reqGet)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = respGet.Body.Close() }()
		Expect(respGet.StatusCode).To(Equal(http.StatusOK))

		By("GET /v1/environments/{uuid}/applications shows LatestDeployment populated")
		reqProjApps, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:18080/v1/environments/%s/applications", envResp.UUID), nil)
		reqProjApps.Header.Set("Authorization", "Bearer "+apiKey)
		Eventually(func() bool {
			resp, err := httpClient.Do(reqProjApps)
			if err != nil {
				return false
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				return false
			}
			var apps []struct {
				UUID             string      `json:"uuid"`
				LatestDeployment interface{} `json:"latestDeployment"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&apps)
			for _, a := range apps {
				if a.UUID == appResp.UUID {
					return a.LatestDeployment != nil
				}
			}
			return false
		}, "4m", "5s").Should(BeTrue())
	})
})
