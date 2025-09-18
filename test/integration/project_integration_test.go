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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/auth"
	"github.com/kibamail/kibaship-operator/pkg/handlers"
	"github.com/kibamail/kibaship-operator/pkg/models"
	"github.com/kibamail/kibaship-operator/pkg/services"
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
	scheme    *runtime.Scheme
)

func TestMain(m *testing.M) {
	// Set up test environment
	setup()

	// Run tests
	code := m.Run()

	// Tear down
	teardown()

	os.Exit(code)
}

func setup() {
	// Initialize scheme
	scheme = runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
	err = v1alpha1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	// Set up envtest environment
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		panic(err)
	}

	// Create Kubernetes client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}
}

func teardown() {
	if testEnv != nil {
		testEnv.Stop()
	}
}

func TestProjectCreationIntegration(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		payload        models.ProjectCreateRequest
		expectedStatus int
		validateFunc   func(*testing.T, *models.ProjectResponse)
	}{
		{
			name: "Create minimal project",
			payload: models.ProjectCreateRequest{
				Name:          "Integration Test Project",
				WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, resp *models.ProjectResponse) {
				// Verify response structure
				assert.NotEmpty(t, resp.UUID)
				assert.NotEmpty(t, resp.Slug)
				assert.Len(t, resp.Slug, 8)
				assert.Equal(t, "Integration Test Project", resp.Name)
				assert.Equal(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8", resp.WorkspaceUUID)
				assert.Equal(t, models.ResourceProfileDevelopment, resp.ResourceProfile)
				assert.Equal(t, "Pending", resp.Status)
				assert.NotZero(t, resp.CreatedAt)

				// Verify Project CRD was actually created in Kubernetes
				var project v1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name: "project-" + resp.Slug,
				}, &project)
				require.NoError(t, err, "Project CRD should exist in Kubernetes")

				// Verify CRD labels
				labels := project.GetLabels()
				assert.Equal(t, resp.UUID, labels[validation.LabelResourceUUID])
				assert.Equal(t, resp.Slug, labels[validation.LabelResourceSlug])
				assert.Equal(t, resp.WorkspaceUUID, labels[validation.LabelWorkspaceUUID])

				// Verify CRD spec has correct defaults
				assert.True(t, project.Spec.ApplicationTypes.MySQL.Enabled)
				assert.True(t, project.Spec.ApplicationTypes.Postgres.Enabled)
				assert.True(t, project.Spec.ApplicationTypes.DockerImage.Enabled)
				assert.True(t, project.Spec.ApplicationTypes.GitRepository.Enabled)
				assert.False(t, project.Spec.ApplicationTypes.MySQLCluster.Enabled)
				assert.False(t, project.Spec.ApplicationTypes.PostgresCluster.Enabled)
				assert.Equal(t, "50Gi", project.Spec.Volumes.MaxStorageSize)

				// Clean up - delete the created project
				err = k8sClient.Delete(ctx, &project)
				assert.NoError(t, err)
			},
		},
		{
			name: "Create production project with custom settings",
			payload: models.ProjectCreateRequest{
				Name:          "Production Project",
				Description:   "A production environment project",
				WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				EnabledApplicationTypes: &models.ApplicationTypeSettings{
					MySQL:           boolPtr(true),
					MySQLCluster:    boolPtr(false),
					Postgres:        boolPtr(false),
					PostgresCluster: boolPtr(false),
					DockerImage:     boolPtr(true),
					GitRepository:   boolPtr(true),
				},
				ResourceProfile: resourceProfilePtr(models.ResourceProfileProduction),
				VolumeSettings: &models.VolumeSettings{
					MaxStorageSize: "200Gi",
				},
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, resp *models.ProjectResponse) {
				// Verify response
				assert.Equal(t, "Production Project", resp.Name)
				assert.Equal(t, models.ResourceProfileProduction, resp.ResourceProfile)
				assert.Equal(t, "200Gi", resp.VolumeSettings.MaxStorageSize)

				// Verify enablement settings
				assert.True(t, *resp.EnabledApplicationTypes.MySQL)
				assert.False(t, *resp.EnabledApplicationTypes.MySQLCluster)
				assert.False(t, *resp.EnabledApplicationTypes.Postgres)
				assert.False(t, *resp.EnabledApplicationTypes.PostgresCluster)
				assert.True(t, *resp.EnabledApplicationTypes.DockerImage)
				assert.True(t, *resp.EnabledApplicationTypes.GitRepository)

				// Verify actual CRD in Kubernetes
				var project v1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name: "project-" + resp.Slug,
				}, &project)
				require.NoError(t, err)

				// Verify application type settings in CRD
				assert.True(t, project.Spec.ApplicationTypes.MySQL.Enabled)
				assert.False(t, project.Spec.ApplicationTypes.MySQLCluster.Enabled)
				assert.False(t, project.Spec.ApplicationTypes.Postgres.Enabled)
				assert.False(t, project.Spec.ApplicationTypes.PostgresCluster.Enabled)
				assert.True(t, project.Spec.ApplicationTypes.DockerImage.Enabled)
				assert.True(t, project.Spec.ApplicationTypes.GitRepository.Enabled)

				// Verify volume settings
				assert.Equal(t, "200Gi", project.Spec.Volumes.MaxStorageSize)

				// Verify production resource defaults were applied
				assert.Equal(t, "2", project.Spec.ApplicationTypes.MySQL.DefaultLimits.CPU)
				assert.Equal(t, "4Gi", project.Spec.ApplicationTypes.MySQL.DefaultLimits.Memory)
				assert.Equal(t, "20Gi", project.Spec.ApplicationTypes.MySQL.DefaultLimits.Storage)

				// Clean up
				err = k8sClient.Delete(ctx, &project)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router with real Kubernetes integration
			apiKey := generateTestAPIKey()
			router := setupIntegrationTestRouter(apiKey)

			// Prepare request
			jsonData, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
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
				var response models.ProjectResponse
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err, "Failed to parse project response: %s", w.Body.String())

				// Run custom validation
				tt.validateFunc(t, &response)
			}
		})
	}
}

func TestProjectRetrievalIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// First, create a project
	payload := models.ProjectCreateRequest{
		Name:          "Test Retrieval Project",
		WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	jsonData, err := json.Marshal(payload)
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

	// Now test retrieval by slug
	t.Run("Get project by slug", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/project/"+createdProject.Slug, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var retrievedProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &retrievedProject)
		require.NoError(t, err)

		// Verify retrieved project matches created project
		assert.Equal(t, createdProject.UUID, retrievedProject.UUID)
		assert.Equal(t, createdProject.Slug, retrievedProject.Slug)
		assert.Equal(t, createdProject.Name, retrievedProject.Name)
		assert.Equal(t, createdProject.WorkspaceUUID, retrievedProject.WorkspaceUUID)
	})

	t.Run("Get non-existent project returns 404", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/projects/notfound", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Clean up - delete the created project
	var project v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + createdProject.Slug,
	}, &project)
	if err == nil {
		k8sClient.Delete(ctx, &project)
	}
}

func TestSlugUniquenessIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// Create multiple projects to test slug uniqueness
	var createdSlugs []string
	const numProjects = 5

	for i := 0; i < numProjects; i++ {
		payload := models.ProjectCreateRequest{
			Name:          fmt.Sprintf("Uniqueness Test Project %d", i+1),
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code, "Response: %s", w.Body.String())

		var response models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify slug is unique
		for _, existingSlug := range createdSlugs {
			assert.NotEqual(t, existingSlug, response.Slug, "Slug should be unique")
		}

		createdSlugs = append(createdSlugs, response.Slug)

		// Verify project exists in Kubernetes
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + response.Slug,
		}, &project)
		require.NoError(t, err, "Project should exist in Kubernetes")
	}

	// Clean up all created projects
	for _, slug := range createdSlugs {
		var project v1alpha1.Project
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + slug,
		}, &project)
		if err == nil {
			k8sClient.Delete(ctx, &project)
		}
	}
}

func setupIntegrationTestRouter(apiKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create authenticator
	authenticator := auth.NewAPIKeyAuthenticator(apiKey)

	// Create real services with Kubernetes client and dependency injection
	projectService := services.NewProjectService(k8sClient, scheme)
	projectHandler := handlers.NewProjectHandler(projectService)
	applicationService := services.NewApplicationService(k8sClient, scheme, projectService)
	deploymentService := services.NewDeploymentService(k8sClient, scheme, applicationService)
	applicationDomainService := services.NewApplicationDomainService(k8sClient, scheme, applicationService)

	// Set circular dependencies for auto-loading
	applicationService.SetDomainService(applicationDomainService)
	applicationService.SetDeploymentService(deploymentService)

	// Create handlers
	applicationHandler := handlers.NewApplicationHandler(applicationService)
	deploymentHandler := handlers.NewDeploymentHandler(deploymentService)
	applicationDomainHandler := handlers.NewApplicationDomainHandler(applicationDomainService)

	// Protected routes
	protected := router.Group("/")
	protected.Use(authenticator.Middleware())
	{
		// Project endpoints
		protected.POST("/projects", projectHandler.CreateProject)
		protected.GET("/project/:slug", projectHandler.GetProject)
		protected.PATCH("/project/:slug", projectHandler.UpdateProject)
		protected.DELETE("/project/:slug", projectHandler.DeleteProject)

		// Application endpoints
		protected.POST("/projects/:projectSlug/applications", applicationHandler.CreateApplication)
		protected.GET("/projects/:projectSlug/applications", applicationHandler.GetApplicationsByProject)
		protected.GET("/application/:slug", applicationHandler.GetApplication)
		protected.PATCH("/application/:slug", applicationHandler.UpdateApplication)
		protected.DELETE("/application/:slug", applicationHandler.DeleteApplication)

		// Deployment endpoints
		protected.POST("/applications/:applicationSlug/deployments", deploymentHandler.CreateDeployment)
		protected.GET("/applications/:applicationSlug/deployments", deploymentHandler.GetDeploymentsByApplication)
		protected.GET("/deployments/:slug", deploymentHandler.GetDeployment)

		// Application Domain endpoints
		protected.POST("/applications/:applicationSlug/domains", applicationDomainHandler.CreateApplicationDomain)
		protected.GET("/domains/:slug", applicationDomainHandler.GetApplicationDomain)
		protected.DELETE("/domains/:slug", applicationDomainHandler.DeleteApplicationDomain)
	}

	return router
}

func generateTestAPIKey() string {
	return "test-api-key-for-integration-tests-12345678901234567890"
}

func boolPtr(b bool) *bool {
	return &b
}

func resourceProfilePtr(profile models.ResourceProfile) *models.ResourceProfile {
	return &profile
}

func TestProjectUpdateIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// First, create a project to update
	payload := models.ProjectCreateRequest{
		Name:            "Project To Update",
		Description:     "Original description",
		WorkspaceUUID:   "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		ResourceProfile: resourceProfilePtr(models.ResourceProfileDevelopment),
	}

	jsonData, err := json.Marshal(payload)
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

	// Test 1: Update project name and description
	t.Run("Update project name and description", func(t *testing.T) {
		updateReq := models.ProjectUpdateRequest{
			Name:        stringPtr("Updated Project Name"),
			Description: stringPtr("Updated description"),
		}

		jsonData, err := json.Marshal(updateReq)
		require.NoError(t, err)

		req, err := http.NewRequest("PATCH", "/project/"+createdProject.Slug, bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updatedProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &updatedProject)
		require.NoError(t, err)

		assert.Equal(t, "Updated Project Name", updatedProject.Name)
		assert.Equal(t, "Updated description", updatedProject.Description)
		assert.Equal(t, createdProject.UUID, updatedProject.UUID)
		assert.Equal(t, createdProject.Slug, updatedProject.Slug)

		// Verify in Kubernetes
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		require.NoError(t, err)

		annotations := project.GetAnnotations()
		assert.Equal(t, "Updated Project Name", annotations[validation.AnnotationResourceName])
		assert.Equal(t, "Updated description", annotations[validation.AnnotationResourceDescription])
	})

	// Test 2: Update resource profile
	t.Run("Update resource profile to production", func(t *testing.T) {
		updateReq := models.ProjectUpdateRequest{
			ResourceProfile: resourceProfilePtr(models.ResourceProfileProduction),
		}

		jsonData, err := json.Marshal(updateReq)
		require.NoError(t, err)

		req, err := http.NewRequest("PATCH", "/project/"+createdProject.Slug, bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updatedProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &updatedProject)
		require.NoError(t, err)

		assert.Equal(t, models.ResourceProfileProduction, updatedProject.ResourceProfile)

		// Verify production defaults were applied in Kubernetes
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		require.NoError(t, err)

		// Production profile should have higher limits
		assert.Equal(t, "2", project.Spec.ApplicationTypes.MySQL.DefaultLimits.CPU)
		assert.Equal(t, "4Gi", project.Spec.ApplicationTypes.MySQL.DefaultLimits.Memory)
	})

	// Test 3: Update non-existent project
	t.Run("Update non-existent project returns 404", func(t *testing.T) {
		updateReq := models.ProjectUpdateRequest{
			Name: stringPtr("Non-existent"),
		}

		jsonData, err := json.Marshal(updateReq)
		require.NoError(t, err)

		req, err := http.NewRequest("PATCH", "/projects/notfound", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Clean up
	var project v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + createdProject.Slug,
	}, &project)
	if err == nil {
		k8sClient.Delete(ctx, &project)
	}
}

func TestProjectDeleteIntegration(t *testing.T) {
	ctx := context.Background()
	apiKey := generateTestAPIKey()
	router := setupIntegrationTestRouter(apiKey)

	// First, create a project to delete
	payload := models.ProjectCreateRequest{
		Name:          "Project To Delete",
		WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}

	jsonData, err := json.Marshal(payload)
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

	// Verify project exists in Kubernetes
	var project v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: "project-" + createdProject.Slug,
	}, &project)
	require.NoError(t, err, "Project should exist before deletion")

	// Test 1: Delete the project
	t.Run("Delete existing project", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/project/"+createdProject.Slug, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify project no longer exists in Kubernetes
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		assert.Error(t, err, "Project should not exist after deletion")
	})

	// Test 2: Try to delete the same project again (should return 404)
	t.Run("Delete non-existent project returns 404", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/project/"+createdProject.Slug, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Test 3: Try to delete with non-existent slug
	t.Run("Delete with non-existent slug returns 404", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/projects/notfound", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func stringPtr(s string) *string {
	return &s
}
