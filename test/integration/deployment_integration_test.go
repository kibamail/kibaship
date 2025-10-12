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

var _ = Describe("Deployment Integration", func() {
	It("creates GitRepository deployment successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Deployment",
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
			Name: "My Git App",
			Type: models.ApplicationTypeGitRepository,
			GitRepository: &models.GitRepositoryConfig{
				Provider:     models.GitProviderGitHub,
				Repository:   "myorg/myapp",
				Branch:       "main",
				PublicAccess: true,
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

		// Create deployment
		payload := models.DeploymentCreateRequest{
			ApplicationUUID: createdApplication.UUID,
			GitRepository: &models.GitRepositoryDeploymentConfig{
				CommitSHA: "abc123def456",
				Branch:    "main",
			},
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/applications/"+createdApplication.UUID+"/deployments", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var response models.DeploymentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		// Validate response
		Expect(response.UUID).NotTo(BeEmpty())
		Expect(response.Slug).NotTo(BeEmpty())
		Expect(response.Slug).To(HaveLen(8))
		Expect(response.ApplicationSlug).To(Equal(createdApplication.Slug))
		Expect(response.Phase).To(Equal(models.DeploymentPhaseInitializing))
		Expect(response.GitRepository).NotTo(BeNil())
		Expect(response.GitRepository.CommitSHA).To(Equal("abc123def456"))
		Expect(response.GitRepository.Branch).To(Equal("main"))

		// Verify CRD
		var deployment v1alpha1.Deployment
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "deployment-" + response.UUID + "",
			Namespace: "default",
		}, &deployment)
		Expect(err).NotTo(HaveOccurred())

		labels := deployment.GetLabels()
		Expect(labels[validation.LabelResourceUUID]).To(Equal(response.UUID))
		Expect(labels[validation.LabelResourceSlug]).To(Equal(response.Slug))
		Expect(labels[validation.LabelProjectUUID]).To(Equal(response.ProjectUUID))
		Expect(labels[validation.LabelApplicationUUID]).To(Equal(response.ApplicationUUID))

		Expect(deployment.Spec.GitRepository).NotTo(BeNil())
		Expect(deployment.Spec.GitRepository.CommitSHA).To(Equal("abc123def456"))
		Expect(deployment.Spec.GitRepository.Branch).To(Equal("main"))

		// Cleanup
		_ = k8sClient.Delete(ctx, &deployment)
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &application)
		if err == nil {
			_ = k8sClient.Delete(ctx, &application)
		}
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.UUID + "",
		}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("retrieves deployment by UUID", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		createdProject, createdApplication, createdDeployment := createTestDeployment(router, apiKey)

		req, err := http.NewRequest("GET", "/v1/deployments/"+createdDeployment.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response models.DeploymentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response.UUID).To(Equal(createdDeployment.UUID))
		Expect(response.Slug).To(Equal(createdDeployment.Slug))
		Expect(response.ApplicationUUID).To(Equal(createdApplication.UUID))

		// Cleanup
		cleanupTestDeployment(ctx, createdProject, createdApplication, createdDeployment)
	})

	It("retrieves deployments by application", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		createdProject, createdApplication, createdDeployment := createTestDeployment(router, apiKey)

		req, err := http.NewRequest("GET", "/v1/applications/"+createdApplication.UUID+"/deployments", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response []models.DeploymentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response).To(HaveLen(1))
		Expect(response[0].UUID).To(Equal(createdDeployment.UUID))

		// Cleanup
		cleanupTestDeployment(ctx, createdProject, createdApplication, createdDeployment)
	})

	It("promotes deployment successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		createdProject, createdApplication, createdDeployment := createTestDeployment(router, apiKey)

		// Promote the deployment
		req, err := http.NewRequest("POST", "/v1/deployments/"+createdDeployment.UUID+"/promote", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(response["message"]).To(Equal("Deployment promoted successfully"))

		// Verify that the application's currentDeploymentRef is updated
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &application)
		Expect(err).NotTo(HaveOccurred())

		Expect(application.Spec.CurrentDeploymentRef).NotTo(BeNil())
		Expect(application.Spec.CurrentDeploymentRef.Name).To(Equal("deployment-" + createdDeployment.UUID))

		// Cleanup
		cleanupTestDeployment(ctx, createdProject, createdApplication, createdDeployment)
	})

	It("returns 404 when promoting non-existent deployment", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Try to promote non-existent deployment
		req, err := http.NewRequest("POST", "/v1/deployments/nonexistent-uuid/promote", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(response["error"]).To(Equal("Not Found"))
	})

	It("creates ImageFromRegistry deployment successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for ImageFromRegistry",
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

		// Create ImageFromRegistry application
		applicationPayload := models.ApplicationCreateRequest{
			Name: "My Nginx App",
			Type: models.ApplicationTypeImageFromRegistry,
			ImageFromRegistry: &models.ImageFromRegistryConfig{
				Registry:   "dockerhub",
				Repository: "library/nginx",
				DefaultTag: "1.21",
				Port:       80,
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

		// Create deployment with specific tag
		payload := models.DeploymentCreateRequest{
			ApplicationUUID: createdApplication.UUID,
			ImageFromRegistry: &models.ImageFromRegistryDeploymentConfig{
				Tag: "1.22",
			},
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/applications/"+createdApplication.UUID+"/deployments", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var response models.DeploymentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		// Validate response
		Expect(response.UUID).NotTo(BeEmpty())
		Expect(response.Slug).NotTo(BeEmpty())
		Expect(response.Slug).To(HaveLen(8))
		Expect(response.ApplicationSlug).To(Equal(createdApplication.Slug))
		Expect(response.Phase).To(Equal(models.DeploymentPhaseInitializing))
		Expect(response.ImageFromRegistry).NotTo(BeNil())
		Expect(response.ImageFromRegistry.Tag).To(Equal("1.22"))

		// Verify CRD
		var deployment v1alpha1.Deployment
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "deployment-" + response.UUID + "",
			Namespace: "default",
		}, &deployment)
		Expect(err).NotTo(HaveOccurred())

		// Validate CRD fields
		Expect(deployment.Spec.ApplicationRef.Name).To(Equal("application-" + createdApplication.UUID))
		Expect(deployment.Spec.ImageFromRegistry).NotTo(BeNil())
		Expect(deployment.Spec.ImageFromRegistry.Tag).To(Equal("1.22"))

		// Cleanup
		defer func() {
			var proj v1alpha1.Project
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &proj); err == nil {
				_ = k8sClient.Delete(ctx, &proj)
			}
		}()
	})
})

// Helper function to create a test deployment
func createTestDeployment(router http.Handler, apiKey string) (*models.ProjectResponse, *models.ApplicationResponse, *models.DeploymentResponse) {
	// Create project
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Deployment Retrieval",
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
		Description: "Production environment for testing",
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
		Name: "Test Git App",
		Type: models.ApplicationTypeGitRepository,
		GitRepository: &models.GitRepositoryConfig{
			Provider:     models.GitProviderGitHub,
			Repository:   "test/repo",
			Branch:       "main",
			PublicAccess: true,
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

	// Create deployment
	deploymentPayload := models.DeploymentCreateRequest{
		ApplicationUUID: createdApplication.UUID,
		GitRepository: &models.GitRepositoryDeploymentConfig{
			CommitSHA: "test123commit",
			Branch:    "develop",
		},
	}

	jsonData, err = json.Marshal(deploymentPayload)
	Expect(err).NotTo(HaveOccurred())

	req, err = http.NewRequest("POST", "/v1/applications/"+createdApplication.UUID+"/deployments", bytes.NewBuffer(jsonData))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var createdDeployment models.DeploymentResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdDeployment)
	Expect(err).NotTo(HaveOccurred())

	return &createdProject, &createdApplication, &createdDeployment
}

// Helper function to clean up test resources
func cleanupTestDeployment(ctx context.Context, project *models.ProjectResponse, application *models.ApplicationResponse, deployment *models.DeploymentResponse) {
	// Clean up deployment
	var dep v1alpha1.Deployment
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      "deployment-" + deployment.UUID + "",
		Namespace: "default",
	}, &dep)
	if err == nil {
		_ = k8sClient.Delete(ctx, &dep)
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
		Name:      "project-" + project.UUID + "",
		Namespace: "default",
	}, &proj)
	if err == nil {
		_ = k8sClient.Delete(ctx, &proj)
	}
}
