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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

var _ = Describe("Application Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		application := &platformv1alpha1.Application{}

		BeforeEach(func() {
			By("creating a test project with UUID")
			testProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-project",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{
					ApplicationTypes: platformv1alpha1.ApplicationTypesConfig{},
					Volumes:          platformv1alpha1.VolumeConfig{},
				},
			}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-project", Namespace: "default"}, &platformv1alpha1.Project{})
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, testProject)).To(Succeed())
			}

			By("creating the custom resource for the Kind Application")
			err = k8sClient.Get(ctx, typeNamespacedName, application)
			if err != nil && errors.IsNotFound(err) {
				resource := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
						Labels: map[string]string{
							"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440001",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						ProjectRef: corev1.LocalObjectReference{
							Name: "test-project",
						},
						Type: platformv1alpha1.ApplicationTypeDockerImage,
						DockerImage: &platformv1alpha1.DockerImageConfig{
							Image: "nginx:latest",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &platformv1alpha1.Application{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Application")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile adds finalizer
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile creates deployment
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Application has a finalizer")
			var updatedApp platformv1alpha1.Application
			Expect(k8sClient.Get(ctx, typeNamespacedName, &updatedApp)).To(Succeed())
			Expect(updatedApp.Finalizers).To(ContainElement("platform.operator.kibaship.com/application-finalizer"))

			By("Verifying the Application has the project UUID label")
			Expect(updatedApp.Labels).To(HaveKeyWithValue("platform.kibaship.com/project-uuid", "550e8400-e29b-41d4-a716-446655440000"))

			By("Verifying a Deployment was created")
			deploymentName := resourceName + "-deployment"
			deploymentKey := types.NamespacedName{
				Name:      deploymentName,
				Namespace: "default",
			}
			var deployment platformv1alpha1.Deployment
			Eventually(func() error {
				return k8sClient.Get(ctx, deploymentKey, &deployment)
			}).Should(Succeed())

			By("Verifying the Deployment references the Application")
			Expect(deployment.Spec.ApplicationRef.Name).To(Equal(resourceName))

			By("Verifying the Application status is updated")
			Eventually(func() []metav1.Condition {
				Expect(k8sClient.Get(ctx, typeNamespacedName, &updatedApp)).To(Succeed())
				return updatedApp.Status.Conditions
			}).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal("DeploymentReady"),
				"Status": Equal(metav1.ConditionTrue),
			})))
		})

		It("should validate GitRepository application type", func() {
			By("Creating a GitRepository application")
			gitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-git-app",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440010",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:           platformv1alpha1.GitProviderGitHub,
						Repository:         "myorg/myrepo",
						Branch:             "main",
						RootDirectory:      "./",
						BuildCommand:       "npm install && npm run build",
						StartCommand:       "npm start",
						SpaOutputDirectory: "dist",
						SecretRef: corev1.LocalObjectReference{
							Name: "git-token",
						},
						Env: &corev1.LocalObjectReference{
							Name: "app-env-secret",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, gitApp)).To(Succeed())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, gitApp)).To(Succeed())
			}()
		})

		It("should validate MySQL application type", func() {
			By("Creating a MySQL application")
			mysqlApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mysql-app",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440011",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeMySQL,
					MySQL: &platformv1alpha1.MySQLConfig{
						Version:  "8.0",
						Database: "testdb",
						SecretRef: &corev1.LocalObjectReference{
							Name: "mysql-creds",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mysqlApp)).To(Succeed())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, mysqlApp)).To(Succeed())
			}()
		})

		It("should reject invalid repository format", func() {
			By("Creating a GitRepository application with invalid repo format")
			invalidGitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-git-app",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440012",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:   platformv1alpha1.GitProviderGitHub,
						Repository: "invalid-repo-format",
						SecretRef: corev1.LocalObjectReference{
							Name: "git-token",
						},
					},
				},
			}
			err := k8sClient.Create(ctx, invalidGitApp)
			Expect(err).To(HaveOccurred())
		})

		It("should successfully create valid applications with all required fields", func() {
			By("Creating an application with all required fields")
			validApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-valid-app",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440013",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeDockerImage,
					DockerImage: &platformv1alpha1.DockerImageConfig{
						Image: "nginx:latest",
					},
				},
			}
			err := k8sClient.Create(ctx, validApp)
			Expect(err).NotTo(HaveOccurred())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, validApp)).To(Succeed())
			}()
		})

		It("should successfully create GitRepository with default rootDirectory", func() {
			By("Creating a GitRepository application with minimal fields")
			minimalGitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-minimal-git-app",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440014",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:   platformv1alpha1.GitProviderGitHub,
						Repository: "myorg/minimal-repo",
						SecretRef: corev1.LocalObjectReference{
							Name: "git-token",
						},
						// rootDirectory should default to "./"
						// buildCommand and startCommand are optional
					},
				},
			}
			Expect(k8sClient.Create(ctx, minimalGitApp)).To(Succeed())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, minimalGitApp)).To(Succeed())
			}()
		})

		It("should successfully create GitRepository with spaOutputDirectory", func() {
			By("Creating a GitRepository application with SPA output directory")
			spaGitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-spa-git-app",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440015",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:           platformv1alpha1.GitProviderGitHub,
						Repository:         "myorg/spa-app",
						Branch:             "main",
						BuildCommand:       "npm run build",
						SpaOutputDirectory: "build",
						SecretRef: corev1.LocalObjectReference{
							Name: "git-token",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, spaGitApp)).To(Succeed())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, spaGitApp)).To(Succeed())
			}()
		})

		It("should update existing Deployment when Application is modified", func() {
			By("Creating an Application")
			testApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-update-deployment",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440004",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:   platformv1alpha1.GitProviderGitHub,
						Repository: "myorg/test-repo",
						SecretRef: corev1.LocalObjectReference{
							Name: "git-token",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testApp)).To(Succeed())

			By("Reconciling to create the Deployment")
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			// First reconcile adds finalizer
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testApp.Name,
					Namespace: testApp.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile creates deployment
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testApp.Name,
					Namespace: testApp.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Modifying the Application")
			var updatedApp platformv1alpha1.Application
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testApp.Name,
				Namespace: testApp.Namespace,
			}, &updatedApp)).To(Succeed())

			updatedApp.Spec.GitRepository.Repository = "myorg/updated-repo"
			Expect(k8sClient.Update(ctx, &updatedApp)).To(Succeed())

			By("Reconciling again")
			// This should be a single reconcile since finalizer already exists
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testApp.Name,
					Namespace: testApp.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Deployment still exists and references the Application")
			deploymentName := testApp.Name + "-deployment"
			var deployment platformv1alpha1.Deployment
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      deploymentName,
				Namespace: testApp.Namespace,
			}, &deployment)).To(Succeed())
			Expect(deployment.Spec.ApplicationRef.Name).To(Equal(testApp.Name))

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, &updatedApp)).To(Succeed())
			}()
		})

		It("should create a Deployment when Application is created", func() {
			By("Creating an Application")
			testApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-create-deployment",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440002",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:   platformv1alpha1.GitProviderGitHub,
						Repository: "myorg/test-repo",
						SecretRef: corev1.LocalObjectReference{
							Name: "git-token",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testApp)).To(Succeed())

			By("Reconciling the Application")
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			// First reconcile adds finalizer
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testApp.Name,
					Namespace: testApp.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile creates deployment
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testApp.Name,
					Namespace: testApp.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Deployment was created")
			deploymentName := testApp.Name + "-deployment"
			var deployment platformv1alpha1.Deployment
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      deploymentName,
					Namespace: testApp.Namespace,
				}, &deployment)
			}).Should(Succeed())

			By("Verifying the Deployment has correct labels")
			Expect(deployment.Labels).To(HaveKeyWithValue("platform.operator.kibaship.com/application", testApp.Name))
			Expect(deployment.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", testApp.Name))

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, testApp)).To(Succeed())
			}()
		})

		It("should set project UUID label when reconciling Application", func() {
			By("Creating an Application with only PaaS UUID")
			testApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-uuid-labeling",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440100",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeDockerImage,
					DockerImage: &platformv1alpha1.DockerImageConfig{
						Image: "nginx:latest",
					},
				},
			}
			Expect(k8sClient.Create(ctx, testApp)).To(Succeed())

			By("Reconciling the Application")
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile adds finalizer and UUID labels
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testApp.Name,
					Namespace: testApp.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Application has both UUID labels")
			var updatedApp platformv1alpha1.Application
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testApp.Name,
				Namespace: testApp.Namespace,
			}, &updatedApp)).To(Succeed())

			Expect(updatedApp.Labels).To(HaveKeyWithValue("platform.kibaship.com/uuid", "550e8400-e29b-41d4-a716-446655440100"))
			Expect(updatedApp.Labels).To(HaveKeyWithValue("platform.kibaship.com/project-uuid", "550e8400-e29b-41d4-a716-446655440000"))

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, &updatedApp)).To(Succeed())
			}()
		})

		It("should fail if Application does not have PaaS UUID label", func() {
			By("Creating an Application without PaaS UUID label")
			testApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-missing-uuid",
					Namespace: "default",
					// No platform.kibaship.com/uuid label
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: platformv1alpha1.ApplicationTypeDockerImage,
					DockerImage: &platformv1alpha1.DockerImageConfig{
						Image: "nginx:latest",
					},
				},
			}
			Expect(k8sClient.Create(ctx, testApp)).To(Succeed())

			By("Reconciling should fail due to missing UUID")
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testApp.Name,
					Namespace: testApp.Namespace,
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("application must have label 'platform.kibaship.com/uuid' set by PaaS system"))

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, testApp)).To(Succeed())
			}()
		})

		It("should efficiently delete Deployments using label selector", func() {
			By("Creating multiple applications to test label selector efficiency")

			// Create first application and deployment
			app1 := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-label-selector-1",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: "GitRepository",
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:   "github.com",
						Repository: "user/repo1",
						Branch:     "main",
					},
				},
			}
			Expect(k8sClient.Create(ctx, app1)).To(Succeed())

			// Create second application and deployment
			app2 := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-label-selector-2",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid": "550e8400-e29b-41d4-a716-446655440001",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					ProjectRef: corev1.LocalObjectReference{
						Name: "test-project",
					},
					Type: "GitRepository",
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:   "github.com",
						Repository: "user/repo2",
						Branch:     "main",
					},
				},
			}
			Expect(k8sClient.Create(ctx, app2)).To(Succeed())

			// Create reconciler instance
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Reconcile both applications to create their deployments
			// First reconcile adds finalizer for app1
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app1.Name,
					Namespace: app1.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile creates deployment for app1
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app1.Name,
					Namespace: app1.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// First reconcile adds finalizer for app2
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app2.Name,
					Namespace: app2.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile creates deployment for app2
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app2.Name,
					Namespace: app2.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify both deployments exist
			var deployment1 platformv1alpha1.Deployment
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-label-selector-1-deployment",
				Namespace: app1.Namespace,
			}, &deployment1)).To(Succeed())

			var deployment2 platformv1alpha1.Deployment
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-label-selector-2-deployment",
				Namespace: app2.Namespace,
			}, &deployment2)).To(Succeed())

			// Verify deployments have correct labels
			Expect(deployment1.Labels["platform.operator.kibaship.com/application"]).To(Equal("test-label-selector-1"))
			Expect(deployment2.Labels["platform.operator.kibaship.com/application"]).To(Equal("test-label-selector-2"))

			// Delete first application - this should only delete its associated deployment
			Expect(k8sClient.Delete(ctx, app1)).To(Succeed())

			// Reconcile deletion
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app1.Name,
					Namespace: app1.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify only deployment1 is deleted, deployment2 still exists
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-label-selector-1-deployment",
				Namespace: app1.Namespace,
			}, &deployment1)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// deployment2 should still exist
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-label-selector-2-deployment",
				Namespace: app2.Namespace,
			}, &deployment2)).To(Succeed())

			// Cleanup
			Expect(k8sClient.Delete(ctx, app2)).To(Succeed())
		})
	})
})
