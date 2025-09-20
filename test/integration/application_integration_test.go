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

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/models"
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

var _ = Describe("Application Creation Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdProject *models.ProjectResponse

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)

		// Create a project for applications
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Apps",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(projectPayload)
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
		// Clean up project
		var project v1alpha1.Project
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	DescribeTable("application creation scenarios",
		func(payload models.ApplicationCreateRequest, expectedStatus int, validateFunc func(*models.ApplicationResponse)) {
			// Prepare request
			jsonData, err := json.Marshal(payload)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
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
				var response models.ApplicationResponse
				err = json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).NotTo(HaveOccurred(), "Failed to parse application response: %s", w.Body.String())

				// Run custom validation
				validateFunc(&response)
			}
		},
		Entry("Create DockerImage application",
			models.ApplicationCreateRequest{
				Name: "My Web App",
				Type: models.ApplicationTypeDockerImage,
				DockerImage: &models.DockerImageConfig{
					Image: "nginx:latest",
					Tag:   "v1.0.0",
				},
			},
			http.StatusCreated,
			func(resp *models.ApplicationResponse) {
				// Verify response structure
				Expect(resp.UUID).NotTo(BeEmpty())
				Expect(resp.Slug).NotTo(BeEmpty())
				Expect(resp.Slug).To(HaveLen(8))
				Expect(resp.Name).To(Equal("My Web App"))
				Expect(resp.ProjectUUID).To(Equal(createdProject.UUID))
				Expect(resp.ProjectSlug).To(Equal(createdProject.Slug))
				Expect(resp.Type).To(Equal(models.ApplicationTypeDockerImage))
				Expect(resp.Status).To(Equal("Pending"))
				Expect(resp.CreatedAt).NotTo(BeZero())

				// Verify DockerImage configuration
				Expect(resp.DockerImage).NotTo(BeNil())
				Expect(resp.DockerImage.Image).To(Equal("nginx:latest"))
				Expect(resp.DockerImage.Tag).To(Equal("v1.0.0"))

				// Verify Application CRD was actually created in Kubernetes
				var application v1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      "application-" + resp.Slug + "-kibaship-com",
					Namespace: "default",
				}, &application)
				Expect(err).NotTo(HaveOccurred(), "Application CRD should exist in Kubernetes")

				// Verify CRD labels
				labels := application.GetLabels()
				Expect(labels[validation.LabelResourceUUID]).To(Equal(resp.UUID))
				Expect(labels[validation.LabelResourceSlug]).To(Equal(resp.Slug))
				Expect(labels[validation.LabelProjectUUID]).To(Equal(resp.ProjectUUID))

				// Verify CRD annotations
				annotations := application.GetAnnotations()
				Expect(annotations[validation.AnnotationResourceName]).To(Equal(resp.Name))

				// Verify CRD spec
				Expect(application.Spec.Type).To(Equal(v1alpha1.ApplicationTypeDockerImage))
				Expect(application.Spec.DockerImage).NotTo(BeNil())
				Expect(application.Spec.DockerImage.Image).To(Equal("nginx:latest"))
				Expect(application.Spec.DockerImage.Tag).To(Equal("v1.0.0"))

				// Clean up - delete the created application
				err = k8sClient.Delete(ctx, &application)
				Expect(err).NotTo(HaveOccurred())
			},
		),
		Entry("Create GitRepository application",
			models.ApplicationCreateRequest{
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
			http.StatusCreated,
			func(resp *models.ApplicationResponse) {
				Expect(resp.Name).To(Equal("My Node App"))
				Expect(resp.Type).To(Equal(models.ApplicationTypeGitRepository))

				// Verify GitRepository configuration
				Expect(resp.GitRepository).NotTo(BeNil())
				Expect(resp.GitRepository.Provider).To(Equal(models.GitProviderGitHub))
				Expect(resp.GitRepository.Repository).To(Equal("myorg/myapp"))
				Expect(resp.GitRepository.PublicAccess).To(BeTrue())
				Expect(resp.GitRepository.Branch).To(Equal("main"))
				Expect(resp.GitRepository.BuildCommand).To(Equal("npm run build"))
				Expect(resp.GitRepository.StartCommand).To(Equal("npm start"))

				// Verify in Kubernetes
				var application v1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      "application-" + resp.Slug + "-kibaship-com",
					Namespace: "default",
				}, &application)
				Expect(err).NotTo(HaveOccurred())

				Expect(application.Spec.Type).To(Equal(v1alpha1.ApplicationTypeGitRepository))
				Expect(application.Spec.GitRepository).NotTo(BeNil())
				Expect(application.Spec.GitRepository.Provider).To(Equal(v1alpha1.GitProviderGitHub))
				Expect(application.Spec.GitRepository.Repository).To(Equal("myorg/myapp"))

				// Clean up
				err = k8sClient.Delete(ctx, &application)
				Expect(err).NotTo(HaveOccurred())
			},
		),
		Entry("Create MySQL application",
			models.ApplicationCreateRequest{
				Name: "My Database",
				Type: models.ApplicationTypeMySQL,
				MySQL: &models.MySQLConfig{
					Version:  "8.0",
					Database: "myapp",
				},
			},
			http.StatusCreated,
			func(resp *models.ApplicationResponse) {
				Expect(resp.Name).To(Equal("My Database"))
				Expect(resp.Type).To(Equal(models.ApplicationTypeMySQL))

				// Verify MySQL configuration
				Expect(resp.MySQL).NotTo(BeNil())
				Expect(resp.MySQL.Version).To(Equal("8.0"))
				Expect(resp.MySQL.Database).To(Equal("myapp"))

				// Verify in Kubernetes
				var application v1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      "application-" + resp.Slug + "-kibaship-com",
					Namespace: "default",
				}, &application)
				Expect(err).NotTo(HaveOccurred())

				Expect(application.Spec.Type).To(Equal(v1alpha1.ApplicationTypeMySQL))
				Expect(application.Spec.MySQL).NotTo(BeNil())
				Expect(application.Spec.MySQL.Version).To(Equal("8.0"))
				Expect(application.Spec.MySQL.Database).To(Equal("myapp"))

				// Clean up
				err = k8sClient.Delete(ctx, &application)
				Expect(err).NotTo(HaveOccurred())
			},
		),
	)
})

var _ = Describe("Application Retrieval Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdProject *models.ProjectResponse
	var createdApplication *models.ApplicationResponse

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)

		// Create a project first
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Retrieval",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(projectPayload)
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

		// Create an application
		appPayload := models.ApplicationCreateRequest{
			Name: "Test App for Retrieval",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "redis:latest",
			},
		}

		jsonData, err = json.Marshal(appPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var application models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &application)
		Expect(err).NotTo(HaveOccurred())
		createdApplication = &application
	})

	AfterEach(func() {
		// Clean up application
		var application v1alpha1.Application
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.Slug + "-kibaship-com",
			Namespace: "default",
		}, &application)
		if err == nil {
			_ = k8sClient.Delete(ctx, &application)
		}

		// Clean up project
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("retrieves application by slug", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		req, err := http.NewRequest("GET", "/application/"+createdApplication.Slug, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var retrievedApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &retrievedApplication)
		Expect(err).NotTo(HaveOccurred())

		// Verify retrieved application matches created application
		Expect(retrievedApplication.UUID).To(Equal(createdApplication.UUID))
		Expect(retrievedApplication.Slug).To(Equal(createdApplication.Slug))
		Expect(retrievedApplication.Name).To(Equal(createdApplication.Name))
		Expect(retrievedApplication.Type).To(Equal(createdApplication.Type))
	})

	It("retrieves applications by project", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		req, err := http.NewRequest("GET", "/projects/"+createdProject.Slug+"/applications", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var applications []models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &applications)
		Expect(err).NotTo(HaveOccurred())

		Expect(applications).To(HaveLen(1))
		Expect(applications[0].UUID).To(Equal(createdApplication.UUID))
	})
})

var _ = Describe("Application Update Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdProject *models.ProjectResponse
	var createdApplication *models.ApplicationResponse

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)

		// Create project and application
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Update",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(projectPayload)
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

		// Create application
		appPayload := models.ApplicationCreateRequest{
			Name: "App to Update",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:1.0",
			},
		}

		jsonData, err = json.Marshal(appPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var application models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &application)
		Expect(err).NotTo(HaveOccurred())
		createdApplication = &application
	})

	AfterEach(func() {
		// Clean up application
		var application v1alpha1.Application
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.Slug + "-kibaship-com",
			Namespace: "default",
		}, &application)
		if err == nil {
			_ = k8sClient.Delete(ctx, &application)
		}

		// Clean up project
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("updates application name and Docker image", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		updateReq := models.ApplicationUpdateRequest{
			Name: stringPtr("Updated App Name"),
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:2.0",
				Tag:   "latest",
			},
		}

		jsonData, err := json.Marshal(updateReq)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("PATCH", "/application/"+createdApplication.Slug, bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var updatedApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &updatedApplication)
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedApplication.Name).To(Equal("Updated App Name"))
		Expect(updatedApplication.DockerImage.Image).To(Equal("nginx:2.0"))
		Expect(updatedApplication.DockerImage.Tag).To(Equal("latest"))

		// Verify in Kubernetes
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.Slug + "-kibaship-com",
			Namespace: "default",
		}, &application)
		Expect(err).NotTo(HaveOccurred())

		annotations := application.GetAnnotations()
		Expect(annotations[validation.AnnotationResourceName]).To(Equal("Updated App Name"))
		Expect(application.Spec.DockerImage.Image).To(Equal("nginx:2.0"))
		Expect(application.Spec.DockerImage.Tag).To(Equal("latest"))
	})
})

var _ = Describe("Application Delete Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdProject *models.ProjectResponse
	var createdApplication *models.ApplicationResponse

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)

		// Create project and application
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Delete",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(projectPayload)
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

		// Create application to delete
		appPayload := models.ApplicationCreateRequest{
			Name: "App to Delete",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:latest",
			},
		}

		jsonData, err = json.Marshal(appPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var application models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &application)
		Expect(err).NotTo(HaveOccurred())
		createdApplication = &application

		// Verify application exists in Kubernetes
		var k8sApplication v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.Slug + "-kibaship-com",
			Namespace: "default",
		}, &k8sApplication)
		Expect(err).NotTo(HaveOccurred(), "Application should exist before deletion")
	})

	AfterEach(func() {
		// Clean up project
		var project v1alpha1.Project
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("deletes existing application", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		req, err := http.NewRequest("DELETE", "/application/"+createdApplication.Slug, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusNoContent))

		// Verify application no longer exists in Kubernetes
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.Slug + "-kibaship-com",
			Namespace: "default",
		}, &application)
		Expect(err).To(HaveOccurred(), "Application should not exist after deletion")
	})
})

var _ = Describe("Application Domain Auto Loading Integration", func() {
	var ctx context.Context
	var apiKey string
	var router *gin.Engine
	var createdProject *models.ProjectResponse
	var createdApplication *models.ApplicationResponse
	var createdDomains []models.ApplicationDomainResponse

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)

		// Create project first
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Domain Loading",
			WorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}

		jsonData, err := json.Marshal(projectPayload)
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

		// Create application
		appPayload := models.ApplicationCreateRequest{
			Name: "App with Domains",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:latest",
			},
		}

		jsonData, err = json.Marshal(appPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var application models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &application)
		Expect(err).NotTo(HaveOccurred())
		createdApplication = &application

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

		createdDomains = []models.ApplicationDomainResponse{}
		for _, domainPayload := range domainPayloads {
			jsonData, err = json.Marshal(domainPayload)
			Expect(err).NotTo(HaveOccurred())

			req, err = http.NewRequest("POST", "/applications/"+createdApplication.Slug+"/domains", bytes.NewBuffer(jsonData))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusCreated))

			var createdDomain models.ApplicationDomainResponse
			err = json.Unmarshal(w.Body.Bytes(), &createdDomain)
			Expect(err).NotTo(HaveOccurred())
			createdDomains = append(createdDomains, createdDomain)
		}
	})

	AfterEach(func() {
		// Clean up domains
		for _, domain := range createdDomains {
			var applicationDomain v1alpha1.ApplicationDomain
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      "domain-" + domain.Slug + "-kibaship-com",
				Namespace: "default",
			}, &applicationDomain)
			if err == nil {
				_ = k8sClient.Delete(ctx, &applicationDomain)
			}
		}

		// Clean up application
		var application v1alpha1.Application
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.Slug + "-kibaship-com",
			Namespace: "default",
		}, &application)
		if err == nil {
			_ = k8sClient.Delete(ctx, &application)
		}

		// Clean up project
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name: "project-" + createdProject.Slug,
		}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("retrieves single application with auto-loaded domains", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		req, err := http.NewRequest("GET", "/application/"+createdApplication.Slug, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var retrievedApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &retrievedApplication)
		Expect(err).NotTo(HaveOccurred())

		// Verify application details
		Expect(retrievedApplication.UUID).To(Equal(createdApplication.UUID))
		Expect(retrievedApplication.Slug).To(Equal(createdApplication.Slug))
		Expect(retrievedApplication.Name).To(Equal(createdApplication.Name))

		// Verify domains are auto-loaded
		Expect(retrievedApplication.Domains).To(HaveLen(2), "Application should have 2 domains auto-loaded")

		// Verify domain details
		domainsByDomain := make(map[string]models.ApplicationDomainResponse)
		for _, domain := range retrievedApplication.Domains {
			domainsByDomain[domain.Domain] = domain
		}

		// Check default domain
		defaultDomain, exists := domainsByDomain["app.example.com"]
		Expect(exists).To(BeTrue(), "Default domain should exist")
		Expect(defaultDomain.Port).To(Equal(int32(3000)))
		Expect(defaultDomain.Type).To(Equal(models.ApplicationDomainTypeDefault))
		Expect(defaultDomain.Default).To(BeTrue())
		Expect(defaultDomain.TLSEnabled).To(BeTrue())

		// Check custom domain
		customDomain, exists := domainsByDomain["custom.example.com"]
		Expect(exists).To(BeTrue(), "Custom domain should exist")
		Expect(customDomain.Port).To(Equal(int32(8080)))
		Expect(customDomain.Type).To(Equal(models.ApplicationDomainTypeCustom))
		Expect(customDomain.Default).To(BeFalse())
		Expect(customDomain.TLSEnabled).To(BeFalse())
	})

	It("retrieves applications by project with batch-loaded domains and deployments", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		// First create a deployment for the application
		deploymentPayload := models.DeploymentCreateRequest{
			ApplicationSlug: createdApplication.Slug,
			GitRepository: &models.GitRepositoryDeploymentConfig{
				CommitSHA: "abc123def456",
				Branch:    "main",
			},
		}

		jsonData, err := json.Marshal(deploymentPayload)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest("POST", "/applications/"+createdApplication.Slug+"/deployments", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var createdDeployment models.DeploymentResponse
		err = json.Unmarshal(w.Body.Bytes(), &createdDeployment)
		Expect(err).NotTo(HaveOccurred())

		// Now test the applications endpoint
		req, err = http.NewRequest("GET", "/projects/"+createdProject.Slug+"/applications", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))

		var applications []models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &applications)
		Expect(err).NotTo(HaveOccurred())

		Expect(applications).To(HaveLen(1), "Should have 1 application")

		// Verify the application has domains and latest deployment batch-loaded
		app := applications[0]
		Expect(app.UUID).To(Equal(createdApplication.UUID))
		Expect(app.Domains).To(HaveLen(2), "Application should have 2 domains batch-loaded")

		// Verify domain details are correct
		domainsByDomain := make(map[string]models.ApplicationDomainResponse)
		for _, domain := range app.Domains {
			domainsByDomain[domain.Domain] = domain
		}

		Expect(domainsByDomain).To(HaveKey("app.example.com"))
		Expect(domainsByDomain).To(HaveKey("custom.example.com"))

		// Verify latest deployment is loaded
		Expect(app.LatestDeployment).NotTo(BeNil(), "Application should have latest deployment loaded")
		Expect(app.LatestDeployment.UUID).To(Equal(createdDeployment.UUID))
		Expect(app.LatestDeployment.Slug).To(Equal(createdDeployment.Slug))
		Expect(app.LatestDeployment.GitRepository.CommitSHA).To(Equal("abc123def456"))
		Expect(app.LatestDeployment.GitRepository.Branch).To(Equal("main"))

		// Clean up deployment
		var deployment v1alpha1.Deployment
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "deployment-" + createdDeployment.Slug + "-kibaship-com",
			Namespace: "default",
		}, &deployment)
		if err == nil {
			_ = k8sClient.Delete(ctx, &deployment)
		}
	})
})
