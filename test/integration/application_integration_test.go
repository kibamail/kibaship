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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/models"
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

func TestApplicationCreationIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// First, create a project
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Apps",
		WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	jsonData, err := json.Marshal(projectPayload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdProject models.ProjectResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdProject)
	require.NoError(t, err)

	tests := []struct {
		name           string
		payload        models.ApplicationCreateRequest
		expectedStatus int
		validateFunc   func(*testing.T, *models.ApplicationResponse)
	}{
		{
			name: "Create DockerImage application",
			payload: models.ApplicationCreateRequest{
				Name: "My Web App",
				Type: models.ApplicationTypeDockerImage,
				DockerImage: &models.DockerImageConfig{
					Image: "nginx:latest",
					Tag:   "v1.0.0",
				},
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, resp *models.ApplicationResponse) {
				// Verify response structure
				assert.NotEmpty(t, resp.UUID)
				assert.NotEmpty(t, resp.Slug)
				assert.Len(t, resp.Slug, 8)
				assert.Equal(t, "My Web App", resp.Name)
				assert.Equal(t, createdProject.UUID, resp.ProjectUUID)
				assert.Equal(t, createdProject.Slug, resp.ProjectSlug)
				assert.Equal(t, models.ApplicationTypeDockerImage, resp.Type)
				assert.Equal(t, "Pending", resp.Status)
				assert.NotZero(t, resp.CreatedAt)

				// Verify DockerImage configuration
				require.NotNil(t, resp.DockerImage)
				assert.Equal(t, "nginx:latest", resp.DockerImage.Image)
				assert.Equal(t, "v1.0.0", resp.DockerImage.Tag)

				// Verify Application CRD was actually created in Kubernetes
				var application v1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      "application-" + resp.Slug + "-kibaship-com",
					Namespace: "default",
				}, &application)
				require.NoError(t, err, "Application CRD should exist in Kubernetes")

				// Verify CRD labels
				labels := application.GetLabels()
				assert.Equal(t, resp.UUID, labels[validation.LabelResourceUUID])
				assert.Equal(t, resp.Slug, labels[validation.LabelResourceSlug])
				assert.Equal(t, resp.ProjectUUID, labels[validation.LabelProjectUUID])

				// Verify CRD annotations
				annotations := application.GetAnnotations()
				assert.Equal(t, resp.Name, annotations[validation.AnnotationResourceName])

				// Verify CRD spec
				assert.Equal(t, v1alpha1.ApplicationTypeDockerImage, application.Spec.Type)
				require.NotNil(t, application.Spec.DockerImage)
				assert.Equal(t, "nginx:latest", application.Spec.DockerImage.Image)
				assert.Equal(t, "v1.0.0", application.Spec.DockerImage.Tag)

				// Clean up - delete the created application
				err = k8sClient.Delete(ctx, &application)
				assert.NoError(t, err)
			},
		},
		{
			name: "Create GitRepository application",
			payload: models.ApplicationCreateRequest{
				Name: "My Node App",
				Type: models.ApplicationTypeGitRepository,
				GitRepository: &models.GitRepositoryConfig{
					Provider:           models.GitProviderGitHub,
					Repository:         "myorg/myapp",
					PublicAccess:       true,
					Branch:             "main",
					BuildCommand:       "npm run build",
					StartCommand:       "npm start",
					RootDirectory:      "./",
					SpaOutputDirectory: "dist",
				},
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, resp *models.ApplicationResponse) {
				assert.Equal(t, "My Node App", resp.Name)
				assert.Equal(t, models.ApplicationTypeGitRepository, resp.Type)

				// Verify GitRepository configuration
				require.NotNil(t, resp.GitRepository)
				assert.Equal(t, models.GitProviderGitHub, resp.GitRepository.Provider)
				assert.Equal(t, "myorg/myapp", resp.GitRepository.Repository)
				assert.True(t, resp.GitRepository.PublicAccess)
				assert.Equal(t, "main", resp.GitRepository.Branch)
				assert.Equal(t, "npm run build", resp.GitRepository.BuildCommand)
				assert.Equal(t, "npm start", resp.GitRepository.StartCommand)

				// Verify in Kubernetes
				var application v1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      "application-" + resp.Slug + "-kibaship-com",
					Namespace: "default",
				}, &application)
				require.NoError(t, err)

				assert.Equal(t, v1alpha1.ApplicationTypeGitRepository, application.Spec.Type)
				require.NotNil(t, application.Spec.GitRepository)
				assert.Equal(t, v1alpha1.GitProviderGitHub, application.Spec.GitRepository.Provider)
				assert.Equal(t, "myorg/myapp", application.Spec.GitRepository.Repository)

				// Clean up
				err = k8sClient.Delete(ctx, &application)
				assert.NoError(t, err)
			},
		},
		{
			name: "Create MySQL application",
			payload: models.ApplicationCreateRequest{
				Name: "My Database",
				Type: models.ApplicationTypeMySQL,
				MySQL: &models.MySQLConfig{
					Version:  "8.0",
					Database: "myapp",
				},
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, resp *models.ApplicationResponse) {
				assert.Equal(t, "My Database", resp.Name)
				assert.Equal(t, models.ApplicationTypeMySQL, resp.Type)

				// Verify MySQL configuration
				require.NotNil(t, resp.MySQL)
				assert.Equal(t, "8.0", resp.MySQL.Version)
				assert.Equal(t, "myapp", resp.MySQL.Database)

				// Verify in Kubernetes
				var application v1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      "application-" + resp.Slug + "-kibaship-com",
					Namespace: "default",
				}, &application)
				require.NoError(t, err)

				assert.Equal(t, v1alpha1.ApplicationTypeMySQL, application.Spec.Type)
				require.NotNil(t, application.Spec.MySQL)
				assert.Equal(t, "8.0", application.Spec.MySQL.Version)
				assert.Equal(t, "myapp", application.Spec.MySQL.Database)

				// Clean up
				err = k8sClient.Delete(ctx, &application)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			jsonData, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Response: %s", w.Body.String())

			if tt.expectedStatus == http.StatusCreated {
				// Parse response
				var response models.ApplicationResponse
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err, "Failed to parse application response: %s", w.Body.String())

				// Run custom validation
				tt.validateFunc(t, &response)
			}
		})
	}

	// Clean up project
	var project v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + createdProject.Slug,
	}, &project)
	if err == nil {
		k8sClient.Delete(ctx, &project)
	}
}

func TestApplicationRetrievalIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// Create a project first
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Retrieval",
		WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	jsonData, err := json.Marshal(projectPayload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdProject models.ProjectResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdProject)
	require.NoError(t, err)

	// Create an application
	appPayload := models.ApplicationCreateRequest{
		Name: "Test App for Retrieval",
		Type: models.ApplicationTypeDockerImage,
		DockerImage: &models.DockerImageConfig{
			Image: "redis:latest",
		},
	}

	jsonData, err = json.Marshal(appPayload)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdApplication models.ApplicationResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdApplication)
	require.NoError(t, err)

	// Test retrieval by slug
	t.Run("Get application by slug", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/application/"+createdApplication.Slug, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var retrievedApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &retrievedApplication)
		require.NoError(t, err)

		// Verify retrieved application matches created application
		assert.Equal(t, createdApplication.UUID, retrievedApplication.UUID)
		assert.Equal(t, createdApplication.Slug, retrievedApplication.Slug)
		assert.Equal(t, createdApplication.Name, retrievedApplication.Name)
		assert.Equal(t, createdApplication.Type, retrievedApplication.Type)
	})

	// Test getting applications by project
	t.Run("Get applications by project", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/projects/"+createdProject.Slug+"/applications", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var applications []models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &applications)
		require.NoError(t, err)

		assert.Len(t, applications, 1)
		assert.Equal(t, createdApplication.UUID, applications[0].UUID)
	})

	// Clean up
	var application v1alpha1.Application
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "application-" + createdApplication.Slug + "-kibaship-com",
		Namespace: "default",
	}, &application)
	if err == nil {
		k8sClient.Delete(ctx, &application)
	}

	var project v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + createdProject.Slug,
	}, &project)
	if err == nil {
		k8sClient.Delete(ctx, &project)
	}
}

func TestApplicationUpdateIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// Create project and application
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Update",
		WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	jsonData, err := json.Marshal(projectPayload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdProject models.ProjectResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdProject)
	require.NoError(t, err)

	// Create application
	appPayload := models.ApplicationCreateRequest{
		Name: "App to Update",
		Type: models.ApplicationTypeDockerImage,
		DockerImage: &models.DockerImageConfig{
			Image: "nginx:1.0",
		},
	}

	jsonData, err = json.Marshal(appPayload)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdApplication models.ApplicationResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdApplication)
	require.NoError(t, err)

	// Test update
	t.Run("Update application name and Docker image", func(t *testing.T) {
		updateReq := models.ApplicationUpdateRequest{
			Name: stringPtr("Updated App Name"),
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:2.0",
				Tag:   "latest",
			},
		}

		jsonData, err := json.Marshal(updateReq)
		require.NoError(t, err)

		req, err := http.NewRequest("PATCH", "/application/"+createdApplication.Slug, bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updatedApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &updatedApplication)
		require.NoError(t, err)

		assert.Equal(t, "Updated App Name", updatedApplication.Name)
		assert.Equal(t, "nginx:2.0", updatedApplication.DockerImage.Image)
		assert.Equal(t, "latest", updatedApplication.DockerImage.Tag)

		// Verify in Kubernetes
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.Slug + "-kibaship-com",
			Namespace: "default",
		}, &application)
		require.NoError(t, err)

		annotations := application.GetAnnotations()
		assert.Equal(t, "Updated App Name", annotations[validation.AnnotationResourceName])
		assert.Equal(t, "nginx:2.0", application.Spec.DockerImage.Image)
		assert.Equal(t, "latest", application.Spec.DockerImage.Tag)
	})

	// Clean up
	var application v1alpha1.Application
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "application-" + createdApplication.Slug + "-kibaship-com",
		Namespace: "default",
	}, &application)
	if err == nil {
		k8sClient.Delete(ctx, &application)
	}

	var project v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + createdProject.Slug,
	}, &project)
	if err == nil {
		k8sClient.Delete(ctx, &project)
	}
}

func TestApplicationDeleteIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// Create project and application
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Delete",
		WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	jsonData, err := json.Marshal(projectPayload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdProject models.ProjectResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdProject)
	require.NoError(t, err)

	// Create application to delete
	appPayload := models.ApplicationCreateRequest{
		Name: "App to Delete",
		Type: models.ApplicationTypeDockerImage,
		DockerImage: &models.DockerImageConfig{
			Image: "nginx:latest",
		},
	}

	jsonData, err = json.Marshal(appPayload)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdApplication models.ApplicationResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdApplication)
	require.NoError(t, err)

	// Verify application exists in Kubernetes
	var application v1alpha1.Application
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "application-" + createdApplication.Slug + "-kibaship-com",
		Namespace: "default",
	}, &application)
	require.NoError(t, err, "Application should exist before deletion")

	// Test deletion
	t.Run("Delete existing application", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/application/"+createdApplication.Slug, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify application no longer exists in Kubernetes
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.Slug + "-kibaship-com",
			Namespace: "default",
		}, &application)
		assert.Error(t, err, "Application should not exist after deletion")
	})

	// Clean up project
	var project v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + createdProject.Slug,
	}, &project)
	if err == nil {
		k8sClient.Delete(ctx, &project)
	}
}
func TestApplicationDomainAutoLoadingIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// Create project first
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Domain Loading",
		WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	jsonData, err := json.Marshal(projectPayload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdProject models.ProjectResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdProject)
	require.NoError(t, err)

	// Create application
	appPayload := models.ApplicationCreateRequest{
		Name: "App with Domains",
		Type: models.ApplicationTypeDockerImage,
		DockerImage: &models.DockerImageConfig{
			Image: "nginx:latest",
		},
	}

	jsonData, err = json.Marshal(appPayload)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdApplication models.ApplicationResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdApplication)
	require.NoError(t, err)

	// Create domains for the application
	domainPayloads := []models.ApplicationDomainCreateRequest{
		{
			ApplicationSlug: createdApplication.Slug,
			Domain:          "app.example.com",
			Port:            3000,
			Type:            models.ApplicationDomainTypeDefault,
			Default:         true,
			TLSEnabled:      true,
		},
		{
			ApplicationSlug: createdApplication.Slug,
			Domain:          "custom.example.com",
			Port:            8080,
			Type:            models.ApplicationDomainTypeCustom,
			Default:         false,
			TLSEnabled:      false,
		},
	}

	var createdDomains []models.ApplicationDomainResponse
	for _, domainPayload := range domainPayloads {
		jsonData, err = json.Marshal(domainPayload)
		require.NoError(t, err)

		req, err = http.NewRequest("POST", "/applications/"+createdApplication.Slug+"/domains", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var createdDomain models.ApplicationDomainResponse
		err = json.Unmarshal(w.Body.Bytes(), &createdDomain)
		require.NoError(t, err)
		createdDomains = append(createdDomains, createdDomain)
	}

	// Test single application retrieval with auto-loaded domains
	t.Run("Get single application with auto-loaded domains", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/application/"+createdApplication.Slug, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var retrievedApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &retrievedApplication)
		require.NoError(t, err)

		// Verify application details
		assert.Equal(t, createdApplication.UUID, retrievedApplication.UUID)
		assert.Equal(t, createdApplication.Slug, retrievedApplication.Slug)
		assert.Equal(t, createdApplication.Name, retrievedApplication.Name)

		// Verify domains are auto-loaded
		assert.Len(t, retrievedApplication.Domains, 2, "Application should have 2 domains auto-loaded")

		// Verify domain details
		domainsByDomain := make(map[string]models.ApplicationDomainResponse)
		for _, domain := range retrievedApplication.Domains {
			domainsByDomain[domain.Domain] = domain
		}

		// Check default domain
		defaultDomain, exists := domainsByDomain["app.example.com"]
		assert.True(t, exists, "Default domain should exist")
		assert.Equal(t, int32(3000), defaultDomain.Port)
		assert.Equal(t, models.ApplicationDomainTypeDefault, defaultDomain.Type)
		assert.True(t, defaultDomain.Default)
		assert.True(t, defaultDomain.TLSEnabled)

		// Check custom domain
		customDomain, exists := domainsByDomain["custom.example.com"]
		assert.True(t, exists, "Custom domain should exist")
		assert.Equal(t, int32(8080), customDomain.Port)
		assert.Equal(t, models.ApplicationDomainTypeCustom, customDomain.Type)
		assert.False(t, customDomain.Default)
		assert.False(t, customDomain.TLSEnabled)
	})

	// Test multiple applications retrieval with batch-loaded domains and deployments
	t.Run("Get applications by project with batch-loaded domains and deployments", func(t *testing.T) {
		// First create a deployment for the application
		deploymentPayload := models.DeploymentCreateRequest{
			ApplicationSlug: createdApplication.Slug,
			GitRepository: &models.GitRepositoryDeploymentConfig{
				CommitSHA: "abc123def456",
				Branch:    "main",
			},
		}

		jsonData, err := json.Marshal(deploymentPayload)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/applications/"+createdApplication.Slug+"/deployments", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var createdDeployment models.DeploymentResponse
		err = json.Unmarshal(w.Body.Bytes(), &createdDeployment)
		require.NoError(t, err)

		// Now test the applications endpoint
		req, err = http.NewRequest("GET", "/projects/"+createdProject.Slug+"/applications", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var applications []models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &applications)
		require.NoError(t, err)

		assert.Len(t, applications, 1, "Should have 1 application")

		// Verify the application has domains and latest deployment batch-loaded
		app := applications[0]
		assert.Equal(t, createdApplication.UUID, app.UUID)
		assert.Len(t, app.Domains, 2, "Application should have 2 domains batch-loaded")

		// Verify domain details are correct
		domainsByDomain := make(map[string]models.ApplicationDomainResponse)
		for _, domain := range app.Domains {
			domainsByDomain[domain.Domain] = domain
		}

		assert.Contains(t, domainsByDomain, "app.example.com")
		assert.Contains(t, domainsByDomain, "custom.example.com")

		// Verify latest deployment is loaded
		require.NotNil(t, app.LatestDeployment, "Application should have latest deployment loaded")
		assert.Equal(t, createdDeployment.UUID, app.LatestDeployment.UUID)
		assert.Equal(t, createdDeployment.Slug, app.LatestDeployment.Slug)
		assert.Equal(t, "abc123def456", app.LatestDeployment.GitRepository.CommitSHA)
		assert.Equal(t, "main", app.LatestDeployment.GitRepository.Branch)

		// Clean up deployment
		var deployment v1alpha1.Deployment
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "deployment-" + createdDeployment.Slug + "-kibaship-com",
			Namespace: "default",
		}, &deployment)
		if err == nil {
			k8sClient.Delete(ctx, &deployment)
		}
	})

	// Clean up domains
	for _, domain := range createdDomains {
		var applicationDomain v1alpha1.ApplicationDomain
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "domain-" + domain.Slug + "-kibaship-com",
			Namespace: "default",
		}, &applicationDomain)
		if err == nil {
			k8sClient.Delete(ctx, &applicationDomain)
		}
	}

	// Clean up application
	var application v1alpha1.Application
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "application-" + createdApplication.Slug + "-kibaship-com",
		Namespace: "default",
	}, &application)
	if err == nil {
		k8sClient.Delete(ctx, &application)
	}

	// Clean up project
	var project v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + createdProject.Slug,
	}, &project)
	if err == nil {
		k8sClient.Delete(ctx, &project)
	}
}
