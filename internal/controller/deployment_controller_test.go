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

package controller

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/config"
	"github.com/kibamail/kibaship/pkg/utils"
	"github.com/kibamail/kibaship/pkg/validation"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	expectedPipelineName = "pipeline-deployment-uuid-123"
	// TODO: Database-specific constants removed - will be reimplemented
)

var _ = Describe("Deployment Controller", func() {
	var (
		ctx                  context.Context
		deploymentReconciler *DeploymentReconciler
		testNamespace        *corev1.Namespace
		testProject          *platformv1alpha1.Project
		testEnvironment      *platformv1alpha1.Environment
		testApplication      *platformv1alpha1.Application
		testDeployment       *platformv1alpha1.Deployment
	)

	BeforeEach(func() {
		ctx = context.Background()
		deploymentReconciler = &DeploymentReconciler{
			Client:           k8sClient,
			Scheme:           k8sClient.Scheme(),
			NamespaceManager: NewNamespaceManager(k8sClient),
		}

		// Generate unique IDs to avoid test conflicts
		uniqueID := time.Now().UnixNano()
		projectUUID := fmt.Sprintf("550e8400-e29b-41d4-a716-%012d", uniqueID%1000000000000)
		envUUID := fmt.Sprintf("env-uuid-%d", uniqueID)

		// Create test namespace with unique name
		namespaceName := fmt.Sprintf("test-deployment-ns-%d", rand.Int32())
		testNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}
		Expect(k8sClient.Create(ctx, testNamespace)).To(Succeed())

		// Create test project with unique UUID
		testProject = &platformv1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: utils.GetProjectResourceName(projectUUID),
				Labels: map[string]string{
					"platform.kibaship.com/uuid":           projectUUID,
					"platform.kibaship.com/slug":           "test123",
					"platform.kibaship.com/workspace-uuid": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				},
			},
			Spec: platformv1alpha1.ProjectSpec{},
		}
		Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

		// Create test environment with unique UUID
		testEnvironment = &platformv1alpha1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.GetEnvironmentResourceName(envUUID),
				Namespace: namespaceName,
				Labels: map[string]string{
					validation.LabelResourceUUID: envUUID,
					validation.LabelResourceSlug: "production",
					validation.LabelProjectUUID:  projectUUID,
				},
			},
			Spec: platformv1alpha1.EnvironmentSpec{
				ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
			},
		}
		Expect(k8sClient.Create(ctx, testEnvironment)).To(Succeed())
	})

	AfterEach(func() {
		// Clean up test resources
		if testDeployment != nil {
			_ = k8sClient.Delete(ctx, testDeployment)
		}
		if testApplication != nil {
			_ = k8sClient.Delete(ctx, testApplication)
		}
		if testEnvironment != nil {
			_ = k8sClient.Delete(ctx, testEnvironment)
		}
		if testProject != nil {
			_ = k8sClient.Delete(ctx, testProject)
		}
		if testNamespace != nil {
			_ = k8sClient.Delete(ctx, testNamespace)
		}
	})

	Describe("GitRepository Pipeline Creation", func() {
		Context("When deployment references GitRepository application", func() {
			BeforeEach(func() {
				// Create GitRepository application with branch configured
				testApplication = &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.GetApplicationResourceName("app-uuid-123"),
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "app-uuid-123",
							"platform.kibaship.com/slug":             "myapp",
							"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
							"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
						Type:           platformv1alpha1.ApplicationTypeGitRepository,
						GitRepository: &platformv1alpha1.GitRepositoryConfig{
							Provider:   platformv1alpha1.GitProviderGitHub,
							Repository: "user/test-repo",
							Branch:     "develop",
							SecretRef:  &corev1.LocalObjectReference{Name: "git-secret"},
						},
					},
				}
				Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

				testDeployment = &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.GetDeploymentResourceName("deployment-uuid-123"),
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-123",
							"platform.kibaship.com/slug":             "web",
							"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
							"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
							"platform.kibaship.com/application-uuid": "app-uuid-123",
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: testApplication.Name},
						GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
							CommitSHA: "abc123def456",
							Branch:    "main",
						},
					},
				}
				Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())
			})

			It("should create a pipeline with correct parameters", func() {
				By("Verifying deployment and application exist before reconcile")
				GinkgoWriter.Printf("Deployment: %s/%s\n", testDeployment.Namespace, testDeployment.Name)
				GinkgoWriter.Printf("Application: %s/%s\n", testApplication.Namespace, testApplication.Name)
				GinkgoWriter.Printf("Application Type: %s\n", testApplication.Spec.Type)

				By("Reconciling the deployment")
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying the pipeline was created")
				GinkgoWriter.Printf("Expected pipeline name: %s\n", expectedPipelineName)
				pipeline := &tektonv1.Pipeline{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name:      expectedPipelineName,
						Namespace: testNamespace.Name,
					}, pipeline)
				}).Should(Succeed())

				By("Verifying pipeline has correct parameters")
				Expect(pipeline.Spec.Params).To(HaveLen(2))

				// Check git-commit parameter
				commitParam := pipeline.Spec.Params[0]
				Expect(commitParam.Name).To(Equal("git-commit"))
				Expect(commitParam.Type).To(Equal(tektonv1.ParamTypeString))

				// Check git-branch parameter with default
				branchParam := pipeline.Spec.Params[1]
				Expect(branchParam.Name).To(Equal("git-branch"))
				Expect(branchParam.Type).To(Equal(tektonv1.ParamTypeString))
				Expect(branchParam.Default).NotTo(BeNil())
				Expect(branchParam.Default.StringVal).To(Equal("develop")) // From application config

				By("Verifying task parameters use correct values")
				Expect(pipeline.Spec.Tasks).To(HaveLen(3)) // clone, prepare, build

				// Verify clone task
				cloneTask := pipeline.Spec.Tasks[0]
				Expect(cloneTask.Name).To(Equal("clone-repository"))
				cloneParams := cloneTask.Params
				Expect(cloneParams).To(HaveLen(5))

				urlParam := findTaskParam(cloneParams, "url")
				Expect(urlParam).NotTo(BeNil())
				Expect(urlParam.Value.StringVal).To(Equal("https://github.com/user/test-repo"))

				branchTaskParam := findTaskParam(cloneParams, "branch")
				Expect(branchTaskParam).NotTo(BeNil())
				Expect(branchTaskParam.Value.StringVal).To(Equal("$(params.git-branch)"))

				commitTaskParam := findTaskParam(cloneParams, "commit")
				Expect(commitTaskParam).NotTo(BeNil())
				Expect(commitTaskParam.Value.StringVal).To(Equal("$(params.git-commit)"))

				tokenParam := findTaskParam(cloneParams, "token-secret")
				Expect(tokenParam).NotTo(BeNil())
				Expect(tokenParam.Value.StringVal).To(Equal("git-secret"))

				// Verify prepare and build tasks exist
				Expect(pipeline.Spec.Tasks[1].Name).To(Equal("prepare"))
				Expect(pipeline.Spec.Tasks[2].Name).To(Equal("build"))

				By("Verifying pipeline has correct labels and annotations")
				Expect(pipeline.Labels["app.kubernetes.io/name"]).To(Equal(testProject.Name))
				Expect(pipeline.Labels["app.kubernetes.io/managed-by"]).To(Equal("kibaship"))
				Expect(pipeline.Labels["project.kibaship.com/slug"]).To(Equal("test123"))
				Expect(pipeline.Labels["tekton.dev/pipeline"]).To(Equal("git-repository-railpack"))
			})

			It("should use default branch when application branch is empty", func() {
				By("Updating application to have empty branch")
				testApplication.Spec.GitRepository.Branch = ""
				Expect(k8sClient.Update(ctx, testApplication)).To(Succeed())

				By("Reconciling the deployment twice")
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying pipeline uses 'main' as default branch")
				pipeline := &tektonv1.Pipeline{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name:      expectedPipelineName,
						Namespace: testNamespace.Name,
					}, pipeline)
				}).Should(Succeed())

				branchParam := pipeline.Spec.Params[1]
				Expect(branchParam.Name).To(Equal("git-branch"))
				Expect(branchParam.Default.StringVal).To(Equal("main"))
			})

			It("should not create duplicate pipeline on subsequent reconciles", func() {
				By("First reconcile - creating pipeline")
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying pipeline exists")
				pipeline := &tektonv1.Pipeline{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name:      expectedPipelineName,
						Namespace: testNamespace.Name,
					}, pipeline)
				}).Should(Succeed())

				originalCreationTime := pipeline.CreationTimestamp

				By("Second reconcile - should not create duplicate")
				err = reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying pipeline was not recreated")
				err = k8sClient.Get(ctx, types.NamespacedName{
					Name:      expectedPipelineName,
					Namespace: testNamespace.Name,
				}, pipeline)
				Expect(err).NotTo(HaveOccurred())
				Expect(pipeline.CreationTimestamp).To(Equal(originalCreationTime))
			})

			It("should handle public repositories correctly", func() {
				By("Updating application to have public access")
				testApplication.Spec.GitRepository.PublicAccess = true
				testApplication.Spec.GitRepository.SecretRef = nil
				Expect(k8sClient.Update(ctx, testApplication)).To(Succeed())

				By("Reconciling the deployment")
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying pipeline uses empty token secret for public repos")
				pipeline := &tektonv1.Pipeline{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name:      expectedPipelineName,
						Namespace: testNamespace.Name,
					}, pipeline)
				}).Should(Succeed())

				task := pipeline.Spec.Tasks[0]
				tokenParam := findTaskParam(task.Params, "token-secret")
				Expect(tokenParam).NotTo(BeNil())
				Expect(tokenParam.Value.StringVal).To(Equal("")) // Empty for public repos
			})
		})

		Context("When deployment references non-GitRepository application", func() {
			BeforeEach(func() {
				// Create DockerImage application
				testApplication = &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.GetApplicationResourceName("app-uuid-456"),
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "app-uuid-456",
							"platform.kibaship.com/slug":             "dockerapp",
							"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
							"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
						Type:           platformv1alpha1.ApplicationTypeDockerImage,
						DockerImage: &platformv1alpha1.DockerImageConfig{
							Image: "nginx:latest",
						},
					},
				}
				Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

				testDeployment = &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.GetDeploymentResourceName("deployment-uuid-456"),
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-456",
							"platform.kibaship.com/slug":             "web",
							"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
							"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
							"platform.kibaship.com/application-uuid": "app-uuid-456",
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: testApplication.Name},
					},
				}
				Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())
			})

			It("should not create a pipeline for non-GitRepository applications", func() {
				By("Reconciling the deployment")
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying no pipeline was created")
				pipeline := &tektonv1.Pipeline{}
				err = k8sClient.Get(ctx, types.NamespacedName{
					Name:      expectedPipelineName,
					Namespace: testNamespace.Name,
				}, pipeline)
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})
		})

		It("should create PipelineRun when deployment has GitRepository configuration", func() {
			// Create fresh application for this test
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetApplicationResourceName("app-uuid-testproj-testapp"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "app-uuid-testproj-testapp",
						"platform.kibaship.com/slug":             "testapp",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
					Type:           platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "user/test-repo",
						Branch:       "main",
						PublicAccess: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			// Create fresh deployment for this test
			deployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetDeploymentResourceName("deployment-uuid-testproj-testdeploy"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-testproj-testdeploy",
						"platform.kibaship.com/slug":             "testdeploy",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/application-uuid": "app-uuid-testproj-testapp",
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "abc123def456",
						Branch:    "feature-branch",
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			// Reconcile deployment twice (first adds finalizer, second creates resources)
			err := reconcileDeploymentTwice(ctx, deploymentReconciler, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pipeline was created
			pipeline := &tektonv1.Pipeline{}
			expectedPipelineName := fmt.Sprintf("pipeline-%s", "deployment-uuid-testproj-testdeploy")
			pipelineKey := types.NamespacedName{Name: expectedPipelineName, Namespace: testNamespace.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineKey, pipeline)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			// Verify Pipeline has correct configuration
			Expect(pipeline.Spec.Params).To(HaveLen(2))
			Expect(pipeline.Spec.Params[0].Name).To(Equal("git-commit"))
			Expect(pipeline.Spec.Params[1].Name).To(Equal("git-branch"))

			// Verify PipelineRun was created
			pipelineRun := &tektonv1.PipelineRun{}
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-%s-%d", "deployment-uuid-testproj-testdeploy", deployment.Generation)
			pipelineRunKey := types.NamespacedName{Name: expectedPipelineRunName, Namespace: testNamespace.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineRunKey, pipelineRun)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			// Verify PipelineRun has correct configuration
			Expect(pipelineRun.Spec.PipelineRef.Name).To(Equal(expectedPipelineName))
			Expect(pipelineRun.Spec.Params).To(HaveLen(2))
			Expect(pipelineRun.Spec.Params[0].Name).To(Equal("git-commit"))
			Expect(pipelineRun.Spec.Params[0].Value.StringVal).To(Equal("abc123def456"))
			Expect(pipelineRun.Spec.Params[1].Name).To(Equal("git-branch"))
			Expect(pipelineRun.Spec.Params[1].Value.StringVal).To(Equal("feature-branch"))

			// Verify service account name
			expectedServiceAccountName := fmt.Sprintf("%s-sa", testProject.Name)
			Expect(pipelineRun.Spec.TaskRunTemplate.ServiceAccountName).To(Equal(expectedServiceAccountName))

			// Verify workspace configuration (includes workspace PVC + registry secrets + env vars)
			Expect(pipelineRun.Spec.Workspaces).To(HaveLen(4))

			// Find workspace PVC (first workspace with VolumeClaimTemplate)
			var workspacePVC *tektonv1.WorkspaceBinding
			for i := range pipelineRun.Spec.Workspaces {
				if pipelineRun.Spec.Workspaces[i].VolumeClaimTemplate != nil {
					workspacePVC = &pipelineRun.Spec.Workspaces[i]
					break
				}
			}
			Expect(workspacePVC).NotTo(BeNil())
			Expect(workspacePVC.Name).To(Equal(fmt.Sprintf("workspace-%s", "deployment-uuid-testproj-testdeploy")))

			// Verify PVC storage allocation is 24GB
			pvc := workspacePVC.VolumeClaimTemplate
			storageRequest := pvc.Spec.Resources.Requests["storage"]
			Expect(storageRequest.String()).To(Equal("24Gi"))

			// Verify storage class is set to storage-replica-1
			Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
			Expect(*pvc.Spec.StorageClassName).To(Equal(config.StorageClassReplica1))
		})

		It("should allocate 24GB storage for PipelineRun workspace PVC", func() {
			// Create fresh application for this test
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetApplicationResourceName("app-uuid-teststorage-testapp"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "app-uuid-teststorage-testapp",
						"platform.kibaship.com/slug":             "testapp",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
					Type:           platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "user/test-repo",
						Branch:       "main",
						PublicAccess: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			// Create fresh deployment for this test
			deployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetDeploymentResourceName("deployment-uuid-teststorage-storage"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-teststorage-storage",
						"platform.kibaship.com/slug":             "storage",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/application-uuid": "app-uuid-teststorage-testapp",
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "abc123def456",
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			// Reconcile deployment twice
			err := reconcileDeploymentTwice(ctx, deploymentReconciler, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify PipelineRun was created
			pipelineRun := &tektonv1.PipelineRun{}
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-deployment-uuid-teststorage-storage-%d", deployment.Generation)
			pipelineRunKey := types.NamespacedName{Name: expectedPipelineRunName, Namespace: testNamespace.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineRunKey, pipelineRun)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			// Verify PVC storage allocation is exactly 24GB (includes workspace PVC + registry secrets + env vars)
			Expect(pipelineRun.Spec.Workspaces).To(HaveLen(4))

			// Find workspace PVC (first workspace with VolumeClaimTemplate)
			var workspacePVC *tektonv1.WorkspaceBinding
			for i := range pipelineRun.Spec.Workspaces {
				if pipelineRun.Spec.Workspaces[i].VolumeClaimTemplate != nil {
					workspacePVC = &pipelineRun.Spec.Workspaces[i]
					break
				}
			}
			Expect(workspacePVC).NotTo(BeNil())
			Expect(workspacePVC.VolumeClaimTemplate).NotTo(BeNil())

			pvc := workspacePVC.VolumeClaimTemplate
			storageRequest := pvc.Spec.Resources.Requests["storage"]
			Expect(storageRequest.String()).To(Equal("24Gi"))

			// Verify storage class is set to storage-replica-1
			Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
			Expect(*pvc.Spec.StorageClassName).To(Equal(config.StorageClassReplica1))

			// Also verify access mode is ReadWriteOnce
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
		})

		It("should use storage-replica-1 storage class for PipelineRun workspace PVC", func() {
			// Create fresh application for this test
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetApplicationResourceName("app-uuid-teststorageclass-testapp"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "app-uuid-teststorageclass-testapp",
						"platform.kibaship.com/slug":             "testapp",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
					Type:           platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "user/test-repo",
						Branch:       "main",
						PublicAccess: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			// Create fresh deployment for this test
			deployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetDeploymentResourceName("deployment-uuid-teststorageclass-storageclass"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-teststorageclass-storageclass",
						"platform.kibaship.com/slug":             "storageclass",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/application-uuid": "app-uuid-teststorageclass-testapp",
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "abc123def456",
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			// Reconcile deployment twice
			err := reconcileDeploymentTwice(ctx, deploymentReconciler, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify PipelineRun was created
			pipelineRun := &tektonv1.PipelineRun{}
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-deployment-uuid-teststorageclass-storageclass-%d", deployment.Generation)
			pipelineRunKey := types.NamespacedName{Name: expectedPipelineRunName, Namespace: testNamespace.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineRunKey, pipelineRun)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			// Focus specifically on storage class verification (includes workspace PVC + registry secrets + env vars)
			Expect(pipelineRun.Spec.Workspaces).To(HaveLen(4))

			// Find workspace PVC (first workspace with VolumeClaimTemplate)
			var workspacePVC *tektonv1.WorkspaceBinding
			for i := range pipelineRun.Spec.Workspaces {
				if pipelineRun.Spec.Workspaces[i].VolumeClaimTemplate != nil {
					workspacePVC = &pipelineRun.Spec.Workspaces[i]
					break
				}
			}
			Expect(workspacePVC).NotTo(BeNil())
			Expect(workspacePVC.VolumeClaimTemplate).NotTo(BeNil())

			pvc := workspacePVC.VolumeClaimTemplate

			// Primary assertion: storage class must be storage-replica-1
			Expect(pvc.Spec.StorageClassName).NotTo(BeNil(), "StorageClassName should not be nil")
			Expect(*pvc.Spec.StorageClassName).To(Equal(config.StorageClassReplica1), "StorageClassName should be 'storage-replica-1'")
		})

		It("should use application branch when deployment branch is not specified", func() {
			// Create fresh application for this test
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetApplicationResourceName("app-uuid-testbranch-testapp"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "app-uuid-testbranch-testapp",
						"platform.kibaship.com/slug":             "testapp",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
					Type:           platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "user/test-repo",
						Branch:       "develop",
						PublicAccess: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			// Create fresh deployment without branch specification
			deployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetDeploymentResourceName("deployment-uuid-testbranch-testbranch"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-testbranch-testbranch",
						"platform.kibaship.com/slug":             "testbranch",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/application-uuid": "app-uuid-testbranch-testapp",
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "abc123def456",
						// Branch is omitted - should use application's branch
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			// Reconcile deployment twice
			err := reconcileDeploymentTwice(ctx, deploymentReconciler, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify PipelineRun uses application's branch
			pipelineRun := &tektonv1.PipelineRun{}
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-deployment-uuid-testbranch-testbranch-%d", deployment.Generation)
			pipelineRunKey := types.NamespacedName{Name: expectedPipelineRunName, Namespace: testNamespace.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineRunKey, pipelineRun)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			// Find git-branch parameter
			var branchParam *tektonv1.Param
			for i, param := range pipelineRun.Spec.Params {
				if param.Name == "git-branch" {
					branchParam = &pipelineRun.Spec.Params[i]
					break
				}
			}
			Expect(branchParam).NotTo(BeNil())
			Expect(branchParam.Value.StringVal).To(Equal("develop"))
		})

		It("should not create duplicate PipelineRun for same generation", func() {
			// Create fresh application for this test
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetApplicationResourceName("app-uuid-testdup-testapp"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "app-uuid-testdup-testapp",
						"platform.kibaship.com/slug":             "testapp",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
					Type:           platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "user/test-repo",
						Branch:       "main",
						PublicAccess: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			// Create fresh deployment for this test
			deployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetDeploymentResourceName("deployment-uuid-testdup-testdup"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-testdup-testdup",
						"platform.kibaship.com/slug":             "testdup",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/application-uuid": "app-uuid-testdup-testapp",
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
					GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
						CommitSHA: "abc123def456",
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			// Reconcile deployment twice
			err := reconcileDeploymentTwice(ctx, deploymentReconciler, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify first PipelineRun was created
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-deployment-uuid-testdup-testdup-%d", deployment.Generation)
			pipelineRunKey := types.NamespacedName{Name: expectedPipelineRunName, Namespace: testNamespace.Name}
			pipelineRun := &tektonv1.PipelineRun{}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineRunKey, pipelineRun)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			// Store the creation timestamp
			firstCreationTime := pipelineRun.CreationTimestamp

			// Reconcile again - should not create duplicate
			_, err = deploymentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      deployment.Name,
					Namespace: deployment.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify PipelineRun still exists and wasn't recreated
			Expect(k8sClient.Get(ctx, pipelineRunKey, pipelineRun)).To(Succeed())
			Expect(pipelineRun.CreationTimestamp).To(Equal(firstCreationTime))
		})

		It("should require GitRepository configuration for GitRepository applications", func() {
			// Create fresh application for this test
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetApplicationResourceName("app-uuid-testrequired-testapp"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "app-uuid-testrequired-testapp",
						"platform.kibaship.com/slug":             "testapp",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
					Type:           platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "user/test-repo",
						Branch:       "main",
						PublicAccess: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			// Create fresh deployment WITHOUT GitRepository configuration
			deployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetDeploymentResourceName("deployment-uuid-testrequired-testrequired"),
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-testrequired-testrequired",
						"platform.kibaship.com/slug":             "testrequired",
						"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
						"platform.kibaship.com/application-uuid": "app-uuid-testrequired-testapp",
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
					// GitRepository is intentionally nil to test validation
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			// First reconcile to add finalizer
			_, err := deploymentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      deployment.Name,
					Namespace: deployment.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile should fail due to missing GitRepository configuration
			_, err = deploymentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      deployment.Name,
					Namespace: deployment.Namespace,
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("GitRepository configuration is required"))
		})

		Context("Branch Parameter Testing", func() {
			It("should handle different git providers correctly", func() {
				testCases := []struct {
					provider    platformv1alpha1.GitProvider
					repository  string
					expectedURL string
				}{
					{platformv1alpha1.GitProviderGitHub, "user/repo", "https://github.com/user/repo"},
					{platformv1alpha1.GitProviderGitLab, "group/project", "https://gitlab.com/group/project"},
					{platformv1alpha1.GitProviderBitbucket, "workspace/repo", "https://bitbucket.com/workspace/repo"},
				}

				for i, tc := range testCases {
					By("Testing " + string(tc.provider))

					// Create application with specific provider
					app := &platformv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("application-app-uuid-provider%d", i),
							Namespace: testNamespace.Name,
							Labels: map[string]string{
								"platform.kibaship.com/uuid":             fmt.Sprintf("app-uuid-provider%d", i),
								"platform.kibaship.com/slug":             fmt.Sprintf("provider%d", i),
								"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
								"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
							},
						},
						Spec: platformv1alpha1.ApplicationSpec{
							EnvironmentRef: corev1.LocalObjectReference{Name: testEnvironment.Name},
							Type:           platformv1alpha1.ApplicationTypeGitRepository,
							GitRepository: &platformv1alpha1.GitRepositoryConfig{
								Provider:   tc.provider,
								Repository: tc.repository,
								Branch:     "main",
							},
						},
					}
					Expect(k8sClient.Create(ctx, app)).To(Succeed())

					deployment := &platformv1alpha1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("deployment-deployment-uuid-provider%d", i),
							Namespace: testNamespace.Name,
							Labels: map[string]string{
								"platform.kibaship.com/uuid":             fmt.Sprintf("deployment-uuid-provider%d", i),
								"platform.kibaship.com/slug":             fmt.Sprintf("web%d", i),
								"platform.kibaship.com/environment-uuid": testEnvironment.Labels[validation.LabelResourceUUID],
								"platform.kibaship.com/project-uuid":     testProject.Labels[validation.LabelResourceUUID],
								"platform.kibaship.com/application-uuid": fmt.Sprintf("app-uuid-provider%d", i),
							},
						},
						Spec: platformv1alpha1.DeploymentSpec{
							ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
							GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{
								CommitSHA: "abc123def456",
								Branch:    "main",
							},
						},
					}
					Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

					err := reconcileDeploymentTwice(ctx, deploymentReconciler, deployment)
					Expect(err).NotTo(HaveOccurred())

					// Verify pipeline was created with correct URL
					expectedPipelineName := fmt.Sprintf("pipeline-deployment-uuid-provider%d", i)
					pipeline := &tektonv1.Pipeline{}
					Eventually(func() error {
						return k8sClient.Get(ctx, types.NamespacedName{
							Name:      expectedPipelineName,
							Namespace: testNamespace.Name,
						}, pipeline)
					}).Should(Succeed())

					task := pipeline.Spec.Tasks[0]
					urlParam := findTaskParam(task.Params, "url")
					Expect(urlParam).NotTo(BeNil())
					Expect(urlParam.Value.StringVal).To(Equal(tc.expectedURL))

					// Clean up
					Expect(k8sClient.Delete(ctx, deployment)).To(Succeed())
					Expect(k8sClient.Delete(ctx, app)).To(Succeed())
				}
			})
		})
	})

})

// Helper function to reconcile deployment twice (finalizer + actual logic)
func reconcileDeploymentTwice(ctx context.Context, reconciler *DeploymentReconciler, deployment *platformv1alpha1.Deployment) error {
	// First reconcile adds finalizer
	_, err := reconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
		},
	})
	if err != nil {
		return err
	}

	// Second reconcile performs actual logic
	_, err = reconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
		},
	})
	return err
}

// Helper function to find a task parameter by name
func findTaskParam(params []tektonv1.Param, name string) *tektonv1.Param {
	for i, param := range params {
		if param.Name == name {
			return &params[i]
		}
	}
	return nil
}
