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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
			By("creating the custom resource for the Kind Application")
			err := k8sClient.Get(ctx, typeNamespacedName, application)
			if err != nil && errors.IsNotFound(err) {
				resource := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
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

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})

		It("should validate GitRepository application type", func() {
			By("Creating a GitRepository application")
			gitApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-git-app",
					Namespace: "default",
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
	})
})
