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

var _ = Describe("Deployment Integration", func() {
	var ctx context.Context
	var apiKey string
	var router http.Handler

	BeforeEach(func() {
		ctx = context.Background()
		apiKey = generateTestAPIKey()
		router = setupIntegrationTestRouter(apiKey)
	})

	Describe("Deployment Creation", func() {
		var createdProject *models.ProjectResponse
		var createdApplication *models.ApplicationResponse

		BeforeEach(func() {
			// Create a project
			projectPayload := models.ProjectCreateRequest{
				Name:          "Test Project for Deployments",
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

			err = json.Unmarshal(w.Body.Bytes(), &createdProject)
			Expect(err).NotTo(HaveOccurred())

			// Create a GitRepository application
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
			Expect(err).NotTo(HaveOccurred())

			req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusCreated))

			err = json.Unmarshal(w.Body.Bytes(), &createdApplication)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// Clean up created application and project
			var application v1alpha1.Application
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      "application-" + createdApplication.Slug + "-kibaship-com",
				Namespace: "default",
			}, &application)
			if err == nil {
				_ = k8sClient.Delete(ctx, &application)
			}

			var project v1alpha1.Project
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name:      "project-" + createdProject.Slug + "-kibaship-com",
				Namespace: "default",
			}, &project)
			if err == nil {
				_ = k8sClient.Delete(ctx, &project)
			}
		})

		It("creates GitRepository deployment successfully", NodeTimeout(30*time.Second), func(ctx SpecContext) {
			payload := models.DeploymentCreateRequest{
				ApplicationSlug: createdApplication.Slug,
				GitRepository: &models.GitRepositoryDeploymentConfig{
					CommitSHA: "abc123def456",
					Branch:    "main",
				},
			}

			// Prepare request
			jsonData, err := json.Marshal(payload)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("POST", "/applications/"+createdApplication.Slug+"/deployments", bytes.NewBuffer(jsonData))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status code
			Expect(w.Code).To(Equal(http.StatusCreated))

			// Parse response
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

			// Verify Deployment CRD was actually created in Kubernetes
			var deployment v1alpha1.Deployment
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name:      "deployment-" + response.Slug + "-kibaship-com",
				Namespace: "default",
			}, &deployment)
			Expect(err).NotTo(HaveOccurred(), "Deployment CRD should exist in Kubernetes")

			// Verify CRD labels
			labels := deployment.GetLabels()
			Expect(labels[validation.LabelResourceUUID]).To(Equal(response.UUID))
			Expect(labels[validation.LabelResourceSlug]).To(Equal(response.Slug))
			Expect(labels[validation.LabelProjectUUID]).To(Equal(response.ProjectUUID))
			Expect(labels[validation.LabelApplicationUUID]).To(Equal(response.ApplicationUUID))

			// Verify GitRepository config
			Expect(deployment.Spec.GitRepository).NotTo(BeNil())
			Expect(deployment.Spec.GitRepository.CommitSHA).To(Equal("abc123def456"))
			Expect(deployment.Spec.GitRepository.Branch).To(Equal("main"))

			// Clean up the created deployment
			_ = k8sClient.Delete(ctx, &deployment)
		})
	})

	Describe("Deployment Retrieval", func() {
		var createdProject *models.ProjectResponse
		var createdApplication *models.ApplicationResponse
		var createdDeployment *models.DeploymentResponse

		BeforeEach(func() {
			// Create project, application and deployment
			createdProject, createdApplication, createdDeployment = createTestDeployment(router, apiKey)
		})

		AfterEach(func() {
			// Clean up
			cleanupTestDeployment(ctx, createdProject, createdApplication, createdDeployment)
		})

		It("retrieves deployment by slug", NodeTimeout(30*time.Second), func(ctx SpecContext) {
			req, err := http.NewRequest("GET", "/deployments/"+createdDeployment.Slug, nil)
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
		})

		It("retrieves deployments by application", NodeTimeout(30*time.Second), func(ctx SpecContext) {
			req, err := http.NewRequest("GET", "/applications/"+createdApplication.Slug+"/deployments", nil)
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
		})
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

	req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(http.StatusCreated))

	var createdProject models.ProjectResponse
	err = json.Unmarshal(w.Body.Bytes(), &createdProject)
	Expect(err).NotTo(HaveOccurred())

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
	Expect(err).NotTo(HaveOccurred())

	req, err = http.NewRequest("POST", "/projects/"+createdProject.Slug+"/applications", bytes.NewBuffer(jsonData))
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
		ApplicationSlug: createdApplication.Slug,
		GitRepository: &models.GitRepositoryDeploymentConfig{
			CommitSHA: "test123commit",
			Branch:    "develop",
		},
	}

	jsonData, err = json.Marshal(deploymentPayload)
	Expect(err).NotTo(HaveOccurred())

	req, err = http.NewRequest("POST", "/applications/"+createdApplication.Slug+"/deployments", bytes.NewBuffer(jsonData))
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
		Name:      "deployment-" + deployment.Slug + "-kibaship-com",
		Namespace: "default",
	}, &dep)
	if err == nil {
		_ = k8sClient.Delete(ctx, &dep)
	}

	// Clean up application
	var app v1alpha1.Application
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "application-" + application.Slug + "-kibaship-com",
		Namespace: "default",
	}, &app)
	if err == nil {
		_ = k8sClient.Delete(ctx, &app)
	}

	// Clean up project
	var proj v1alpha1.Project
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "project-" + project.Slug + "-kibaship-com",
		Namespace: "default",
	}, &proj)
	if err == nil {
		_ = k8sClient.Delete(ctx, &proj)
	}
}
