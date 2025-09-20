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
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
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

var _ = BeforeSuite(func() {
	// Initialize scheme
	scheme = runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = v1alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	// Fail fast if envtest assets are not configured
	assets := os.Getenv("KUBEBUILDER_ASSETS")
	Expect(assets).NotTo(BeEmpty(),
		"KUBEBUILDER_ASSETS is not set. Install envtest binaries and export KUBEBUILDER_ASSETS. For example:\n  go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest\n  export KUBEBUILDER_ASSETS=$(setup-envtest use -p path 1.30.x)")

	// Set up envtest environment
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	// Start control plane with a timeout so tests never hang
	var cfg *rest.Config
	cfgCh := make(chan *rest.Config, 1)
	errCh := make(chan error, 1)
	go func() {
		c, e := testEnv.Start()
		if e != nil {
			errCh <- e
			return
		}
		cfgCh <- c
	}()

	select {
	case e := <-errCh:
		Expect(e).NotTo(HaveOccurred())
	case c := <-cfgCh:
		cfg = c
	case <-time.After(30 * time.Second):
		Fail("timed out starting envtest control plane after 30s; verify KUBEBUILDER_ASSETS points to valid kube-apiserver/etcd binaries")
	}

	// Ensure all API requests fail fast rather than hang
	cfg.Timeout = 15 * time.Second

	// Create Kubernetes client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	if testEnv != nil {
		_ = testEnv.Stop()
	}
})

var _ = Describe("Project Creation Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)
	})

	DescribeTable("project creation scenarios",
		func(payload models.ProjectCreateRequest, expectedStatus int, validateFunc func(*models.ProjectResponse)) {
			// Prepare request
			jsonData, err := json.Marshal(payload)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status code
			Expect(w.Code).To(Equal(expectedStatus), "Response: %s", w.Body.String())

			if expectedStatus == http.StatusCreated {
				// Parse response
				var response models.ProjectResponse
				err = json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).NotTo(HaveOccurred(), "Failed to parse project response: %s", w.Body.String())

				// Run custom validation
				validateFunc(&response)
			}
		},
		Entry("Create minimal project",
			models.ProjectCreateRequest{
				Name:          "Integration Test Project",
				WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			},
			http.StatusCreated,
			func(resp *models.ProjectResponse) {
				// Verify response structure
				Expect(resp.UUID).NotTo(BeEmpty())
				Expect(resp.Slug).NotTo(BeEmpty())
				Expect(resp.Slug).To(HaveLen(8))
				Expect(resp.Name).To(Equal("Integration Test Project"))
				Expect(resp.WorkspaceUUID).To(Equal("6ba7b810-9dad-11d1-80b4-00c04fd430c8"))
				Expect(resp.ResourceProfile).To(Equal(models.ResourceProfileDevelopment))
				Expect(resp.Status).To(Equal("Pending"))
				Expect(resp.CreatedAt).NotTo(BeZero())

				// Verify Project CRD was actually created in Kubernetes
				var project v1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name: "project-" + resp.Slug,
				}, &project)
				Expect(err).NotTo(HaveOccurred(), "Project CRD should exist in Kubernetes")

				// Verify CRD labels
				labels := project.GetLabels()
				Expect(labels[validation.LabelResourceUUID]).To(Equal(resp.UUID))
				Expect(labels[validation.LabelResourceSlug]).To(Equal(resp.Slug))
				Expect(labels[validation.LabelWorkspaceUUID]).To(Equal(resp.WorkspaceUUID))

				// Verify CRD spec has correct defaults
				Expect(project.Spec.ApplicationTypes.MySQL.Enabled).To(BeTrue())
				Expect(project.Spec.ApplicationTypes.Postgres.Enabled).To(BeTrue())
				Expect(project.Spec.ApplicationTypes.DockerImage.Enabled).To(BeTrue())
				Expect(project.Spec.ApplicationTypes.GitRepository.Enabled).To(BeTrue())
				Expect(project.Spec.ApplicationTypes.MySQLCluster.Enabled).To(BeFalse())
				Expect(project.Spec.ApplicationTypes.PostgresCluster.Enabled).To(BeFalse())
				Expect(project.Spec.Volumes.MaxStorageSize).To(Equal("50Gi"))

				// Clean up - delete the created project
				err = k8sClient.Delete(ctx, &project)
				Expect(err).NotTo(HaveOccurred())
			},
		),
		Entry("Create production project with custom settings",
			models.ProjectCreateRequest{
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
			http.StatusCreated,
			func(resp *models.ProjectResponse) {
				// Verify response
				Expect(resp.Name).To(Equal("Production Project"))
				Expect(resp.ResourceProfile).To(Equal(models.ResourceProfileProduction))
				Expect(resp.VolumeSettings.MaxStorageSize).To(Equal("200Gi"))

				// Verify enablement settings
				Expect(*resp.EnabledApplicationTypes.MySQL).To(BeTrue())
				Expect(*resp.EnabledApplicationTypes.MySQLCluster).To(BeFalse())
				Expect(*resp.EnabledApplicationTypes.Postgres).To(BeFalse())
				Expect(*resp.EnabledApplicationTypes.PostgresCluster).To(BeFalse())
				Expect(*resp.EnabledApplicationTypes.DockerImage).To(BeTrue())
				Expect(*resp.EnabledApplicationTypes.GitRepository).To(BeTrue())

				// Verify actual CRD in Kubernetes
				var project v1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name: "project-" + resp.Slug,
				}, &project)
				Expect(err).NotTo(HaveOccurred())

				// Verify application type settings in CRD
				Expect(project.Spec.ApplicationTypes.MySQL.Enabled).To(BeTrue())
				Expect(project.Spec.ApplicationTypes.MySQLCluster.Enabled).To(BeFalse())
				Expect(project.Spec.ApplicationTypes.Postgres.Enabled).To(BeFalse())
				Expect(project.Spec.ApplicationTypes.PostgresCluster.Enabled).To(BeFalse())
				Expect(project.Spec.ApplicationTypes.DockerImage.Enabled).To(BeTrue())
				Expect(project.Spec.ApplicationTypes.GitRepository.Enabled).To(BeTrue())

				// Verify volume settings
				Expect(project.Spec.Volumes.MaxStorageSize).To(Equal("200Gi"))

				// Verify production resource defaults were applied
				Expect(project.Spec.ApplicationTypes.MySQL.DefaultLimits.CPU).To(Equal("2"))
				Expect(project.Spec.ApplicationTypes.MySQL.DefaultLimits.Memory).To(Equal("4Gi"))
				Expect(project.Spec.ApplicationTypes.MySQL.DefaultLimits.Storage).To(Equal("20Gi"))

				// Clean up
				err = k8sClient.Delete(ctx, &project)
				Expect(err).NotTo(HaveOccurred())
			},
		),
	)
})

var _ = Describe("Project Retrieval Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdProject *models.ProjectResponse

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)

		// Create a project for retrieval tests
		payload := models.ProjectCreateRequest{
			Name:          "Test Retrieval Project",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var project models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &project)
		Expect(err).NotTo(HaveOccurred())
		createdProject = &project
	})

	AfterEach(func() {
		// Clean up - delete the created project
		var project v1alpha1.Project
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("retrieves project by slug", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		req, err := http.NewRequest("GET", "/project/"+createdProject.Slug, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var retrievedProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &retrievedProject)
		Expect(err).NotTo(HaveOccurred())

		// Verify retrieved project matches created project
		Expect(retrievedProject.UUID).To(Equal(createdProject.UUID))
		Expect(retrievedProject.Slug).To(Equal(createdProject.Slug))
		Expect(retrievedProject.Name).To(Equal(createdProject.Name))
		Expect(retrievedProject.WorkspaceUUID).To(Equal(createdProject.WorkspaceUUID))
	})

	It("returns 404 for non-existent project", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		req, err := http.NewRequest("GET", "/projects/notfound", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})

var _ = Describe("Project Slug Uniqueness Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdSlugs []string

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)
		createdSlugs = []string{}
	})

	AfterEach(func() {
		// Clean up all created projects
		for _, slug := range createdSlugs {
			var project v1alpha1.Project
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name: "project-" + slug,
			}, &project)
			if err == nil {
				_ = k8sClient.Delete(ctx, &project)
			}
		}
	})

	It("generates unique slugs for multiple projects", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		const numProjects = 5

		for i := 0; i < numProjects; i++ {
			payload := models.ProjectCreateRequest{
				Name:          fmt.Sprintf("Uniqueness Test Project %d", i+1),
				WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			}

			jsonData, err := json.Marshal(payload)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusCreated), "Response: %s", w.Body.String())

			var response models.ProjectResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			// Verify slug is unique
			for _, existingSlug := range createdSlugs {
				Expect(response.Slug).NotTo(Equal(existingSlug), "Slug should be unique")
			}

			createdSlugs = append(createdSlugs, response.Slug)

			// Verify project exists in Kubernetes
			var project v1alpha1.Project
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name: "project-" + response.Slug,
			}, &project)
			Expect(err).NotTo(HaveOccurred(), "Project should exist in Kubernetes")
		}
	})
})

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

	// NOTE: Do NOT wire circular dependencies in integration tests.
	// We intentionally avoid auto-loading domains/deployments to prevent
	// recursive calls (ApplicationService <-> ApplicationDomainService)
	// and to keep tests focused on API validation + CRD insertion only.

	// Wire only after breaking circular dependency in services: safe auto-loading in tests
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

var _ = Describe("Project Update Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdProject *models.ProjectResponse

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)

		// Create a project to update
		payload := models.ProjectCreateRequest{
			Name:            "Project To Update",
			Description:     "Original description",
			WorkspaceUUID:   "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			ResourceProfile: resourceProfilePtr(models.ResourceProfileDevelopment),
		}

		jsonData, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var project models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &project)
		Expect(err).NotTo(HaveOccurred())
		createdProject = &project
	})

	AfterEach(func() {
		// Clean up
		var project v1alpha1.Project
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("updates project name and description", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		updateReq := models.ProjectUpdateRequest{
			Name:        stringPtr("Updated Project Name"),
			Description: stringPtr("Updated description"),
		}

		jsonData, err := json.Marshal(updateReq)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("PATCH", "/project/"+createdProject.Slug, bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var updatedProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &updatedProject)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedProject.Name).To(Equal("Updated Project Name"))
		Expect(updatedProject.Description).To(Equal("Updated description"))
		Expect(updatedProject.UUID).To(Equal(createdProject.UUID))
		Expect(updatedProject.Slug).To(Equal(createdProject.Slug))

		// Verify in Kubernetes
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		Expect(err).NotTo(HaveOccurred())

		annotations := project.GetAnnotations()
		Expect(annotations[validation.AnnotationResourceName]).To(Equal("Updated Project Name"))
		Expect(annotations[validation.AnnotationResourceDescription]).To(Equal("Updated description"))
	})

	It("updates resource profile to production", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		updateReq := models.ProjectUpdateRequest{
			ResourceProfile: resourceProfilePtr(models.ResourceProfileProduction),
		}

		jsonData, err := json.Marshal(updateReq)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("PATCH", "/project/"+createdProject.Slug, bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var updatedProject models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &updatedProject)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedProject.ResourceProfile).To(Equal(models.ResourceProfileProduction))

		// Verify production defaults were applied in Kubernetes
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		Expect(err).NotTo(HaveOccurred())

		// Production profile should have higher limits
		Expect(project.Spec.ApplicationTypes.MySQL.DefaultLimits.CPU).To(Equal("2"))
		Expect(project.Spec.ApplicationTypes.MySQL.DefaultLimits.Memory).To(Equal("4Gi"))
	})

	It("returns 404 for non-existent project update", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		updateReq := models.ProjectUpdateRequest{
			Name: stringPtr("Non-existent"),
		}

		jsonData, err := json.Marshal(updateReq)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("PATCH", "/projects/notfound", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})

var _ = Describe("Project Delete Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdProject *models.ProjectResponse

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)

		// Create a project to delete
		payload := models.ProjectCreateRequest{
			Name:          "Project To Delete",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var project models.ProjectResponse
		err = json.Unmarshal(w.Body.Bytes(), &project)
		Expect(err).NotTo(HaveOccurred())
		createdProject = &project

		// Verify project exists in Kubernetes
		var k8sProject v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &k8sProject)
		Expect(err).NotTo(HaveOccurred(), "Project should exist before deletion")
	})

	It("deletes existing project", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		req, err := http.NewRequest("DELETE", "/project/"+createdProject.Slug, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNoContent))

		// Verify project no longer exists in Kubernetes
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		Expect(err).To(HaveOccurred(), "Project should not exist after deletion")
	})

	It("returns 404 when deleting same project again", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		// First delete the project
		req, err := http.NewRequest("DELETE", "/project/"+createdProject.Slug, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNoContent))

		// Try to delete it again
		req, err = http.NewRequest("DELETE", "/project/"+createdProject.Slug, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 404 for non-existent project slug", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		req, err := http.NewRequest("DELETE", "/projects/notfound", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})

func stringPtr(s string) *string {
	return &s
}
