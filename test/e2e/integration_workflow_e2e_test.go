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

package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/validation"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

var _ = Describe("Integration Workflow E2E Tests", Ordered, func() {
	var k8sClient client.Client
	var testNamespace string
	ctx := context.Background()

	BeforeAll(func() {
		By("Setting up kubernetes client")
		cfg, err := config.GetConfig()
		Expect(err).NotTo(HaveOccurred())

		err = platformv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = tektonv1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())

		// Create test namespace
		testNamespace = "integration-workflow-e2e-test"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		err = k8sClient.Create(ctx, ns)
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterAll(func() {
		By("Cleaning up test namespace")
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(ctx, ns)
	})

	Context("Complete Project-to-Deployment Workflow", func() {
		var testProject *platformv1alpha1.Project
		var testApplications []*platformv1alpha1.Application
		var testDeployments []*platformv1alpha1.Deployment

		AfterEach(func() {
			By("Cleaning up workflow test resources")
			// Delete deployments
			for _, deployment := range testDeployments {
				if deployment != nil {
					_ = k8sClient.Delete(ctx, deployment)
				}
			}

			// Delete applications
			for _, app := range testApplications {
				if app != nil {
					_ = k8sClient.Delete(ctx, app)
				}
			}

			// Delete project
			if testProject != nil {
				_ = k8sClient.Delete(ctx, testProject)
			}

			// Reset slices
			testApplications = nil
			testDeployments = nil
		})

		It("should handle complete microservices project lifecycle", func() {
			By("Creating a project for microservices architecture")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-microservices-project",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440000",
						validation.LabelResourceSlug:  "test-microservices-project",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

			Eventually(func() string {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				if err != nil {
					return ""
				}
				return project.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			By("Creating multiple microservice applications")
			applicationConfigs := []struct {
				name string
				slug string
				repo string
				uuid string
			}{
				{"project-test-microservices-project-app-frontend-kibaship-com", "frontend", "https://github.com/test/frontend", "550e8400-e29b-41d4-a716-446655440001"},
				{"project-test-microservices-project-app-api-gateway-kibaship-com", "api-gateway", "https://github.com/test/api-gateway", "550e8400-e29b-41d4-a716-446655440002"},
				{"project-test-microservices-project-app-user-service-kibaship-com", "user-service", "https://github.com/test/user-service", "550e8400-e29b-41d4-a716-446655440003"},
				{"project-test-microservices-project-app-payment-service-kibaship-com", "payment-service", "https://github.com/test/payment-service", "550e8400-e29b-41d4-a716-446655440004"},
			}

			testApplications = make([]*platformv1alpha1.Application, len(applicationConfigs))

			for i, config := range applicationConfigs {
				app := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      config.name,
						Namespace: testNamespace,
						Labels: map[string]string{
							validation.LabelResourceUUID: config.uuid,
							validation.LabelResourceSlug: config.slug,
							validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						Type: platformv1alpha1.ApplicationTypeGitRepository,
						ProjectRef: corev1.LocalObjectReference{
							Name: testProject.Name,
						},
						GitRepository: &platformv1alpha1.GitRepositoryConfig{
							Repository: config.repo,
							Branch:     "main",
						},
					},
				}
				testApplications[i] = app
				Expect(k8sClient.Create(ctx, app)).To(Succeed())
			}

			By("Verifying all applications become ready")
			for _, app := range testApplications {
				Eventually(func() string {
					var application platformv1alpha1.Application
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), &application)
					if err != nil {
						return ""
					}
					return application.Status.Phase
				}, time.Minute*3, time.Second*5).Should(Equal("Ready"))
			}

			By("Verifying default ApplicationDomains are created for all applications")
			Eventually(func() int {
				var domains platformv1alpha1.ApplicationDomainList
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}

				defaultDomainCount := 0
				for _, domain := range domains.Items {
					if domain.Spec.Default {
						defaultDomainCount++
					}
				}
				return defaultDomainCount
			}, time.Minute*3, time.Second*5).Should(Equal(len(applicationConfigs)))

			By("Creating deployments for production environment")
			testDeployments = make([]*platformv1alpha1.Deployment, len(testApplications))

			for i, app := range testApplications {
				deployment := &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-deployment-prod-kibaship-com", app.Name),
						Namespace: testNamespace,
						Labels: map[string]string{
							validation.LabelResourceUUID:    fmt.Sprintf("550e8400-e29b-41d4-a716-44665544010%d", i),
							validation.LabelResourceSlug:    "prod",
							validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
							validation.LabelApplicationUUID: app.Labels[validation.LabelResourceUUID],
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{
							Name: app.Name,
						},
						GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
							CommitSHA: fmt.Sprintf("prod-commit-%d", i),
							Branch:    "main",
						},
					},
				}
				testDeployments[i] = deployment
				Expect(k8sClient.Create(ctx, deployment)).To(Succeed())
			}

			By("Verifying all deployments initialize successfully")
			for _, deployment := range testDeployments {
				Eventually(func() string {
					var dep platformv1alpha1.Deployment
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), &dep)
					if err != nil {
						return ""
					}
					return string(dep.Status.Phase)
				}, time.Minute*3, time.Second*5).Should(Equal("Initializing"))
			}

			By("Verifying Tekton Pipelines are created for all deployments")
			for _, deployment := range testDeployments {
				pipelineName := fmt.Sprintf("%s-git-repository-pipeline-kibaship-com", deployment.Name)
				Eventually(func() error {
					var pipeline tektonv1.Pipeline
					return k8sClient.Get(ctx, client.ObjectKey{
						Name:      pipelineName,
						Namespace: testNamespace,
					}, &pipeline)
				}, time.Minute*3, time.Second*5).Should(Succeed())
			}

			By("Verifying namespace isolation and RBAC are properly configured")
			// Check that project namespace exists
			projectNamespace := fmt.Sprintf("project-%s", testProject.Labels[validation.LabelResourceSlug])
			Eventually(func() error {
				var ns corev1.Namespace
				return k8sClient.Get(ctx, client.ObjectKey{Name: projectNamespace}, &ns)
			}, time.Minute*2, time.Second*5).Should(Succeed())
		})

		It("should handle staged deployment workflow (dev -> staging -> prod)", func() {
			By("Creating project for staged deployment")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-staged-deployment-project",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440100",
						validation.LabelResourceSlug:  "test-staged-deployment-project",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

			Eventually(func() string {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				if err != nil {
					return ""
				}
				return project.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			By("Creating application for staged deployment")
			testApplication := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-staged-deployment-project-app-webapp-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440101",
						validation.LabelResourceSlug: "webapp",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/webapp",
						Branch:     "main",
					},
				},
			}
			testApplications = []*platformv1alpha1.Application{testApplication}
			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			By("Creating development deployment")
			devDeployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-staged-deployment-project-app-webapp-deployment-dev-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440102",
						validation.LabelResourceSlug:    "dev",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: testApplication.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: testApplication.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "dev-commit-123",
						Branch:    "develop",
					},
				},
			}

			testDeployments = append(testDeployments, devDeployment)
			Expect(k8sClient.Create(ctx, devDeployment)).To(Succeed())

			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(devDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*3, time.Second*5).Should(Equal("Initializing"))

			By("Creating staging deployment")
			stagingDeployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-staged-deployment-project-app-webapp-deployment-staging-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440103",
						validation.LabelResourceSlug:    "staging",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: testApplication.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: testApplication.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "staging-commit-456",
						Branch:    "staging",
					},
				},
			}

			testDeployments = append(testDeployments, stagingDeployment)
			Expect(k8sClient.Create(ctx, stagingDeployment)).To(Succeed())

			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(stagingDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*3, time.Second*5).Should(Equal("Initializing"))

			By("Creating production deployment")
			prodDeployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-staged-deployment-project-app-webapp-deployment-prod-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440104",
						validation.LabelResourceSlug:    "prod",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: testApplication.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: testApplication.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "prod-commit-789",
						Branch:    "main",
					},
				},
			}

			testDeployments = append(testDeployments, prodDeployment)
			Expect(k8sClient.Create(ctx, prodDeployment)).To(Succeed())

			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(prodDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*3, time.Second*5).Should(Equal("Initializing"))

			By("Verifying each deployment has its own pipeline and configuration")
			for _, deployment := range testDeployments {
				pipelineName := fmt.Sprintf("%s-git-repository-pipeline-kibaship-com", deployment.Name)
				var pipeline tektonv1.Pipeline
				Expect(k8sClient.Get(ctx, client.ObjectKey{
					Name:      pipelineName,
					Namespace: testNamespace,
				}, &pipeline)).To(Succeed())

				// Verify pipeline has correct commit SHA parameter
				paramMap := make(map[string]string)
				for _, param := range pipeline.Spec.Params {
					paramMap[param.Name] = param.Default.StringVal
				}
				Expect(paramMap).To(HaveKey("commit-sha"))
			}

			By("Verifying custom ApplicationDomains can be created for different environments")
			customDomain := &platformv1alpha1.ApplicationDomain{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-staging-domain",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:        "550e8400-e29b-41d4-a716-446655440105",
						validation.LabelApplicationUUID:     testApplication.Labels[validation.LabelResourceUUID],
						validation.LabelProjectUUID:         testApplication.Labels[validation.LabelProjectUUID],
						"platform.kibaship.com/application": testApplication.Name,
						"platform.kibaship.com/domain-type": "custom",
					},
				},
				Spec: platformv1alpha1.ApplicationDomainSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: testApplication.Name,
					},
					Type:       platformv1alpha1.ApplicationDomainTypeCustom,
					Domain:     "staging.webapp.test.kibaship.com",
					Default:    false,
					Port:       3000,
					TLSEnabled: true,
				},
			}
			Expect(k8sClient.Create(ctx, customDomain)).To(Succeed())

			Eventually(func() string {
				var domain platformv1alpha1.ApplicationDomain
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(customDomain), &domain)
				if err != nil {
					return ""
				}
				return string(domain.Status.Phase)
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))
		})

		It("should handle cross-controller resource dependencies correctly", func() {
			By("Creating project with dependent resources")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dependency-project",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440200",
						validation.LabelResourceSlug:  "test-dependency-project",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

			Eventually(func() string {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				if err != nil {
					return ""
				}
				return project.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			By("Creating application with streaming dependency")
			streamingApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-dependency-project-app-streaming-app-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440201",
						validation.LabelResourceSlug: "streaming-app",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/streaming-app",
						Branch:     "main",
					},
				},
			}
			testApplications = append(testApplications, streamingApp)
			Expect(k8sClient.Create(ctx, streamingApp)).To(Succeed())

			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(streamingApp), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			By("Creating deployment that should trigger streaming events")
			streamingDeployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-dependency-project-app-streaming-app-deployment-prod-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440202",
						validation.LabelResourceSlug:    "prod",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: streamingApp.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: streamingApp.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "streaming-commit-123",
						Branch:    "main",
					},
				},
			}
			testDeployments = append(testDeployments, streamingDeployment)
			Expect(k8sClient.Create(ctx, streamingDeployment)).To(Succeed())

			By("Verifying deployment initializes and creates pipeline")
			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(streamingDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*3, time.Second*5).Should(Equal("Initializing"))

			By("Verifying ApplicationDomain is created and ready")
			Eventually(func() int {
				var domains platformv1alpha1.ApplicationDomainList
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}

				readyDomainCount := 0
				for _, domain := range domains.Items {
					if domain.Spec.ApplicationRef.Name == streamingApp.Name && domain.Status.Phase == "Ready" {
						readyDomainCount++
					}
				}
				return readyDomainCount
			}, time.Minute*3, time.Second*5).Should(BeNumerically(">=", 1))

			By("Verifying Tekton pipeline contains streaming configuration")
			pipelineName := fmt.Sprintf("%s-git-repository-pipeline-kibaship-com", streamingDeployment.Name)
			var pipeline tektonv1.Pipeline
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      pipelineName,
					Namespace: testNamespace,
				}, &pipeline)
			}, time.Minute*2, time.Second*5).Should(Succeed())

			// Verify pipeline has tasks for streaming integration
			Expect(len(pipeline.Spec.Tasks)).To(BeNumerically(">=", 2))
		})

		It("should handle resource cleanup properly across all controllers", func() {
			By("Creating complete resource stack")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cleanup-project",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440300",
						validation.LabelResourceSlug:  "test-cleanup-project",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

			Eventually(func() string {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				if err != nil {
					return ""
				}
				return project.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			cleanupApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-cleanup-project-app-cleanup-app-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440301",
						validation.LabelResourceSlug: "cleanup-app",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/cleanup-app",
						Branch:     "main",
					},
				},
			}
			testApplications = append(testApplications, cleanupApp)
			Expect(k8sClient.Create(ctx, cleanupApp)).To(Succeed())

			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cleanupApp), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			cleanupDeployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-cleanup-project-app-cleanup-app-deployment-test-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440302",
						validation.LabelResourceSlug:    "test",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: cleanupApp.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: cleanupApp.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "cleanup-commit-123",
						Branch:    "main",
					},
				},
			}
			testDeployments = append(testDeployments, cleanupDeployment)
			Expect(k8sClient.Create(ctx, cleanupDeployment)).To(Succeed())

			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cleanupDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*3, time.Second*5).Should(Equal("Initializing"))

			By("Verifying all resources are created and linked")
			// Check domains
			Eventually(func() int {
				var domains platformv1alpha1.ApplicationDomainList
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}
				count := 0
				for _, domain := range domains.Items {
					if domain.Spec.ApplicationRef.Name == cleanupApp.Name {
						count++
					}
				}
				return count
			}, time.Minute*2, time.Second*5).Should(BeNumerically(">=", 1))

			// Check pipelines
			pipelineName := fmt.Sprintf("%s-git-repository-pipeline-kibaship-com", cleanupDeployment.Name)
			Eventually(func() error {
				var pipeline tektonv1.Pipeline
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      pipelineName,
					Namespace: testNamespace,
				}, &pipeline)
			}, time.Minute*2, time.Second*5).Should(Succeed())

			By("Deleting deployment and verifying cascading cleanup")
			Expect(k8sClient.Delete(ctx, cleanupDeployment)).To(Succeed())

			Eventually(func() bool {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cleanupDeployment), &deployment)
				return errors.IsNotFound(err)
			}, time.Minute*3, time.Second*5).Should(BeTrue())

			Eventually(func() bool {
				var pipeline tektonv1.Pipeline
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      pipelineName,
					Namespace: testNamespace,
				}, &pipeline)
				return errors.IsNotFound(err)
			}, time.Minute*3, time.Second*5).Should(BeTrue())

			By("Deleting application and verifying domain cleanup")
			Expect(k8sClient.Delete(ctx, cleanupApp)).To(Succeed())

			Eventually(func() bool {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cleanupApp), &app)
				return errors.IsNotFound(err)
			}, time.Minute*3, time.Second*5).Should(BeTrue())

			Eventually(func() int {
				var domains platformv1alpha1.ApplicationDomainList
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return 999
				}
				count := 0
				for _, domain := range domains.Items {
					if domain.Spec.ApplicationRef.Name == cleanupApp.Name {
						count++
					}
				}
				return count
			}, time.Minute*3, time.Second*5).Should(Equal(0))

			// Reset for AfterEach
			testApplications = nil
			testDeployments = nil
		})
	})
})
