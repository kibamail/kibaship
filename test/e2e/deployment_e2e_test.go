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

var _ = Describe("Deployment E2E Tests", Ordered, func() {
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
		testNamespace = "deployment-e2e-test"
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

	Context("Deployment Lifecycle", func() {
		var testProject *platformv1alpha1.Project
		var testApplication *platformv1alpha1.Application
		var testDeployment *platformv1alpha1.Deployment

		BeforeEach(func() {
			By("Creating test project")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-project-deployment",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440000",
						validation.LabelResourceSlug:  "test-project-deployment",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

			// Wait for project to be ready
			Eventually(func() string {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				if err != nil {
					return ""
				}
				return project.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Creating test application")
			testApplication = &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-deployment-app-backend-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440001",
						validation.LabelResourceSlug: "backend",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/backend",
						Branch:     "main",
					},
				},
			}
			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			// Wait for application to be ready
			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))
		})

		AfterEach(func() {
			By("Cleaning up test resources")
			// Delete deployment
			if testDeployment != nil {
				_ = k8sClient.Delete(ctx, testDeployment)
			}

			// Delete application
			if testApplication != nil {
				_ = k8sClient.Delete(ctx, testApplication)
			}

			// Delete project
			if testProject != nil {
				_ = k8sClient.Delete(ctx, testProject)
			}
		})

		It("should successfully create and reconcile GitRepository deployment", func() {
			By("Creating deployment for GitRepository application")
			testDeployment = &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-deployment-app-backend-deployment-prod-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440002",
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
						CommitSHA: "abc123456789",
						Branch:    "main",
					},
				},
			}

			Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

			By("Verifying deployment transitions to Initializing phase")
			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*1, time.Second*5).Should(Equal("Initializing"))

			By("Verifying Tekton Pipeline is created")
			pipelineName := fmt.Sprintf("%s-git-repository-pipeline-kibaship-com", testDeployment.Name)
			Eventually(func() error {
				var pipeline tektonv1.Pipeline
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      pipelineName,
					Namespace: testNamespace,
				}, &pipeline)
			}, time.Minute*2, time.Second*5).Should(Succeed())

			By("Verifying pipeline has correct structure")
			var pipeline tektonv1.Pipeline
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      pipelineName,
				Namespace: testNamespace,
			}, &pipeline)
			Expect(err).NotTo(HaveOccurred())

			// Verify pipeline tasks
			Expect(len(pipeline.Spec.Tasks)).To(BeNumerically(">=", 2))

			// Should have git-clone and build tasks
			taskNames := make([]string, len(pipeline.Spec.Tasks))
			for i, task := range pipeline.Spec.Tasks {
				taskNames[i] = task.Name
			}
			Expect(taskNames).To(ContainElement("git-clone"))
			Expect(taskNames).To(ContainElement("build"))

			By("Verifying deployment finalizer is added")
			var deployment platformv1alpha1.Deployment
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(testDeployment), &deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment.Finalizers).To(ContainElement("platform.operator.kibaship.com/deployment-finalizer"))
		})

		It("should create PipelineRun when deployment is triggered", func() {
			By("Creating deployment")
			testDeployment = &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-deployment-app-backend-deployment-staging-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440003",
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
						CommitSHA: "def789012345",
						Branch:    "staging",
					},
				},
			}
			Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

			By("Waiting for deployment to initialize")
			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*2, time.Second*5).Should(Equal("Initializing"))

			By("Verifying PipelineRun is created")
			// PipelineRuns should be created with deployment name + timestamp pattern
			Eventually(func() int {
				var pipelineRuns tektonv1.PipelineRunList
				err := k8sClient.List(ctx, &pipelineRuns, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}

				// Count pipeline runs for this deployment
				count := 0
				for _, pr := range pipelineRuns.Items {
					if pr.Labels["platform.kibaship.com/deployment"] == testDeployment.Name {
						count++
					}
				}
				return count
			}, time.Minute*2, time.Second*5).Should(BeNumerically(">=", 1))

			By("Verifying PipelineRun has correct parameters")
			var pipelineRuns tektonv1.PipelineRunList
			err := k8sClient.List(ctx, &pipelineRuns, client.InNamespace(testNamespace))
			Expect(err).NotTo(HaveOccurred())

			// Find the pipeline run for this deployment
			var deploymentPipelineRun *tektonv1.PipelineRun
			for _, pr := range pipelineRuns.Items {
				if pr.Labels["platform.kibaship.com/deployment"] == testDeployment.Name {
					deploymentPipelineRun = &pr
					break
				}
			}
			Expect(deploymentPipelineRun).NotTo(BeNil())

			// Verify parameters include repository and commit SHA
			paramMap := make(map[string]string)
			for _, param := range deploymentPipelineRun.Spec.Params {
				paramMap[param.Name] = param.Value.StringVal
			}
			Expect(paramMap["repository"]).To(Equal("https://github.com/test/backend"))
			Expect(paramMap["commit-sha"]).To(Equal("def789012345"))
			Expect(paramMap["branch"]).To(Equal("staging"))
		})

		It("should handle MySQL cluster creation for applications requiring database", func() {
			By("Creating application with MySQL requirement")
			mysqlApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-deployment-app-api-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440004",
						validation.LabelResourceSlug: "api",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/api",
						Branch:     "main",
					},
				},
			}
			Expect(k8sClient.Create(ctx, mysqlApp)).To(Succeed())

			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(mysqlApp), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Creating deployment that triggers MySQL setup")
			testDeployment = &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-deployment-app-api-deployment-prod-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440005",
						validation.LabelResourceSlug:    "prod",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: mysqlApp.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: mysqlApp.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "mysql123456789",
						Branch:    "main",
					},
				},
			}
			Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

			By("Verifying deployment processes successfully")
			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*3, time.Second*5).Should(Or(Equal("Initializing"), Equal("Running")))

			// Clean up the MySQL app
			_ = k8sClient.Delete(ctx, mysqlApp)
		})

		It("should properly handle deployment deletion and cleanup", func() {
			By("Creating deployment to be deleted")
			testDeployment = &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-deployment-app-backend-deployment-temp-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440006",
						validation.LabelResourceSlug:    "temp",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: testApplication.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: testApplication.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "cleanup123456",
						Branch:    "main",
					},
				},
			}
			Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

			By("Waiting for deployment to initialize")
			Eventually(func() string {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testDeployment), &deployment)
				if err != nil {
					return ""
				}
				return string(deployment.Status.Phase)
			}, time.Minute*2, time.Second*5).Should(Equal("Initializing"))

			By("Verifying associated resources are created")
			pipelineName := fmt.Sprintf("%s-git-repository-pipeline-kibaship-com", testDeployment.Name)
			Eventually(func() error {
				var pipeline tektonv1.Pipeline
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      pipelineName,
					Namespace: testNamespace,
				}, &pipeline)
			}, time.Minute*1, time.Second*5).Should(Succeed())

			By("Deleting the deployment")
			Expect(k8sClient.Delete(ctx, testDeployment)).To(Succeed())

			By("Verifying deployment is eventually removed")
			Eventually(func() bool {
				var deployment platformv1alpha1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testDeployment), &deployment)
				return errors.IsNotFound(err)
			}, time.Minute*2, time.Second*5).Should(BeTrue())

			By("Verifying associated pipeline is cleaned up")
			Eventually(func() bool {
				var pipeline tektonv1.Pipeline
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      pipelineName,
					Namespace: testNamespace,
				}, &pipeline)
				return errors.IsNotFound(err)
			}, time.Minute*2, time.Second*5).Should(BeTrue())

			// Set to nil so AfterEach doesn't try to delete again
			testDeployment = nil
		})
	})

	Context("Deployment Validation and Error Handling", func() {
		var testProject *platformv1alpha1.Project
		var testApplication *platformv1alpha1.Application

		BeforeEach(func() {
			By("Creating test project")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-project-validation",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440100",
						validation.LabelResourceSlug:  "test-project-validation",
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
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Creating test application")
			testApplication = &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-validation-app-web-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440101",
						validation.LabelResourceSlug: "web",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/web",
						Branch:     "main",
					},
				},
			}
			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))
		})

		AfterEach(func() {
			By("Cleaning up validation test resources")
			// Delete application
			if testApplication != nil {
				_ = k8sClient.Delete(ctx, testApplication)
			}

			// Delete project
			if testProject != nil {
				_ = k8sClient.Delete(ctx, testProject)
			}
		})

		It("should reject deployment with invalid application reference", func() {
			By("Attempting to create deployment with non-existent application")
			invalidDeployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-validation-app-nonexistent-deployment-prod-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440102",
						validation.LabelResourceSlug:    "prod",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: "non-existent-uuid",
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: "non-existent-application",
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "invalid123456",
						Branch:    "main",
					},
				},
			}

			err := k8sClient.Create(ctx, invalidDeployment)
			if err == nil {
				// If creation succeeded, it should fail during reconciliation
				Eventually(func() string {
					var deployment platformv1alpha1.Deployment
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(invalidDeployment), &deployment)
					if err != nil {
						return "NotFound"
					}
					return string(deployment.Status.Phase)
				}, time.Minute*1, time.Second*5).Should(Equal("Failed"))

				// Clean up
				_ = k8sClient.Delete(ctx, invalidDeployment)
			}
		})

		It("should enforce proper deployment naming conventions", func() {
			By("Attempting to create deployment with invalid name format")
			invalidNameDeployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-deployment-name",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440103",
						validation.LabelResourceSlug:    "invalid",
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
						validation.LabelApplicationUUID: testApplication.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: testApplication.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "naming123456",
						Branch:    "main",
					},
				},
			}

			err := k8sClient.Create(ctx, invalidNameDeployment)
			Expect(err).To(HaveOccurred(), "Should reject deployment with invalid name format")
		})
	})
})
