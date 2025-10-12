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

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/models"
	"github.com/kibamail/kibaship/pkg/validation"
)

var _ = Describe("Environment Integration", func() {
	It("creates minimal environment successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create a project for environments
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Minimal Env",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(projectPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/v1/projects", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var createdProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &createdProject)
		Expect(err).NotTo(HaveOccurred())

		// Create environment
		payload := models.EnvironmentCreateRequest{
			Name: "Production",
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/projects/"+createdProject.UUID+"/environments", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var response models.EnvironmentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		// Validate response
		Expect(response.UUID).NotTo(BeEmpty())
		Expect(response.Slug).NotTo(BeEmpty())
		Expect(response.Slug).To(HaveLen(8))
		Expect(response.Name).To(Equal("Production"))
		Expect(response.ProjectUUID).To(Equal(createdProject.UUID))
		Expect(response.ProjectSlug).To(Equal(createdProject.Slug))
		Expect(response.ApplicationCount).To(Equal(int32(0)))
		Expect(response.CreatedAt).NotTo(BeZero())

		// Verify Environment CRD
		var environment v1alpha1.Environment
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "environment-" + response.UUID + "",
			Namespace: "default",
		}, &environment)
		Expect(err).NotTo(HaveOccurred())

		labels := environment.GetLabels()
		Expect(labels[validation.LabelResourceUUID]).To(Equal(response.UUID))
		Expect(labels[validation.LabelResourceSlug]).To(Equal(response.Slug))
		Expect(labels[validation.LabelProjectUUID]).To(Equal(response.ProjectUUID))

		// Clean up
		_ = k8sClient.Delete(ctx, &environment)
		var project v1alpha1.Project
		_ = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		_ = k8sClient.Delete(ctx, &project)
	})

	It("creates environment with description and variables", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Env With Vars",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(projectPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/v1/projects", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var createdProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &createdProject)
		Expect(err).NotTo(HaveOccurred())

		// Create environment with variables
		payload := models.EnvironmentCreateRequest{
			Name:        "Staging",
			Description: "Staging environment for testing",
			Variables: map[string]string{
				"DB_HOST": "staging-db.example.com",
				"API_URL": "https://staging-api.example.com",
			},
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/projects/"+createdProject.UUID+"/environments", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var response models.EnvironmentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response.Name).To(Equal("Staging"))
		Expect(response.Description).To(Equal("Staging environment for testing"))
		Expect(response.Variables).To(HaveLen(2))
		Expect(response.Variables["DB_HOST"]).To(Equal("staging-db.example.com"))

		// Verify in Kubernetes
		var environment v1alpha1.Environment
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "environment-" + response.UUID + "",
			Namespace: "default",
		}, &environment)
		Expect(err).NotTo(HaveOccurred())

		annotations := environment.GetAnnotations()
		Expect(annotations[validation.AnnotationResourceDescription]).To(Equal("Staging environment for testing"))
		// Note: Variables are no longer stored on Environment CRD - they belong at Application level

		// Clean up
		_ = k8sClient.Delete(ctx, &environment)
		var project v1alpha1.Project
		_ = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		_ = k8sClient.Delete(ctx, &project)
	})

	It("returns 400 for missing name", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Invalid Env",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(projectPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/v1/projects", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var createdProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &createdProject)
		Expect(err).NotTo(HaveOccurred())

		// Try to create environment with empty name
		payload := models.EnvironmentCreateRequest{
			Name: "",
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/projects/"+createdProject.UUID+"/environments", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusBadRequest))

		// Clean up
		var project v1alpha1.Project
		_ = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		_ = k8sClient.Delete(ctx, &project)
	})

	It("returns 404 for non-existent project", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		payload := models.EnvironmentCreateRequest{
			Name: "Production",
		}

		jsonData, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/v1/projects/00000000-0000-0000-0000-000000000000/environments", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("retrieves environment by slug", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project and environment
		createdProject, createdEnvironment := createTestEnvironment(router, apiKey)

		// Retrieve environment
		req, err := http.NewRequest("GET", "/v1/environments/"+createdEnvironment.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response models.EnvironmentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response.UUID).To(Equal(createdEnvironment.UUID))
		Expect(response.Slug).To(Equal(createdEnvironment.Slug))

		// Clean up
		cleanupTestEnvironment(ctx, createdProject, createdEnvironment)
	})

	It("retrieves environments by project", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project and environment
		createdProject, createdEnvironment := createTestEnvironment(router, apiKey)

		// Retrieve environments by project
		req, err := http.NewRequest("GET", "/v1/projects/"+createdProject.UUID+"/environments", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response []*models.EnvironmentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response).To(HaveLen(1))
		Expect(response[0].UUID).To(Equal(createdEnvironment.UUID))

		// Clean up
		cleanupTestEnvironment(ctx, createdProject, createdEnvironment)
	})

	It("returns 404 for non-existent environment", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		req, err := http.NewRequest("GET", "/v1/environments/00000000-0000-0000-0000-000000000000", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("updates environment description", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project and environment
		createdProject, createdEnvironment := createTestEnvironment(router, apiKey)

		// Update environment
		updateReq := models.EnvironmentUpdateRequest{
			Description: stringPtr("Updated production environment"),
		}

		jsonData, err := json.Marshal(updateReq)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("PATCH", "/v1/environments/"+createdEnvironment.UUID, bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response models.EnvironmentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response.Description).To(Equal("Updated production environment"))

		// Clean up
		cleanupTestEnvironment(ctx, createdProject, createdEnvironment)
	})

	It("updates environment description only (variables removed from environment)", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project and environment
		createdProject, createdEnvironment := createTestEnvironment(router, apiKey)

		// Update environment description (variables are no longer supported on environments)
		updateReq := models.EnvironmentUpdateRequest{
			Description: stringPtr("Updated with new description"),
		}

		jsonData, err := json.Marshal(updateReq)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("PATCH", "/v1/environments/"+createdEnvironment.UUID, bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response models.EnvironmentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response.Description).To(Equal("Updated with new description"))
		// Note: Variables are now managed at Application level, not Environment level

		// Clean up
		cleanupTestEnvironment(ctx, createdProject, createdEnvironment)
	})

	It("returns 404 for non-existent environment update", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		updateReq := models.EnvironmentUpdateRequest{
			Description: stringPtr("Updated"),
		}

		jsonData, err := json.Marshal(updateReq)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("PATCH", "/v1/environments/00000000-0000-0000-0000-000000000000", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("deletes existing environment", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project and environment
		createdProject, createdEnvironment := createTestEnvironment(router, apiKey)

		// Delete environment
		req, err := http.NewRequest("DELETE", "/v1/environments/"+createdEnvironment.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNoContent))

		// Verify deleted
		var environment v1alpha1.Environment
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "environment-" + createdEnvironment.UUID + "",
			Namespace: "default",
		}, &environment)
		Expect(err).To(HaveOccurred())

		// Clean up project
		var project v1alpha1.Project
		_ = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		_ = k8sClient.Delete(ctx, &project)
	})

	It("returns 404 when deleting same environment again", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project and environment
		createdProject, createdEnvironment := createTestEnvironment(router, apiKey)

		// First delete
		req, err := http.NewRequest("DELETE", "/v1/environments/"+createdEnvironment.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNoContent))

		// Try to delete again
		req, err = http.NewRequest("DELETE", "/v1/environments/"+createdEnvironment.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))

		// Clean up project
		var project v1alpha1.Project
		_ = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		_ = k8sClient.Delete(ctx, &project)
	})

	It("returns 404 for non-existent environment UUID", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		req, err := http.NewRequest("DELETE", "/v1/environments/00000000-0000-0000-0000-000000000000", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})

// Helper functions
func createTestEnvironment(router http.Handler, apiKey string) (*models.ProjectResponse, *models.EnvironmentResponse) {
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Environment",
		WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	jsonData, err := json.Marshal(projectPayload)
	Expect(err).NotTo(HaveOccurred())

	req, err := http.NewRequest("POST", "/v1/projects", bytes.NewBuffer(jsonData))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var createdProject models.ProjectResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdProject)
	Expect(err).NotTo(HaveOccurred())

	environmentPayload := models.EnvironmentCreateRequest{
		Name: "Production",
	}

	jsonData, err = json.Marshal(environmentPayload)
	Expect(err).NotTo(HaveOccurred())

	req, err = http.NewRequest("POST", "/v1/projects/"+createdProject.UUID+"/environments", bytes.NewBuffer(jsonData))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var createdEnvironment models.EnvironmentResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdEnvironment)
	Expect(err).NotTo(HaveOccurred())

	return &createdProject, &createdEnvironment
}

func cleanupTestEnvironment(ctx context.Context, project *models.ProjectResponse, environment *models.EnvironmentResponse) {
	var env v1alpha1.Environment
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      "environment-" + environment.UUID + "",
		Namespace: "default",
	}, &env)
	if err == nil {
		_ = k8sClient.Delete(ctx, &env)
	}

	var proj v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + project.UUID,
	}, &proj)
	if err == nil {
		_ = k8sClient.Delete(ctx, &proj)
	}
}
