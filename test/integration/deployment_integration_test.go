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

func TestDeploymentCreationIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// First, create a project
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Deployments",
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

	// Create a GitRepository application first
	applicationPayload := models.ApplicationCreateRequest{
		Name:        "My Git App",
		ProjectSlug: createdProject.Slug,
		Type:        models.ApplicationTypeGitRepository,
		GitRepository: &models.GitRepositoryConfig{
			Provider:     models.GitProviderGitHub,
			Repository:   "myorg/myapp",
			Branch:       "main",
			PublicAccess: true,
		},
	}

	jsonData, err = json.Marshal(applicationPayload)
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

	tests := []struct {
		name           string
		payload        models.DeploymentCreateRequest
		expectedStatus int
		validateFunc   func(*testing.T, *models.DeploymentResponse)
	}{
		{
			name: "Create GitRepository deployment",
			payload: models.DeploymentCreateRequest{
				ApplicationSlug: createdApplication.Slug,
				GitRepository: &models.GitRepositoryDeploymentConfig{
					CommitSHA: "abc123def456",
					Branch:    "main",
				},
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, resp *models.DeploymentResponse) {
				assert.NotEmpty(t, resp.UUID)
				assert.NotEmpty(t, resp.Slug)
				assert.Len(t, resp.Slug, 8)
				assert.Equal(t, createdApplication.Slug, resp.ApplicationSlug)
				assert.Equal(t, models.DeploymentPhaseInitializing, resp.Phase)
				require.NotNil(t, resp.GitRepository)
				assert.Equal(t, "abc123def456", resp.GitRepository.CommitSHA)
				assert.Equal(t, "main", resp.GitRepository.Branch)

				// Verify Deployment CRD was actually created in Kubernetes
				var deployment v1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      "deployment-" + resp.Slug + "-kibaship-com",
					Namespace: "default",
				}, &deployment)
				require.NoError(t, err, "Deployment CRD should exist in Kubernetes")

				// Verify CRD labels
				labels := deployment.GetLabels()
				assert.Equal(t, resp.UUID, labels[validation.LabelResourceUUID])
				assert.Equal(t, resp.Slug, labels[validation.LabelResourceSlug])
				assert.Equal(t, resp.ProjectUUID, labels[validation.LabelProjectUUID])
				assert.Equal(t, resp.ApplicationUUID, labels[validation.LabelApplicationUUID])

				// Verify GitRepository config
				require.NotNil(t, deployment.Spec.GitRepository)
				assert.Equal(t, "abc123def456", deployment.Spec.GitRepository.CommitSHA)
				assert.Equal(t, "main", deployment.Spec.GitRepository.Branch)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			jsonData, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/applications/"+createdApplication.Slug+"/deployments", bytes.NewBuffer(jsonData))
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
				var response models.DeploymentResponse
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				if tt.validateFunc != nil {
					tt.validateFunc(t, &response)
				}

				// Clean up the created deployment
				var deployment v1alpha1.Deployment
				err = k8sClient.Get(ctx, client.ObjectKey{
					Name:      "deployment-" + response.Slug + "-kibaship-com",
					Namespace: "default",
				}, &deployment)
				if err == nil {
					k8sClient.Delete(ctx, &deployment)
				}
			}
		})
	}

	// Clean up created application and project
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
		Name:      "project-" + createdProject.Slug + "-kibaship-com",
		Namespace: "default",
	}, &project)
	if err == nil {
		k8sClient.Delete(ctx, &project)
	}
}

func TestDeploymentRetrievalIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// Create project, application and deployment first
	createdProject, createdApplication, createdDeployment := createTestDeployment(t, router, apiKey)

	// Test retrieval by slug
	t.Run("Get deployment by slug", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/deployments/"+createdDeployment.Slug, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var response models.DeploymentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, createdDeployment.UUID, response.UUID)
		assert.Equal(t, createdDeployment.Slug, response.Slug)
		assert.Equal(t, createdApplication.UUID, response.ApplicationUUID)
	})

	// Test getting deployments by application
	t.Run("Get deployments by application", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/applications/"+createdApplication.Slug+"/deployments", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.DeploymentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response, 1)
		assert.Equal(t, createdDeployment.UUID, response[0].UUID)
	})

	// Clean up
	cleanupTestDeployment(ctx, createdProject, createdApplication, createdDeployment)
}

// Helper function to create a test deployment
func createTestDeployment(t *testing.T, router http.Handler, apiKey string) (*models.ProjectResponse, *models.ApplicationResponse, *models.DeploymentResponse) {
	// Create project
	projectPayload := models.ProjectCreateRequest{
		Name:          "Test Project for Deployment Retrieval",
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
	applicationPayload := models.ApplicationCreateRequest{
		Name:        "Test Git App",
		ProjectSlug: createdProject.Slug,
		Type:        models.ApplicationTypeGitRepository,
		GitRepository: &models.GitRepositoryConfig{
			Provider:     models.GitProviderGitHub,
			Repository:   "test/repo",
			Branch:       "main",
			PublicAccess: true,
		},
	}

	jsonData, err = json.Marshal(applicationPayload)
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

	// Create deployment
	deploymentPayload := models.DeploymentCreateRequest{
		ApplicationSlug: createdApplication.Slug,
		GitRepository: &models.GitRepositoryDeploymentConfig{
			CommitSHA: "test123commit",
			Branch:    "develop",
		},
	}

	jsonData, err = json.Marshal(deploymentPayload)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", "/applications/"+createdApplication.Slug+"/deployments", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createdDeployment models.DeploymentResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdDeployment)
	require.NoError(t, err)

	return &createdProject, &createdApplication, &createdDeployment
}

// Helper function to clean up test resources
func cleanupTestDeployment(ctx context.Context, project *models.ProjectResponse, application *models.ApplicationResponse, deployment *models.DeploymentResponse) {
	// Clean up deployment
	var dep v1alpha1.Deployment
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      "deployment-" + deployment.Slug + "-kibaship-com",
		Namespace: "default",
	}, &dep)
	if err == nil {
		k8sClient.Delete(ctx, &dep)
	}

	// Clean up application
	var app v1alpha1.Application
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "application-" + application.Slug + "-kibaship-com",
		Namespace: "default",
	}, &app)
	if err == nil {
		k8sClient.Delete(ctx, &app)
	}

	// Clean up project
	var proj v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "project-" + project.Slug + "-kibaship-com",
		Namespace: "default",
	}, &proj)
	if err == nil {
		k8sClient.Delete(ctx, &proj)
	}
}
