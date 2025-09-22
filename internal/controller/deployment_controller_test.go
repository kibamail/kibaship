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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/config"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	expectedPipelineName     = "pipeline-web-kibaship-com"
	expectedMySQLSecretName  = "mysql-secret-deploy1-kibaship-com"
	expectedMySQLClusterName = "mysql-deploy1"
)

var _ = Describe("Deployment Controller", func() {
	var (
		ctx                  context.Context
		deploymentReconciler *DeploymentReconciler
		testNamespace        *corev1.Namespace
		testProject          *platformv1alpha1.Project
		testApplication      *platformv1alpha1.Application
		testDeployment       *platformv1alpha1.Deployment
	)

	BeforeEach(func() {
		ctx = context.Background()
		deploymentReconciler = &DeploymentReconciler{
			Client:           k8sClient,
			Scheme:           k8sClient.Scheme(),
			NamespaceManager: NewNamespaceManager(k8sClient),
			StreamPublisher:  nil,
		}

		// Create test namespace with unique name
		namespaceName := fmt.Sprintf("test-deployment-ns-%d", rand.Int32())
		testNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}
		Expect(k8sClient.Create(ctx, testNamespace)).To(Succeed())

		// Create test project
		testProject = &platformv1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-test123-kibaship-com",
				Labels: map[string]string{
					"platform.kibaship.com/uuid":           "550e8400-e29b-41d4-a716-446655440000",
					"platform.kibaship.com/slug":           "test123",
					"platform.kibaship.com/workspace-uuid": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				},
			},
			Spec: platformv1alpha1.ProjectSpec{},
		}
		Expect(k8sClient.Create(ctx, testProject)).To(Succeed())
	})

	AfterEach(func() {
		// Clean up test resources
		if testDeployment != nil {
			_ = k8sClient.Delete(ctx, testDeployment)
		}
		if testApplication != nil {
			_ = k8sClient.Delete(ctx, testApplication)
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
						Name:      "project-test123-app-myapp-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":         "app-uuid-123",
							"platform.kibaship.com/slug":         "myapp",
							"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
						Type:       platformv1alpha1.ApplicationTypeGitRepository,
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
						Name:      "project-test123-app-myapp-deployment-web-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-123",
							"platform.kibaship.com/slug":             "web",
							"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
				Expect(pipeline.Spec.Tasks).To(HaveLen(1))
				task := pipeline.Spec.Tasks[0]
				Expect(task.Name).To(Equal("clone-repository"))

				// Verify task parameters
				taskParams := task.Params
				Expect(taskParams).To(HaveLen(5))

				urlParam := findTaskParam(taskParams, "url")
				Expect(urlParam).NotTo(BeNil())
				Expect(urlParam.Value.StringVal).To(Equal("https://github.com/user/test-repo"))

				branchTaskParam := findTaskParam(taskParams, "branch")
				Expect(branchTaskParam).NotTo(BeNil())
				Expect(branchTaskParam.Value.StringVal).To(Equal("$(params.git-branch)"))

				commitTaskParam := findTaskParam(taskParams, "commit")
				Expect(commitTaskParam).NotTo(BeNil())
				Expect(commitTaskParam.Value.StringVal).To(Equal("$(params.git-commit)"))

				tokenParam := findTaskParam(taskParams, "token-secret")
				Expect(tokenParam).NotTo(BeNil())
				Expect(tokenParam.Value.StringVal).To(Equal("git-secret"))

				By("Verifying pipeline has correct labels and annotations")
				Expect(pipeline.Labels["app.kubernetes.io/name"]).To(Equal("project-test123"))
				Expect(pipeline.Labels["app.kubernetes.io/managed-by"]).To(Equal("kibaship-operator"))
				Expect(pipeline.Labels["project.kibaship.com/slug"]).To(Equal("test123"))
				Expect(pipeline.Labels["tekton.dev/pipeline"]).To(Equal("git-repository-clone"))
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
						Name:      "project-test123-app-dockerapp-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":         "app-uuid-456",
							"platform.kibaship.com/slug":         "dockerapp",
							"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
						Type:       platformv1alpha1.ApplicationTypeDockerImage,
						DockerImage: &platformv1alpha1.DockerImageConfig{
							Image: "nginx:latest",
						},
					},
				}
				Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

				testDeployment = &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-dockerapp-deployment-web-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-456",
							"platform.kibaship.com/slug":             "web",
							"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
					Name:      "project-testproj-app-testapp-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":         "app-uuid-testproj-testapp",
						"platform.kibaship.com/slug":         "testapp",
						"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
					Type:       platformv1alpha1.ApplicationTypeGitRepository,
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
					Name:      "project-testproj-app-testapp-deployment-testdeploy-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-testproj-testdeploy",
						"platform.kibaship.com/slug":             "testdeploy",
						"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
			expectedPipelineName := "pipeline-testdeploy-kibaship-com"
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
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-testdeploy-%d-kibaship-com", deployment.Generation)
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
			Expect(pipelineRun.Spec.TaskRunTemplate.ServiceAccountName).To(Equal("project-test123-sa-kibaship-com"))

			// Verify workspace configuration
			Expect(pipelineRun.Spec.Workspaces).To(HaveLen(1))
			Expect(pipelineRun.Spec.Workspaces[0].Name).To(Equal("workspace-testdeploy-kibaship-com"))
			Expect(pipelineRun.Spec.Workspaces[0].VolumeClaimTemplate).NotTo(BeNil())

			// Verify PVC storage allocation is 24GB
			pvc := pipelineRun.Spec.Workspaces[0].VolumeClaimTemplate
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
					Name:      "project-teststorage-app-testapp-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":         "app-uuid-teststorage-testapp",
						"platform.kibaship.com/slug":         "testapp",
						"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
					Type:       platformv1alpha1.ApplicationTypeGitRepository,
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
					Name:      "project-teststorage-app-testapp-deployment-storage-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-teststorage-storage",
						"platform.kibaship.com/slug":             "storage",
						"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-storage-%d-kibaship-com", deployment.Generation)
			pipelineRunKey := types.NamespacedName{Name: expectedPipelineRunName, Namespace: testNamespace.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineRunKey, pipelineRun)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			// Verify PVC storage allocation is exactly 24GB
			Expect(pipelineRun.Spec.Workspaces).To(HaveLen(1))
			workspace := pipelineRun.Spec.Workspaces[0]
			Expect(workspace.VolumeClaimTemplate).NotTo(BeNil())

			pvc := workspace.VolumeClaimTemplate
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
					Name:      "project-teststorageclass-app-testapp-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":         "app-uuid-teststorageclass-testapp",
						"platform.kibaship.com/slug":         "testapp",
						"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
					Type:       platformv1alpha1.ApplicationTypeGitRepository,
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
					Name:      "project-teststorageclass-app-testapp-deployment-storageclass-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-teststorageclass-storageclass",
						"platform.kibaship.com/slug":             "storageclass",
						"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-storageclass-%d-kibaship-com", deployment.Generation)
			pipelineRunKey := types.NamespacedName{Name: expectedPipelineRunName, Namespace: testNamespace.Name}
			Eventually(func() error {
				return k8sClient.Get(ctx, pipelineRunKey, pipelineRun)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			// Focus specifically on storage class verification
			Expect(pipelineRun.Spec.Workspaces).To(HaveLen(1))
			workspace := pipelineRun.Spec.Workspaces[0]
			Expect(workspace.VolumeClaimTemplate).NotTo(BeNil())

			pvc := workspace.VolumeClaimTemplate

			// Primary assertion: storage class must be storage-replica-1
			Expect(pvc.Spec.StorageClassName).NotTo(BeNil(), "StorageClassName should not be nil")
			Expect(*pvc.Spec.StorageClassName).To(Equal(config.StorageClassReplica1), "StorageClassName should be 'storage-replica-1'")
		})

		It("should use application branch when deployment branch is not specified", func() {
			// Create fresh application for this test
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-testbranch-app-testapp-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":         "app-uuid-testbranch-testapp",
						"platform.kibaship.com/slug":         "testapp",
						"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
					Type:       platformv1alpha1.ApplicationTypeGitRepository,
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
					Name:      "project-testbranch-app-testapp-deployment-testbranch-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-testbranch-testbranch",
						"platform.kibaship.com/slug":             "testbranch",
						"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-testbranch-%d-kibaship-com", deployment.Generation)
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
					Name:      "project-testdup-app-testapp-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":         "app-uuid-testdup-testapp",
						"platform.kibaship.com/slug":         "testapp",
						"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
					Type:       platformv1alpha1.ApplicationTypeGitRepository,
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
					Name:      "project-testdup-app-testapp-deployment-testdup-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-testdup-testdup",
						"platform.kibaship.com/slug":             "testdup",
						"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
			expectedPipelineRunName := fmt.Sprintf("pipeline-run-testdup-%d-kibaship-com", deployment.Generation)
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
					Name:      "project-testrequired-app-testapp-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":         "app-uuid-testrequired-testapp",
						"platform.kibaship.com/slug":         "testapp",
						"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
					Type:       platformv1alpha1.ApplicationTypeGitRepository,
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
					Name:      "project-testrequired-app-testapp-deployment-testrequired-kibaship-com",
					Namespace: testNamespace.Name,
					Labels: map[string]string{
						"platform.kibaship.com/uuid":             "deployment-uuid-testrequired-testrequired",
						"platform.kibaship.com/slug":             "testrequired",
						"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
							Name:      fmt.Sprintf("project-test123-app-provider%d-kibaship-com", i),
							Namespace: testNamespace.Name,
							Labels: map[string]string{
								"platform.kibaship.com/uuid":         fmt.Sprintf("app-uuid-provider%d", i),
								"platform.kibaship.com/slug":         fmt.Sprintf("provider%d", i),
								"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
							},
						},
						Spec: platformv1alpha1.ApplicationSpec{
							ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
							Type:       platformv1alpha1.ApplicationTypeGitRepository,
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
							Name:      fmt.Sprintf("project-test123-app-provider%d-deployment-web%d-kibaship-com", i, i),
							Namespace: testNamespace.Name,
							Labels: map[string]string{
								"platform.kibaship.com/uuid":             fmt.Sprintf("deployment-uuid-provider%d", i),
								"platform.kibaship.com/slug":             fmt.Sprintf("web%d", i),
								"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
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
					expectedPipelineName := fmt.Sprintf("pipeline-web%d-kibaship-com", i)
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

	Describe("MySQL Deployment", func() {
		Context("When deployment references MySQL application", func() {
			var testMySQLApp *platformv1alpha1.Application

			BeforeEach(func() {
				// Create MySQL application
				testMySQLApp = &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysqlapp-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":         "app-uuid-mysql-mysqlapp",
							"platform.kibaship.com/slug":         "mysqlapp",
							"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
						Type:       platformv1alpha1.ApplicationTypeMySQL,
						MySQL: &platformv1alpha1.MySQLConfig{
							Version: "8.0.28",
						},
					},
				}
				Expect(k8sClient.Create(ctx, testMySQLApp)).To(Succeed())
			})

			AfterEach(func() {
				if testMySQLApp != nil {
					_ = k8sClient.Delete(ctx, testMySQLApp)
				}
			})

			It("should create MySQL credentials secret and InnoDBCluster for new deployment", func() {
				// Create MySQL deployment
				testDeployment = &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysqlapp-deployment-deploy1-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-mysql-deploy1",
							"platform.kibaship.com/slug":             "deploy1",
							"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
							"platform.kibaship.com/application-uuid": "app-uuid-mysql-mysqlapp",
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: testMySQLApp.Name},
					},
				}
				Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

				// Reconcile deployment twice (first adds finalizer, second creates resources)
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Verify MySQL credentials secret was created
				secret := &corev1.Secret{}
				expectedSecretName := expectedMySQLSecretName
				secretKey := types.NamespacedName{Name: expectedSecretName, Namespace: testNamespace.Name}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, time.Second*10, time.Millisecond*250).Should(Succeed())

				// Verify secret contains correct keys (Data field when read back from cluster)
				Expect(string(secret.Data["rootUser"])).To(Equal("root"))
				Expect(string(secret.Data["rootHost"])).To(Equal("%"))
				Expect(string(secret.Data["rootPassword"])).To(HaveLen(32))                   // 32 character password
				Expect(string(secret.Data["rootPassword"])).To(MatchRegexp("^[a-zA-Z0-9]+$")) // Alphanumeric only

				// Verify secret has correct labels
				Expect(secret.Labels["app.kubernetes.io/name"]).To(Equal("project-test123"))
				Expect(secret.Labels["app.kubernetes.io/managed-by"]).To(Equal("kibaship"))
				Expect(secret.Labels["app.kubernetes.io/component"]).To(Equal("mysql-credentials"))
				Expect(secret.Labels["project.kibaship.com/slug"]).To(Equal("test123"))

				// Verify InnoDBCluster was created
				cluster := &unstructured.Unstructured{}
				cluster.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "mysql.oracle.com",
					Version: "v2",
					Kind:    "InnoDBCluster",
				})
				expectedClusterName := expectedMySQLClusterName
				clusterKey := types.NamespacedName{Name: expectedClusterName, Namespace: testNamespace.Name}
				Eventually(func() error {
					return k8sClient.Get(ctx, clusterKey, cluster)
				}, time.Second*10, time.Millisecond*250).Should(Succeed())

				// Verify InnoDBCluster has correct configuration
				spec, _, err := unstructured.NestedMap(cluster.Object, "spec")
				Expect(err).NotTo(HaveOccurred())

				Expect(spec["secretName"]).To(Equal(expectedSecretName))
				Expect(spec["tlsUseSelfSigned"]).To(BeTrue())
				Expect(spec["instances"]).To(Equal(int64(1)))
				Expect(spec["version"]).To(Equal("8.0.28"))

				// Verify router configuration
				router, _, err := unstructured.NestedMap(spec, "router")
				Expect(err).NotTo(HaveOccurred())
				Expect(router["instances"]).To(Equal(int64(0)))

				// Verify storage configuration
				pvcTemplate, _, err := unstructured.NestedMap(spec, "datadirVolumeClaimTemplate")
				Expect(err).NotTo(HaveOccurred())
				pvcSpec, _, err := unstructured.NestedMap(pvcTemplate, "spec")
				Expect(err).NotTo(HaveOccurred())
				Expect(pvcSpec["storageClassName"]).To(Equal(config.StorageClassReplica2))

				resources, _, err := unstructured.NestedMap(pvcSpec, "resources")
				Expect(err).NotTo(HaveOccurred())
				requests, _, err := unstructured.NestedMap(resources, "requests")
				Expect(err).NotTo(HaveOccurred())
				Expect(requests["storage"]).To(Equal("512Mi"))

				// Verify InnoDBCluster has correct labels
				Expect(cluster.GetLabels()["app.kubernetes.io/name"]).To(Equal("project-test123"))
				Expect(cluster.GetLabels()["app.kubernetes.io/managed-by"]).To(Equal("kibaship"))
				Expect(cluster.GetLabels()["app.kubernetes.io/component"]).To(Equal("mysql-database"))
				Expect(cluster.GetLabels()["project.kibaship.com/slug"]).To(Equal("test123"))
			})

			It("should handle existing deployments gracefully", func() {
				// Create first MySQL deployment
				firstDeployment := &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysqlapp-deployment-deploy1-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-mysql-first-deploy1",
							"platform.kibaship.com/slug":             "deploy1",
							"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
							"platform.kibaship.com/application-uuid": "app-uuid-mysql-mysqlapp",
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: testMySQLApp.Name},
					},
				}
				Expect(k8sClient.Create(ctx, firstDeployment)).To(Succeed())

				// Reconcile first deployment
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, firstDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Verify resources were created for first deployment
				secret := &corev1.Secret{}
				expectedSecretName := expectedMySQLSecretName
				secretKey := types.NamespacedName{Name: expectedSecretName, Namespace: testNamespace.Name}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, time.Second*10, time.Millisecond*250).Should(Succeed())

				cluster := &unstructured.Unstructured{}
				cluster.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "mysql.oracle.com",
					Version: "v2",
					Kind:    "InnoDBCluster",
				})
				expectedClusterName := expectedMySQLClusterName
				clusterKey := types.NamespacedName{Name: expectedClusterName, Namespace: testNamespace.Name}
				Eventually(func() error {
					return k8sClient.Get(ctx, clusterKey, cluster)
				}, time.Second*10, time.Millisecond*250).Should(Succeed())

				// Create second MySQL deployment for the same application
				testDeployment = &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysqlapp-deployment-deploy2-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-mysql-deploy2",
							"platform.kibaship.com/slug":             "deploy2",
							"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
							"platform.kibaship.com/application-uuid": "app-uuid-mysql-mysqlapp",
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: testMySQLApp.Name},
					},
				}
				Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

				// Reconcile second deployment
				err = reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Verify that no duplicate resources were created (should reuse existing ones)
				secretList := &corev1.SecretList{}
				Expect(k8sClient.List(ctx, secretList, client.InNamespace(testNamespace.Name))).To(Succeed())

				mysqlSecretsCount := 0
				for _, s := range secretList.Items {
					if s.Labels["app.kubernetes.io/component"] == "mysql-credentials" {
						mysqlSecretsCount++
					}
				}
				Expect(mysqlSecretsCount).To(Equal(1), "Should only have one MySQL credentials secret")

				clusterList := &unstructured.UnstructuredList{}
				clusterList.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "mysql.oracle.com",
					Version: "v2",
					Kind:    "InnoDBClusterList",
				})
				Expect(k8sClient.List(ctx, clusterList, client.InNamespace(testNamespace.Name))).To(Succeed())
				Expect(clusterList.Items).To(HaveLen(1), "Should only have one InnoDBCluster")

				// Clean up first deployment
				_ = k8sClient.Delete(ctx, firstDeployment)
			})

			It("should use default MySQL version when none specified", func() {
				// Create MySQL application without version
				appWithoutVersion := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysql-no-version-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":         "app-uuid-mysql-no-version",
							"platform.kibaship.com/slug":         "mysql-no-version",
							"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
						Type:       platformv1alpha1.ApplicationTypeMySQL,
						MySQL:      &platformv1alpha1.MySQLConfig{}, // No version specified
					},
				}
				Expect(k8sClient.Create(ctx, appWithoutVersion)).To(Succeed())
				defer func() { _ = k8sClient.Delete(ctx, appWithoutVersion) }()

				// Create MySQL deployment
				testDeployment = &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysql-no-version-deployment-deploy1-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-mysql-no-version-deploy1",
							"platform.kibaship.com/slug":             "deploy1",
							"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
							"platform.kibaship.com/application-uuid": "app-uuid-mysql-no-version",
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: appWithoutVersion.Name},
					},
				}
				Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

				// Reconcile deployment
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Verify InnoDBCluster was created without version (should use operator default)
				cluster := &unstructured.Unstructured{}
				cluster.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "mysql.oracle.com",
					Version: "v2",
					Kind:    "InnoDBCluster",
				})
				expectedClusterName := expectedMySQLClusterName
				clusterKey := types.NamespacedName{Name: expectedClusterName, Namespace: testNamespace.Name}
				Eventually(func() error {
					return k8sClient.Get(ctx, clusterKey, cluster)
				}, time.Second*10, time.Millisecond*250).Should(Succeed())

				// Version should not be set, allowing the MySQL operator to use its default
				spec, _, err := unstructured.NestedMap(cluster.Object, "spec")
				Expect(err).NotTo(HaveOccurred())
				_, found := spec["version"]
				Expect(found).To(BeFalse(), "Version should not be set when not specified in application config")
			})

			It("should handle MySQL application without MySQL config", func() {
				// Create MySQL application without MySQL config
				appWithoutConfig := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysql-no-config-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":         "app-uuid-mysql-no-config",
							"platform.kibaship.com/slug":         "mysql-no-config",
							"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
						Type:       platformv1alpha1.ApplicationTypeMySQL,
						// MySQL config is nil
					},
				}
				Expect(k8sClient.Create(ctx, appWithoutConfig)).To(Succeed())
				defer func() { _ = k8sClient.Delete(ctx, appWithoutConfig) }()

				// Create MySQL deployment
				testDeployment = &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysql-no-config-deployment-deploy1-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-mysql-no-config-deploy1",
							"platform.kibaship.com/slug":             "deploy1",
							"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
							"platform.kibaship.com/application-uuid": "app-uuid-mysql-no-config",
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: appWithoutConfig.Name},
					},
				}
				Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

				// Reconcile deployment
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Verify resources were still created successfully
				secret := &corev1.Secret{}
				expectedSecretName := expectedMySQLSecretName
				secretKey := types.NamespacedName{Name: expectedSecretName, Namespace: testNamespace.Name}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, time.Second*10, time.Millisecond*250).Should(Succeed())

				cluster := &unstructured.Unstructured{}
				cluster.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "mysql.oracle.com",
					Version: "v2",
					Kind:    "InnoDBCluster",
				})
				expectedClusterName := expectedMySQLClusterName
				clusterKey := types.NamespacedName{Name: expectedClusterName, Namespace: testNamespace.Name}
				Eventually(func() error {
					return k8sClient.Get(ctx, clusterKey, cluster)
				}, time.Second*10, time.Millisecond*250).Should(Succeed())
			})
		})

		Context("Error Handling", func() {
			It("should handle deployment with non-standard name format using labels", func() {
				// Create MySQL application
				testMySQLApp := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-test123-app-mysqlapp-kibaship-com",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":         "app-uuid-mysql-error-handling",
							"platform.kibaship.com/slug":         "mysqlapp",
							"platform.kibaship.com/project-uuid": "550e8400-e29b-41d4-a716-446655440000",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						ProjectRef: corev1.LocalObjectReference{Name: testProject.Name},
						Type:       platformv1alpha1.ApplicationTypeMySQL,
					},
				}
				Expect(k8sClient.Create(ctx, testMySQLApp)).To(Succeed())
				defer func() { _ = k8sClient.Delete(ctx, testMySQLApp) }()

				// Create deployment with non-standard name format but proper labels
				testDeployment = &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-deployment-name",
						Namespace: testNamespace.Name,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":             "deployment-uuid-invalid-name",
							"platform.kibaship.com/slug":             "invalid",
							"platform.kibaship.com/project-uuid":     "550e8400-e29b-41d4-a716-446655440000",
							"platform.kibaship.com/application-uuid": "app-uuid-mysql-error-handling",
						},
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: testMySQLApp.Name},
					},
				}
				Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

				// Reconcile deployment - should succeed using label-based approach
				err := reconcileDeploymentTwice(ctx, deploymentReconciler, testDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Verify MySQL resources were created successfully
				secret := &corev1.Secret{}
				expectedSecretName := "mysql-secret-invalid-kibaship-com"
				secretKey := types.NamespacedName{Name: expectedSecretName, Namespace: testNamespace.Name}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, time.Second*10, time.Millisecond*250).Should(Succeed())
			})
		})
	})

	Describe("MySQL Utility Functions", func() {
		Context("Password generation", func() {
			It("should generate secure 32-character alphanumeric passwords", func() {
				password, err := generateSecurePassword()
				Expect(err).NotTo(HaveOccurred())
				Expect(password).To(HaveLen(32))
				Expect(password).To(MatchRegexp("^[a-zA-Z0-9]+$"))
			})

			It("should generate different passwords on each call", func() {
				password1, err := generateSecurePassword()
				Expect(err).NotTo(HaveOccurred())

				password2, err := generateSecurePassword()
				Expect(err).NotTo(HaveOccurred())

				Expect(password1).NotTo(Equal(password2))
			})
		})

		Context("Resource name generation", func() {
			It("should generate correct MySQL resource names", func() {
				testDeployment := &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"platform.kibaship.com/slug": "testdeploy",
						},
					},
				}
				secretName, clusterName := generateMySQLResourceNames(testDeployment, "testproject", "myapp")

				Expect(secretName).To(Equal("mysql-secret-testdeploy-kibaship-com"))
				Expect(clusterName).To(Equal("mysql-testdeploy"))
			})
		})

		Context("Existing deployment detection", func() {
			It("should detect existing deployments for same application", func() {
				app := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-app",
					},
				}

				existingDeployment := platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-deployment",
						Namespace: "test-ns",
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: "test-app"},
					},
				}

				currentDeployment := &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "current-deployment",
						Namespace: "test-ns",
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: "test-app"},
					},
				}

				deployments := []platformv1alpha1.Deployment{existingDeployment}
				hasExisting := checkForExistingMySQLDeployments(deployments, currentDeployment, app)

				Expect(hasExisting).To(BeTrue())
			})

			It("should not detect itself as existing deployment", func() {
				app := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-app",
					},
				}

				currentDeployment := &platformv1alpha1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "current-deployment",
						Namespace: "test-ns",
					},
					Spec: platformv1alpha1.DeploymentSpec{
						ApplicationRef: corev1.LocalObjectReference{Name: "test-app"},
					},
				}

				deployments := []platformv1alpha1.Deployment{*currentDeployment}
				hasExisting := checkForExistingMySQLDeployments(deployments, currentDeployment, app)

				Expect(hasExisting).To(BeFalse())
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
