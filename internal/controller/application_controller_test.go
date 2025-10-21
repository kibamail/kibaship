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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/utils"
	"github.com/kibamail/kibaship/pkg/validation"
)

var _ = Describe("Application Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "project-myproject-app-myapp"

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
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440000",
						validation.LabelResourceSlug: "test-project",
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

			By("creating a test environment")
			testEnvironment := &platformv1alpha1.Environment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "environment-production",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID: "env-uuid-production",
						validation.LabelResourceSlug: "production",
						validation.LabelProjectUUID:  "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.EnvironmentSpec{
					ProjectRef: corev1.LocalObjectReference{Name: "test-project"},
				},
			}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "environment-production", Namespace: "default"}, &platformv1alpha1.Environment{})
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, testEnvironment)).To(Succeed())
			}

			By("creating the custom resource for the Kind Application")
			err = k8sClient.Get(ctx, typeNamespacedName, application)
			if err != nil && errors.IsNotFound(err) {
				resource := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
						Labels: map[string]string{
							validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440001",
							validation.LabelResourceSlug:    "myapp",
							validation.LabelEnvironmentUUID: "env-uuid-production",
							validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						EnvironmentRef: corev1.LocalObjectReference{
							Name: "environment-production",
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

			// Second reconcile creates env secret and updates spec
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Third reconcile completes and sets status
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the Application has a finalizer")
			var updatedApp platformv1alpha1.Application
			Expect(k8sClient.Get(ctx, typeNamespacedName, &updatedApp)).To(Succeed())
			Expect(updatedApp.Finalizers).To(ContainElement("platform.operator.kibaship.com/application-finalizer"))

			By("Verifying the Application has the environment UUID label")
			Expect(updatedApp.Labels).To(HaveKeyWithValue("platform.kibaship.com/environment-uuid", "env-uuid-production"))

			By("Verifying the Application status is updated")
			Eventually(func() []metav1.Condition {
				Expect(k8sClient.Get(ctx, typeNamespacedName, &updatedApp)).To(Succeed())
				return updatedApp.Status.Conditions
			}).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal("Ready"),
				"Status": Equal(metav1.ConditionTrue),
			})))

			By("Verifying the Application phase is Ready")
			Eventually(func() string {
				Expect(k8sClient.Get(ctx, typeNamespacedName, &updatedApp)).To(Succeed())
				return updatedApp.Status.Phase
			}).Should(Equal("Ready"))
		})

		It("should validate GitRepository application type", func() {
			By("Creating a GitRepository application")
			gitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-gitapp",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440010",
						validation.LabelResourceSlug:    "gitapp",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
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
						PublicAccess:       false,
						SecretRef: &corev1.LocalObjectReference{
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

		// TODO: MySQL application validation test removed - will be reimplemented
		It("should validate MySQL application type", func() {
			Skip("MySQL application validation test removed - TODO: implement new test")
		})

		It("should reject invalid repository format", func() {
			By("Creating a GitRepository application with invalid repo format")
			invalidGitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-invalidgitapp",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440012",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:   platformv1alpha1.GitProviderGitHub,
						Repository: "invalid-repo-format",
						SecretRef: &corev1.LocalObjectReference{
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
					Name:      "project-myproject-app-validapp",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440013",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
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
					Name:      "project-myproject-app-minimalgitapp",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440014",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "myorg/minimal-repo",
						PublicAccess: false,
						SecretRef: &corev1.LocalObjectReference{
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
					Name:      "project-myproject-app-spagitapp",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440015",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:           platformv1alpha1.GitProviderGitHub,
						Repository:         "myorg/spa-app",
						Branch:             "main",
						BuildCommand:       "npm run build",
						SpaOutputDirectory: "build",
						PublicAccess:       false,
						SecretRef: &corev1.LocalObjectReference{
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

		It("should set environment UUID label when reconciling Application", func() {
			By("Creating an Application with UUID labels")
			testApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-uuidtest",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440100",
						validation.LabelResourceSlug:    "uuidtest",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
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

			By("Verifying the Application has all UUID labels")
			var updatedApp platformv1alpha1.Application
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testApp.Name,
				Namespace: testApp.Namespace,
			}, &updatedApp)).To(Succeed())

			Expect(updatedApp.Labels).To(HaveKeyWithValue(validation.LabelResourceUUID, "550e8400-e29b-41d4-a716-446655440100"))
			Expect(updatedApp.Labels).To(HaveKeyWithValue(validation.LabelEnvironmentUUID, "env-uuid-production"))
			Expect(updatedApp.Labels).To(HaveKeyWithValue(validation.LabelProjectUUID, "550e8400-e29b-41d4-a716-446655440000"))

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
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
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
			Expect(err.Error()).To(ContainSubstring("application must have labels"))

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, testApp)).To(Succeed())
			}()
		})

		It("should allow GitRepository with PublicAccess true and no SecretRef", func() {
			By("Creating a GitRepository application with PublicAccess=true and no SecretRef")
			publicGitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-publicgitapp",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440020",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "myorg/public-repo",
						PublicAccess: true,
						// No SecretRef provided - should be allowed
					},
				},
			}
			err := k8sClient.Create(ctx, publicGitApp)
			Expect(err).NotTo(HaveOccurred())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, publicGitApp)).To(Succeed())
			}()
		})

		It("should allow GitRepository with PublicAccess true and optional SecretRef", func() {
			By("Creating a GitRepository application with PublicAccess=true and SecretRef provided")
			publicGitAppWithSecret := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-publicwithsecret",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440021",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "myorg/public-repo-with-secret",
						PublicAccess: true,
						SecretRef: &corev1.LocalObjectReference{
							Name: "optional-git-token",
						},
					},
				},
			}
			err := k8sClient.Create(ctx, publicGitAppWithSecret)
			Expect(err).NotTo(HaveOccurred())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, publicGitAppWithSecret)).To(Succeed())
			}()
		})

		It("should allow GitRepository with PublicAccess false and SecretRef provided", func() {
			By("Creating a GitRepository application with PublicAccess=false and SecretRef provided")
			privateGitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-privategitapp",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440022",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "myorg/private-repo",
						PublicAccess: false,
						SecretRef: &corev1.LocalObjectReference{
							Name: "required-git-token",
						},
					},
				},
			}
			err := k8sClient.Create(ctx, privateGitApp)
			Expect(err).NotTo(HaveOccurred())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, privateGitApp)).To(Succeed())
			}()
		})

		It("should reject GitRepository with PublicAccess false and no SecretRef", func() {
			By("Testing validation directly for GitRepository application with PublicAccess=false and no SecretRef")
			invalidPrivateGitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-invalidprivate",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440023",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "myorg/private-repo-no-secret",
						PublicAccess: false,
						// No SecretRef provided - should be rejected
					},
				},
			}

			// Test the validation method directly since webhook validation doesn't run in unit tests
			_, err := invalidPrivateGitApp.ValidateCreate(ctx, invalidPrivateGitApp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SecretRef is required when PublicAccess is false"))
		})

		It("should default PublicAccess to false when not specified", func() {
			By("Creating a GitRepository application without specifying PublicAccess")
			defaultPublicAccessApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-defaultaccess",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440024",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:   platformv1alpha1.GitProviderGitHub,
						Repository: "myorg/default-access-repo",
						// PublicAccess not specified - should default to false
						SecretRef: &corev1.LocalObjectReference{
							Name: "git-token",
						},
					},
				},
			}
			err := k8sClient.Create(ctx, defaultPublicAccessApp)
			Expect(err).NotTo(HaveOccurred())

			// Verify that PublicAccess defaults to false
			var createdApp platformv1alpha1.Application
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      defaultPublicAccessApp.Name,
				Namespace: defaultPublicAccessApp.Namespace,
			}, &createdApp)).To(Succeed())

			Expect(createdApp.Spec.GitRepository.PublicAccess).To(BeFalse())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, &createdApp)).To(Succeed())
			}()
		})
	})

	Context("Application Environment Variables Secret", func() {
		It("should create env secret for GitRepository application", func() {
			By("Creating a GitRepository application")
			appUUID := "660e8400-e29b-41d4-a716-446655440025"
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-envtest",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    appUUID,
						validation.LabelResourceSlug:    "envtest1",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "myorg/envtest-repo",
						PublicAccess: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			By("Reconciling the application to create env secret")
			reconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			// First reconcile: adds finalizer
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile: creates env secret
			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying env secret was created")
			expectedSecretName := utils.GetApplicationResourceName(appUUID)
			var secret corev1.Secret
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      expectedSecretName,
					Namespace: app.Namespace,
				}, &secret)
				return err == nil
			}, "5s", "100ms").Should(BeTrue())

			By("Verifying secret has correct labels")
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "kibaship"))
			Expect(secret.Labels).To(HaveKeyWithValue(validation.LabelApplicationUUID, appUUID))
			Expect(secret.Labels).To(HaveKeyWithValue("platform.operator.kibaship.com/type", "application-env-vars"))

			By("Verifying secret ref was set on Application")
			var updatedApp platformv1alpha1.Application
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				}, &updatedApp)
				if err != nil {
					return false
				}
				return updatedApp.Spec.GitRepository != nil &&
					updatedApp.Spec.GitRepository.Env != nil &&
					updatedApp.Spec.GitRepository.Env.Name == expectedSecretName
			}, "5s", "100ms").Should(BeTrue())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, app)).To(Succeed())
				Expect(k8sClient.Delete(ctx, &secret)).To(Succeed())
			}()
		})

		It("should create env secret for DockerImage applications", func() {
			By("Creating a DockerImage application")
			appUUID := "660e8400-e29b-41d4-a716-446655440026"
			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-dockertest",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    appUUID,
						validation.LabelResourceSlug:    "dockertest1",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeDockerImage,
					DockerImage: &platformv1alpha1.DockerImageConfig{
						Image: "nginx:latest",
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			By("Reconciling the application")
			reconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile: adds finalizer
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile: creates env secret
			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying env secret was created")
			expectedSecretName := utils.GetApplicationResourceName(appUUID)
			var secret corev1.Secret
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      expectedSecretName,
					Namespace: app.Namespace,
				}, &secret)
				return err == nil
			}, "5s", "100ms").Should(BeTrue())

			By("Verifying secret has correct labels")
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "kibaship"))
			Expect(secret.Labels).To(HaveKeyWithValue(validation.LabelApplicationUUID, appUUID))
			Expect(secret.Labels).To(HaveKeyWithValue("platform.operator.kibaship.com/type", "application-env-vars"))

			By("Verifying secret ref was set on DockerImage config")
			var updatedApp platformv1alpha1.Application
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				}, &updatedApp)
				if err != nil {
					return false
				}
				return updatedApp.Spec.DockerImage != nil &&
					updatedApp.Spec.DockerImage.Env != nil &&
					updatedApp.Spec.DockerImage.Env.Name == expectedSecretName
			}, "5s", "100ms").Should(BeTrue())

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, app)).To(Succeed())
				Expect(k8sClient.Delete(ctx, &secret)).To(Succeed())
			}()
		})

		It("should not recreate env secret if it already exists", func() {
			By("Creating a GitRepository application with existing secret")
			appUUID := "660e8400-e29b-41d4-a716-446655440027"
			secretName := utils.GetApplicationResourceName(appUUID)

			// Create secret first
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by":        "kibaship",
						validation.LabelApplicationUUID:       appUUID,
						"platform.operator.kibaship.com/type": "application-env-vars",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"EXISTING_VAR": []byte("existing-value"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			app := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-myproject-app-existingsecret",
					Namespace: "default",
					Labels: map[string]string{
						validation.LabelResourceUUID:    appUUID,
						validation.LabelResourceSlug:    "existingsecret1",
						validation.LabelEnvironmentUUID: "env-uuid-production",
						validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					EnvironmentRef: corev1.LocalObjectReference{
						Name: "environment-production",
					},
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     platformv1alpha1.GitProviderGitHub,
						Repository:   "myorg/existingsecret-repo",
						PublicAccess: true,
						Env: &corev1.LocalObjectReference{
							Name: secretName,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			By("Reconciling the application")
			reconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile: adds finalizer
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile: handles env secret check
			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying existing secret data was preserved")
			var retrievedSecret corev1.Secret
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      secretName,
				Namespace: app.Namespace,
			}, &retrievedSecret)).To(Succeed())
			Expect(retrievedSecret.Data).To(HaveKeyWithValue("EXISTING_VAR", []byte("existing-value")))

			// Cleanup
			defer func() {
				Expect(k8sClient.Delete(ctx, app)).To(Succeed())
				Expect(k8sClient.Delete(ctx, &retrievedSecret)).To(Succeed())
			}()
		})
	})
})
