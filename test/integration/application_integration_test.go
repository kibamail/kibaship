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

var _ = Describe("Application Integration", func() {
	It("creates DockerImage application successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Docker App",
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

		// Create production environment via API
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

		// Create application
		payload := models.ApplicationCreateRequest{
			Name: "My Web App",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:latest",
				Tag:   "v1.0.0",
			},
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/environments/"+createdEnvironment.UUID+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated), "Response: %s", w.Body.String())

		var response models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		// Verify response
		Expect(response.UUID).NotTo(BeEmpty())
		Expect(response.Slug).NotTo(BeEmpty())
		Expect(response.Slug).To(HaveLen(8))
		Expect(response.Name).To(Equal("My Web App"))
		Expect(response.ProjectUUID).To(Equal(createdProject.UUID))
		Expect(response.ProjectSlug).To(Equal(createdProject.Slug))
		Expect(response.Type).To(Equal(models.ApplicationTypeDockerImage))
		Expect(response.Status).To(Equal("Pending"))
		Expect(response.DockerImage).NotTo(BeNil())
		Expect(response.DockerImage.Image).To(Equal("nginx:latest"))
		Expect(response.DockerImage.Tag).To(Equal("v1.0.0"))

		// Verify CRD
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + response.UUID + "",
			Namespace: "default",
		}, &application)
		Expect(err).NotTo(HaveOccurred())

		labels := application.GetLabels()
		Expect(labels[validation.LabelResourceUUID]).To(Equal(response.UUID))
		Expect(labels[validation.LabelResourceSlug]).To(Equal(response.Slug))
		Expect(labels[validation.LabelProjectUUID]).To(Equal(response.ProjectUUID))

		annotations := application.GetAnnotations()
		Expect(annotations[validation.AnnotationResourceName]).To(Equal(response.Name))

		Expect(application.Spec.Type).To(Equal(v1alpha1.ApplicationTypeDockerImage))
		Expect(application.Spec.DockerImage).NotTo(BeNil())
		Expect(application.Spec.DockerImage.Image).To(Equal("nginx:latest"))
		Expect(application.Spec.DockerImage.Tag).To(Equal("v1.0.0"))

		// Cleanup
		_ = k8sClient.Delete(ctx, &application)
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("creates GitRepository application successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Git App",
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

		// Create production environment via API
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

		// Create application
		payload := models.ApplicationCreateRequest{
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
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/environments/"+createdEnvironment.UUID+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var response models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		// Verify response
		Expect(response.Name).To(Equal("My Node App"))
		Expect(response.Type).To(Equal(models.ApplicationTypeGitRepository))
		Expect(response.GitRepository).NotTo(BeNil())
		Expect(response.GitRepository.Provider).To(Equal(models.GitProviderGitHub))
		Expect(response.GitRepository.Repository).To(Equal("myorg/myapp"))
		Expect(response.GitRepository.PublicAccess).To(BeTrue())
		Expect(response.GitRepository.Branch).To(Equal("main"))
		Expect(response.GitRepository.BuildCommand).To(Equal("npm run build"))
		Expect(response.GitRepository.StartCommand).To(Equal("npm start"))

		// Verify CRD
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + response.UUID + "",
			Namespace: "default",
		}, &application)
		Expect(err).NotTo(HaveOccurred())

		Expect(application.Spec.Type).To(Equal(v1alpha1.ApplicationTypeGitRepository))
		Expect(application.Spec.GitRepository).NotTo(BeNil())
		Expect(application.Spec.GitRepository.Provider).To(Equal(v1alpha1.GitProviderGitHub))
		Expect(application.Spec.GitRepository.Repository).To(Equal("myorg/myapp"))

		// Cleanup
		_ = k8sClient.Delete(ctx, &application)
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("creates MySQL application successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for MySQL App",
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

		// Create production environment via API
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

		// Create application
		payload := models.ApplicationCreateRequest{
			Name: "My Database",
			Type: models.ApplicationTypeMySQL,
			MySQL: &models.MySQLConfig{
				Version:  "8.0",
				Database: "myapp",
			},
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/environments/"+createdEnvironment.UUID+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated))

		var response models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		// Verify response
		Expect(response.Name).To(Equal("My Database"))
		Expect(response.Type).To(Equal(models.ApplicationTypeMySQL))
		Expect(response.MySQL).NotTo(BeNil())
		Expect(response.MySQL.Version).To(Equal("8.0"))
		Expect(response.MySQL.Database).To(Equal("myapp"))

		// Verify CRD
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + response.UUID + "",
			Namespace: "default",
		}, &application)
		Expect(err).NotTo(HaveOccurred())

		Expect(application.Spec.Type).To(Equal(v1alpha1.ApplicationTypeMySQL))
		Expect(application.Spec.MySQL).NotTo(BeNil())
		Expect(application.Spec.MySQL.Version).To(Equal("8.0"))
		Expect(application.Spec.MySQL.Database).To(Equal("myapp"))

		// Cleanup
		_ = k8sClient.Delete(ctx, &application)
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("retrieves application by slug", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for App Retrieval",
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

		// Create production environment via API
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

		// Create application
		appPayload := models.ApplicationCreateRequest{
			Name: "Test App for Retrieval",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "redis:latest",
			},
		}

		jsonData, err = json.Marshal(appPayload)
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

		// Retrieve application
		req, err = http.NewRequest("GET", "/v1/applications/"+createdApplication.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var retrievedApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &retrievedApplication)
		Expect(err).NotTo(HaveOccurred())

		// Verify
		Expect(retrievedApplication.UUID).To(Equal(createdApplication.UUID))
		Expect(retrievedApplication.Slug).To(Equal(createdApplication.Slug))
		Expect(retrievedApplication.Name).To(Equal(createdApplication.Name))
		Expect(retrievedApplication.Type).To(Equal(createdApplication.Type))

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

	It("retrieves applications by environment", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Apps List",
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

		// Create production environment via API
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

		// Create application
		appPayload := models.ApplicationCreateRequest{
			Name: "Test App for List",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "redis:latest",
			},
		}

		jsonData, err = json.Marshal(appPayload)
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

		// List applications
		req, err = http.NewRequest("GET", "/v1/environments/"+createdEnvironment.UUID+"/applications", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var applications []models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &applications)
		Expect(err).NotTo(HaveOccurred())

		Expect(applications).To(HaveLen(1))
		Expect(applications[0].UUID).To(Equal(createdApplication.UUID))

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

	It("updates application name and Docker image", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for App Update",
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

		// Create production environment via API
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

		// Update application
		updateReq := models.ApplicationUpdateRequest{
			Name: stringPtr("Updated App Name"),
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:2.0",
				Tag:   "latest",
			},
		}

		jsonData, err = json.Marshal(updateReq)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("PATCH", "/v1/applications/"+createdApplication.UUID, bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
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
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &application)
		Expect(err).NotTo(HaveOccurred())

		annotations := application.GetAnnotations()
		Expect(annotations[validation.AnnotationResourceName]).To(Equal("Updated App Name"))
		Expect(application.Spec.DockerImage.Image).To(Equal("nginx:2.0"))
		Expect(application.Spec.DockerImage.Tag).To(Equal("latest"))

		// Cleanup
		_ = k8sClient.Delete(ctx, &application)
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("deletes existing application", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for App Delete",
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

		// Create production environment via API
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

		// Create application
		appPayload := models.ApplicationCreateRequest{
			Name: "App to Delete",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:latest",
			},
		}

		jsonData, err = json.Marshal(appPayload)
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

		// Verify exists
		var k8sApplication v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &k8sApplication)
		Expect(err).NotTo(HaveOccurred())

		// Delete application
		req, err = http.NewRequest("DELETE", "/v1/applications/"+createdApplication.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNoContent))

		// Verify deleted
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + createdApplication.UUID + "",
			Namespace: "default",
		}, &application)
		Expect(err).To(HaveOccurred())

		// Cleanup
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})

	It("retrieves single application with auto-loaded domains", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Domain Loading",
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

		// Create production environment via API
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

		// Create domains
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

		createdDomains := []models.ApplicationDomainResponse{}
		for _, domainPayload := range domainPayloads {
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
			createdDomains = append(createdDomains, createdDomain)
		}

		// Retrieve application
		req, err = http.NewRequest("GET", "/v1/applications/"+createdApplication.UUID, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var retrievedApplication models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &retrievedApplication)
		Expect(err).NotTo(HaveOccurred())

		// Verify
		Expect(retrievedApplication.UUID).To(Equal(createdApplication.UUID))
		Expect(retrievedApplication.Domains).To(HaveLen(2))

		domainsByDomain := make(map[string]models.ApplicationDomainResponse)
		for _, domain := range retrievedApplication.Domains {
			domainsByDomain[domain.Domain] = domain
		}

		defaultDomain, exists := domainsByDomain["app.example.com"]
		Expect(exists).To(BeTrue())
		Expect(defaultDomain.Port).To(Equal(int32(3000)))
		Expect(defaultDomain.Type).To(Equal(models.ApplicationDomainTypeDefault))
		Expect(defaultDomain.Default).To(BeTrue())
		Expect(defaultDomain.TLSEnabled).To(BeTrue())

		customDomain, exists := domainsByDomain["custom.example.com"]
		Expect(exists).To(BeTrue())
		Expect(customDomain.Port).To(Equal(int32(8080)))
		Expect(customDomain.Type).To(Equal(models.ApplicationDomainTypeCustom))
		Expect(customDomain.Default).To(BeFalse())
		Expect(customDomain.TLSEnabled).To(BeFalse())

		// Cleanup
		for _, domain := range createdDomains {
			var applicationDomain v1alpha1.ApplicationDomain
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      "domain-" + domain.UUID + "",
				Namespace: "default",
			}, &applicationDomain)
			if err == nil {
				_ = k8sClient.Delete(ctx, &applicationDomain)
			}
		}
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

	It("retrieves applications by environment with batch-loaded domains and deployments", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Batch Loading",
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

		// Create production environment via API
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

		// Create application
		appPayload := models.ApplicationCreateRequest{
			Name: "App with Batch Data",
			Type: models.ApplicationTypeDockerImage,
			DockerImage: &models.DockerImageConfig{
				Image: "nginx:latest",
			},
		}

		jsonData, err = json.Marshal(appPayload)
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

		// Create domains
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

		for _, domainPayload := range domainPayloads {
			jsonData, err = json.Marshal(domainPayload)
			Expect(err).NotTo(HaveOccurred())

			req, err = http.NewRequest("POST", "/v1/applications/"+createdApplication.UUID+"/domains", bytes.NewBuffer(jsonData))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusCreated))
		}

		// Create deployment
		deploymentPayload := models.DeploymentCreateRequest{
			ApplicationUUID: createdApplication.UUID,
			GitRepository: &models.GitRepositoryDeploymentConfig{
				CommitSHA: "abc123def456",
				Branch:    "main",
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

		// List applications
		req, err = http.NewRequest("GET", "/v1/environments/"+createdEnvironment.UUID+"/applications", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var applications []models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &applications)
		Expect(err).NotTo(HaveOccurred())

		Expect(applications).To(HaveLen(1))

		app := applications[0]
		Expect(app.UUID).To(Equal(createdApplication.UUID))
		Expect(app.Domains).To(HaveLen(2))

		domainsByDomain := make(map[string]models.ApplicationDomainResponse)
		for _, domain := range app.Domains {
			domainsByDomain[domain.Domain] = domain
		}

		Expect(domainsByDomain).To(HaveKey("app.example.com"))
		Expect(domainsByDomain).To(HaveKey("custom.example.com"))

		Expect(app.LatestDeployment).NotTo(BeNil())
		Expect(app.LatestDeployment.UUID).To(Equal(createdDeployment.UUID))
		Expect(app.LatestDeployment.Slug).To(Equal(createdDeployment.Slug))
		Expect(app.LatestDeployment.GitRepository.CommitSHA).To(Equal("abc123def456"))
		Expect(app.LatestDeployment.GitRepository.Branch).To(Equal("main"))

		// Cleanup
		var deployment v1alpha1.Deployment
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "deployment-" + createdDeployment.Slug + "",
			Namespace: "default",
		}, &deployment)
		if err == nil {
			_ = k8sClient.Delete(ctx, &deployment)
		}

		var domainList v1alpha1.ApplicationDomainList
		err = k8sClient.List(ctx, &domainList, client.MatchingLabels{
			validation.LabelApplicationUUID: createdApplication.UUID,
		})
		if err == nil {
			for _, domain := range domainList.Items {
				_ = k8sClient.Delete(ctx, &domain)
			}
		}

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

	It("creates GitRepository application with Dockerfile BuildType successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
		apiKey := generateTestAPIKey()
		router := setupIntegrationTestRouter(apiKey)

		// Create project
		projectPayload := models.ProjectCreateRequest{
			Name:          "Test Project for Dockerfile App",
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

		// Create production environment via API
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

		// Create application with Dockerfile BuildType
		payload := models.ApplicationCreateRequest{
			Name: "My Dockerfile App",
			Type: models.ApplicationTypeGitRepository,
			GitRepository: &models.GitRepositoryConfig{
				Provider:     models.GitProviderGitHub,
				Repository:   "kibamail/todos-api-flask",
				PublicAccess: true,
				Branch:       "main",
				BuildType:    models.BuildTypeDockerfile,
				DockerfileBuild: &models.DockerfileBuildConfig{
					DockerfilePath: "Dockerfile",
					BuildContext:   ".",
				},
			},
		}

		jsonData, err = json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		req, err = http.NewRequest("POST", "/v1/environments/"+createdEnvironment.UUID+"/applications", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusCreated), "Response: %s", w.Body.String())

		var response models.ApplicationResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		Expect(err).NotTo(HaveOccurred())

		// Verify response
		Expect(response.Name).To(Equal("My Dockerfile App"))
		Expect(response.Type).To(Equal(models.ApplicationTypeGitRepository))
		Expect(response.GitRepository).NotTo(BeNil())
		Expect(response.GitRepository.BuildType).To(Equal(models.BuildTypeDockerfile))
		Expect(response.GitRepository.DockerfileBuild).NotTo(BeNil())
		Expect(response.GitRepository.DockerfileBuild.DockerfilePath).To(Equal("Dockerfile"))
		Expect(response.GitRepository.DockerfileBuild.BuildContext).To(Equal("."))

		// Verify CRD
		var application v1alpha1.Application
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      "application-" + response.UUID + "",
			Namespace: "default",
		}, &application)
		Expect(err).NotTo(HaveOccurred())

		Expect(application.Spec.Type).To(Equal(v1alpha1.ApplicationTypeGitRepository))
		Expect(application.Spec.GitRepository).NotTo(BeNil())
		Expect(application.Spec.GitRepository.BuildType).To(Equal(v1alpha1.BuildTypeDockerfile))
		Expect(application.Spec.GitRepository.DockerfileBuild).NotTo(BeNil())
		Expect(application.Spec.GitRepository.DockerfileBuild.DockerfilePath).To(Equal("Dockerfile"))
		Expect(application.Spec.GitRepository.DockerfileBuild.BuildContext).To(Equal("."))

		// Cleanup
		_ = k8sClient.Delete(ctx, &application)
		var project v1alpha1.Project
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "project-" + createdProject.UUID}, &project)
		if err == nil {
			_ = k8sClient.Delete(ctx, &project)
		}
	})
})
