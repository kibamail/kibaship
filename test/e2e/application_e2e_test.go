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
)

var _ = Describe("Application E2E Tests", Ordered, func() {
	var k8sClient client.Client
	var testNamespace string
	ctx := context.Background()

	BeforeAll(func() {
		By("Setting up kubernetes client")
		cfg, err := config.GetConfig()
		Expect(err).NotTo(HaveOccurred())

		err = platformv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())

		// Create test namespace
		testNamespace = "application-e2e-test"
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

	Context("Application Lifecycle Management", func() {
		var testProject *platformv1alpha1.Project
		var testApplication *platformv1alpha1.Application

		BeforeEach(func() {
			By("Creating test project")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-project-app",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440000",
						validation.LabelResourceSlug:  "test-project-app",
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
		})

		AfterEach(func() {
			By("Cleaning up test resources")
			// Delete application
			if testApplication != nil {
				_ = k8sClient.Delete(ctx, testApplication)
			}

			// Delete project
			if testProject != nil {
				_ = k8sClient.Delete(ctx, testProject)
			}
		})

		It("should successfully create and reconcile GitRepository application", func() {
			By("Creating GitRepository application")
			testApplication = &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-app-app-frontend-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440001",
						validation.LabelResourceSlug: "frontend",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/frontend",
						Branch:     "main",
					},
				},
			}

			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			By("Verifying application transitions to Ready phase")
			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Verifying finalizer is added")
			var app platformv1alpha1.Application
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
			Expect(err).NotTo(HaveOccurred())
			Expect(app.Finalizers).To(ContainElement("platform.operator.kibaship.com/application-finalizer"))

			By("Verifying UUID labels are properly set")
			Expect(app.Labels[validation.LabelResourceUUID]).NotTo(BeEmpty())
			Expect(app.Labels[validation.LabelResourceSlug]).To(Equal("frontend"))
			Expect(app.Labels[validation.LabelProjectUUID]).To(Equal(testProject.Labels[validation.LabelResourceUUID]))

			By("Verifying status is updated correctly")
			Expect(app.Status.Phase).To(Equal("Ready"))
			Expect(app.Status.ObservedGeneration).To(Equal(app.Generation))
		})

		It("should automatically create default ApplicationDomain for GitRepository application", func() {
			By("Creating GitRepository application")
			testApplication = &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-app-app-backend-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440002",
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
						Branch:     "develop",
					},
				},
			}

			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			By("Waiting for application to be ready")
			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Verifying default ApplicationDomain was created")
			var domains platformv1alpha1.ApplicationDomainList
			Eventually(func() int {
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}

				// Count domains for this application
				count := 0
				for _, domain := range domains.Items {
					if domain.Spec.ApplicationRef.Name == testApplication.Name && domain.Spec.Default {
						count++
					}
				}
				return count
			}, time.Minute*2, time.Second*5).Should(Equal(1))

			By("Verifying domain properties")
			// Find the default domain
			var defaultDomain *platformv1alpha1.ApplicationDomain
			for _, domain := range domains.Items {
				if domain.Spec.ApplicationRef.Name == testApplication.Name && domain.Spec.Default {
					defaultDomain = &domain
					break
				}
			}
			Expect(defaultDomain).NotTo(BeNil())

			Expect(defaultDomain.Spec.Type).To(Equal(platformv1alpha1.ApplicationDomainTypeDefault))
			Expect(defaultDomain.Spec.Default).To(BeTrue())
			Expect(defaultDomain.Spec.Domain).NotTo(BeEmpty())
			Expect(defaultDomain.Labels["platform.kibaship.com/application"]).To(Equal(testApplication.Name))
			Expect(defaultDomain.Labels["platform.kibaship.com/domain-type"]).To(Equal("default"))
		})

		It("should handle application with missing project reference gracefully", func() {
			By("Creating application with invalid project reference")
			testApplication = &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-nonexistent-app-invalid-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440003",
						validation.LabelResourceSlug: "invalid",
						validation.LabelProjectUUID:  "non-existent-project-uuid",
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: "non-existent-project",
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/invalid",
						Branch:     "main",
					},
				},
			}

			err := k8sClient.Create(ctx, testApplication)
			if err == nil {
				// If creation succeeded, it should fail during reconciliation
				Eventually(func() string {
					var app platformv1alpha1.Application
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
					if err != nil {
						return "NotFound"
					}
					return app.Status.Phase
				}, time.Minute*1, time.Second*5).Should(Or(Equal("Failed"), Equal("Pending")))
			}
		})

		It("should properly update application status during lifecycle", func() {
			By("Creating application")
			testApplication = &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-app-app-api-kibaship-com",
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

			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			By("Verifying application goes through initialization phases")
			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Or(Equal("Initializing"), Equal("Ready")))

			By("Verifying status conditions are set")
			Eventually(func() bool {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return false
				}
				return len(app.Status.Conditions) > 0
			}, time.Minute*1, time.Second*5).Should(BeTrue())

			By("Verifying observedGeneration is updated")
			Eventually(func() bool {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return false
				}
				return app.Status.ObservedGeneration == app.Generation
			}, time.Minute*1, time.Second*5).Should(BeTrue())
		})

		It("should handle application deletion and cleanup properly", func() {
			By("Creating application to be deleted")
			testApplication = &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-app-app-temp-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440005",
						validation.LabelResourceSlug: "temp",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/temp",
						Branch:     "main",
					},
				},
			}

			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			By("Waiting for application to be ready")
			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Verifying default domain was created")
			var domains platformv1alpha1.ApplicationDomainList
			Eventually(func() int {
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}

				count := 0
				for _, domain := range domains.Items {
					if domain.Spec.ApplicationRef.Name == testApplication.Name {
						count++
					}
				}
				return count
			}, time.Minute*1, time.Second*5).Should(BeNumerically(">=", 1))

			By("Deleting the application")
			Expect(k8sClient.Delete(ctx, testApplication)).To(Succeed())

			By("Verifying application is eventually removed")
			Eventually(func() bool {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				return errors.IsNotFound(err)
			}, time.Minute*2, time.Second*5).Should(BeTrue())

			By("Verifying associated domains are cleaned up")
			Eventually(func() int {
				var domainList platformv1alpha1.ApplicationDomainList
				err := k8sClient.List(ctx, &domainList, client.InNamespace(testNamespace))
				if err != nil {
					return 999 // Return high number on error to continue waiting
				}

				count := 0
				for _, domain := range domainList.Items {
					if domain.Spec.ApplicationRef.Name == testApplication.Name {
						count++
					}
				}
				return count
			}, time.Minute*2, time.Second*5).Should(Equal(0))

			// Set to nil so AfterEach doesn't try to delete again
			testApplication = nil
		})
	})

	Context("Application Validation and Edge Cases", func() {
		var testProject *platformv1alpha1.Project

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
		})

		AfterEach(func() {
			By("Cleaning up validation test resources")
			// Delete project
			if testProject != nil {
				_ = k8sClient.Delete(ctx, testProject)
			}
		})

		It("should enforce proper application naming conventions", func() {
			By("Attempting to create application with invalid name format")
			invalidApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-application-name",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440101",
						validation.LabelResourceSlug: "invalid",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/invalid",
						Branch:     "main",
					},
				},
			}

			err := k8sClient.Create(ctx, invalidApp)
			Expect(err).To(HaveOccurred(), "Should reject application with invalid name format")
		})

		It("should handle concurrent application operations gracefully", func() {
			By("Creating multiple applications concurrently")
			applications := make([]*platformv1alpha1.Application, 3)

			for i := 0; i < 3; i++ {
				app := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("project-test-project-validation-app-concurrent-%d-kibaship-com", i),
						Namespace: testNamespace,
						Labels: map[string]string{
							validation.LabelResourceUUID: fmt.Sprintf("550e8400-e29b-41d4-a716-44665544010%d", i),
							validation.LabelResourceSlug: fmt.Sprintf("concurrent-%d", i),
							validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						Type: platformv1alpha1.ApplicationTypeGitRepository,
						ProjectRef: corev1.LocalObjectReference{
							Name: testProject.Name,
						},
						GitRepository: &platformv1alpha1.GitRepositoryConfig{
							Repository: fmt.Sprintf("https://github.com/test/concurrent-%d", i),
							Branch:     "main",
						},
					},
				}
				applications[i] = app

				go func(app *platformv1alpha1.Application) {
					defer GinkgoRecover()
					err := k8sClient.Create(ctx, app)
					Expect(err).NotTo(HaveOccurred())
				}(app)
			}

			By("Verifying all applications eventually become ready")
			for _, app := range applications {
				Eventually(func() string {
					var application platformv1alpha1.Application
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), &application)
					if err != nil {
						return ""
					}
					return application.Status.Phase
				}, time.Minute*3, time.Second*5).Should(Equal("Ready"))
			}

			By("Verifying all applications have default domains")
			for _, app := range applications {
				Eventually(func() int {
					var domains platformv1alpha1.ApplicationDomainList
					err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
					if err != nil {
						return 0
					}

					count := 0
					for _, domain := range domains.Items {
						if domain.Spec.ApplicationRef.Name == app.Name && domain.Spec.Default {
							count++
						}
					}
					return count
				}, time.Minute*2, time.Second*5).Should(Equal(1))
			}

			By("Cleaning up concurrent test applications")
			for _, app := range applications {
				_ = k8sClient.Delete(ctx, app)
			}
		})

		It("should handle GitRepository configuration updates", func() {
			By("Creating application")
			testApp := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "project-test-project-validation-app-updatable-kibaship-com",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440110",
						validation.LabelResourceSlug: "updatable",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/updatable",
						Branch:     "main",
					},
				},
			}

			Expect(k8sClient.Create(ctx, testApp)).To(Succeed())

			Eventually(func() string {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApp), &app)
				if err != nil {
					return ""
				}
				return app.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Updating GitRepository configuration")
			var app platformv1alpha1.Application
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApp), &app)
			Expect(err).NotTo(HaveOccurred())

			app.Spec.GitRepository.Branch = "develop"
			app.Spec.GitRepository.Repository = "https://github.com/test/updatable-v2"

			Expect(k8sClient.Update(ctx, &app)).To(Succeed())

			By("Verifying application processes update successfully")
			Eventually(func() int64 {
				var updatedApp platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApp), &updatedApp)
				if err != nil {
					return 0
				}
				return updatedApp.Status.ObservedGeneration
			}, time.Minute*1, time.Second*5).Should(Equal(int64(2)))

			// Clean up
			_ = k8sClient.Delete(ctx, testApp)
		})
	})
})
