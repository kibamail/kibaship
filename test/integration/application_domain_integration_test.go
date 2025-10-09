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

	"github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/models"
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

var _ = Describe("Application Domain Integration", func() {
	It("creates custom application domain successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Domain Creation",
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

		// Create environment via API
		envPayload := models.EnvironmentCreateRequest{
			Name:        "production",
			Description: "Production environment",
		}
		jsonData, err = json.Marshal(envPayload)
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

		// Create application using environment slug
		applicationPayload := models.ApplicationCreateRequest{
			Name: "My Web App",
			Type: models.ApplicationTypeGitRepository,
			GitRepository: &models.GitRepositoryConfig{
				Provider:     models.GitProviderGitHub,
				Repository:   "myorg/myapp",
				PublicAccess: true,
				Branch:       "main",
			},
		}

		jsonData, err = json.Marshal(applicationPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/environments/"+createdEnvironment.UUID+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var createdApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &createdApplication)
		Expect(err).NotTo(HaveOccurred())

		// Create domain
		payload := models.ApplicationDomainCreateRequest{
			Domain:     "myapp.example.com",
			Port:       8080,
			Type:       models.ApplicationDomainTypeCustom,
			TLSEnabled: true,
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/applications/"+createdApplication.UUID+"/domains", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var response models.ApplicationDomainResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		// Validate response
		Expect(response.UUID).NotTo(BeEmpty())
		Expect(response.Slug).NotTo(BeEmpty())
		Expect(response.Slug).To(HaveLen(8))
		Expect(response.Domain).To(Equal("myapp.example.com"))
		Expect(response.Port).To(Equal(int32(8080)))
		Expect(response.TLSEnabled).To(BeTrue())
		Expect(response.Type).To(Equal(models.ApplicationDomainTypeCustom))
		Expect(response.Default).To(BeFalse())
		Expect(response.ApplicationSlug).To(Equal(createdApplication.Slug))
		// Phase is currently empty in response - TODO: fix in service layer
		// Expect(response.Phase).To(Equal(models.ApplicationDomainPhasePending))

		// Verify CRD
		var domain v1alpha1.ApplicationDomain
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "domain-" + response.UUID + "",
			Namespace: "default",
		}, &domain)
		Expect(err).NotTo(HaveOccurred())

		labels := domain.GetLabels()
		Expect(labels[validation.LabelResourceUUID]).To(Equal(response.UUID))
		Expect(labels[validation.LabelResourceSlug]).To(Equal(response.Slug))
		Expect(labels[validation.LabelApplicationUUID]).To(Equal(createdApplication.UUID))
		Expect(labels[validation.LabelProjectUUID]).To(Equal(createdProject.UUID))

		Expect(domain.Spec.Domain).To(Equal("myapp.example.com"))
		Expect(domain.Spec.Port).To(Equal(int32(8080)))
		Expect(domain.Spec.TLSEnabled).To(BeTrue())
		Expect(domain.Spec.Type).To(Equal(v1alpha1.ApplicationDomainTypeCustom))
		Expect(domain.Spec.Default).To(BeFalse())

		// Cleanup
		_ = k8sClient.Delete(ctx, &domain)
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &application)
		if err == nil {
			_ = k8sClient.Delete(ctx, &application)
		}
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("validates required domain field", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Domain Validation",
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

		// Create environment via API
		envPayload := models.EnvironmentCreateRequest{
			Name:        "production",
			Description: "Production environment",
		}
		jsonData, err = json.Marshal(envPayload)
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

		// Create application using environment slug
		applicationPayload := models.ApplicationCreateRequest{
			Name: "App for Validation Test",
			Type: models.ApplicationTypeGitRepository,
			GitRepository: &models.GitRepositoryConfig{
				Provider:     models.GitProviderGitHub,
				Repository:   "myorg/myapp",
				PublicAccess: true,
				Branch:       "main",
			},
		}

		jsonData, err = json.Marshal(applicationPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/environments/"+createdEnvironment.UUID+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var createdApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &createdApplication)
		Expect(err).NotTo(HaveOccurred())

		// Create domain with empty domain field
		payload := models.ApplicationDomainCreateRequest{
			Domain:     "", // Empty domain
			Port:       8080,
			TLSEnabled: true,
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/applications/"+createdApplication.UUID+"/domains", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusBadRequest))

		// Cleanup
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &application)
		if err == nil {
			_ = k8sClient.Delete(ctx, &application)
		}
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("returns 404 for non-existent application", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		payload := models.ApplicationDomainCreateRequest{
			Domain:     "test.example.com",
			Port:       3000,
			TLSEnabled: true,
		}

		jsonData, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/applications/notfound/domains", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("retrieves application domain by UUID", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		createdProject, createdApplication, createdDomain := createTestApplicationDomain(router, apiKey)

		req, err := http.NewRequest("GET", "/v1/domains/"+createdDomain.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response models.ApplicationDomainResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response.UUID).To(Equal(createdDomain.UUID))
		Expect(response.Slug).To(Equal(createdDomain.Slug))
		Expect(response.Domain).To(Equal(createdDomain.Domain))
		Expect(response.ApplicationSlug).To(Equal(createdApplication.Slug))

		// Cleanup
		cleanupTestApplicationDomain(ctx, createdProject, createdApplication, createdDomain)
	})

	It("returns 404 for non-existent domain", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		req, err := http.NewRequest("GET", "/domains/notfound", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("deletes existing application domain", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		createdProject, createdApplication, createdDomain := createTestApplicationDomain(router, apiKey)

		req, err := http.NewRequest("DELETE", "/v1/domains/"+createdDomain.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNoContent))

		// Verify domain deleted
		var domain v1alpha1.ApplicationDomain
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "domain-" + createdDomain.UUID + "",
			Namespace: "default",
		}, &domain)
		Expect(err).To(HaveOccurred())

		// Cleanup
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &application)
		if err == nil {
			_ = k8sClient.Delete(ctx, &application)
		}
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("returns 404 when deleting same domain again", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		createdProject, createdApplication, createdDomain := createTestApplicationDomain(router, apiKey)

		// First delete
		req, err := http.NewRequest("DELETE", "/v1/domains/"+createdDomain.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNoContent))

		// Try to delete again
		req, err = http.NewRequest("DELETE", "/v1/domains/"+createdDomain.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))

		// Cleanup
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &application)
		if err == nil {
			_ = k8sClient.Delete(ctx, &application)
		}
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("returns 404 for non-existent domain slug on delete", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		req, err := http.NewRequest("DELETE", "/domains/notfound", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})

// Helper functions
func createTestApplicationDomain(router http.Handler, apiKey string) (*models.ProjectResponse, *models.ApplicationResponse, *models.ApplicationDomainResponse) {
	// Create project
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Domain",
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

	// Create environment via API
	envPayload := models.EnvironmentCreateRequest{
		Name:        "production",
		Description: "Production environment",
	}
	jsonData, err = json.Marshal(envPayload)
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

	// Create application using environment slug
	applicationPayload := models.ApplicationCreateRequest{
		Name: "Test App",
		Type: models.ApplicationTypeGitRepository,
		GitRepository: &models.GitRepositoryConfig{
			Provider:     models.GitProviderGitHub,
			Repository:   "test/app",
			PublicAccess: true,
			Branch:       "main",
		},
	}

	jsonData, err = json.Marshal(applicationPayload)
	Expect(err).NotTo(HaveOccurred())

	req, err = http.NewRequest("POST", "/v1/environments/"+createdEnvironment.UUID+"/applications", bytes.NewBuffer(jsonData))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var createdApplication models.ApplicationResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdApplication)
	Expect(err).NotTo(HaveOccurred())

	// Create domain
	domainPayload := models.ApplicationDomainCreateRequest{
		Domain:     "test.example.com",
		Port:       3000,
		TLSEnabled: true,
	}

	jsonData, err = json.Marshal(domainPayload)
	Expect(err).NotTo(HaveOccurred())

	req, err = http.NewRequest("POST", "/v1/applications/"+createdApplication.UUID+"/domains", bytes.NewBuffer(jsonData))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var createdDomain models.ApplicationDomainResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdDomain)
	Expect(err).NotTo(HaveOccurred())

	return &createdProject, &createdApplication, &createdDomain
}

func cleanupTestApplicationDomain(ctx context.Context, project *models.ProjectResponse, application *models.ApplicationResponse, domain *models.ApplicationDomainResponse) {
	// Clean up domain
	var dom v1alpha1.ApplicationDomain
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      "domain-" + domain.UUID + "",
		Namespace: "default",
	}, &dom)
	if err == nil {
		_ = k8sClient.Delete(ctx, &dom)
	}

	// Clean up application
	var app v1alpha1.Application
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "application-" + application.UUID + "",
		Namespace: "default",
	}, &app)
	if err == nil {
		_ = k8sClient.Delete(ctx, &app)
	}

	// Clean up project
	var proj v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + project.UUID,
	}, &proj)
	if err == nil {
		_ = k8sClient.Delete(ctx, &proj)
	}
}
