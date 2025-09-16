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
	"os/exec"
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
	"github.com/kibamail/kibaship-operator/test/utils"
)

var _ = Describe("ApplicationDomain E2E Tests", Ordered, func() {
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
		testNamespace = "applicationdomain-e2e-test"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		err = k8sClient.Create(ctx, ns)
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		// Set environment variables for domain generation
		By("Setting environment variables for domain testing")
		cmd := exec.Command("kubectl", "set", "env", "deployment/kibaship-operator-controller-manager",
			"KIBASHIP_OPERATOR_DOMAIN=test.kibaship.com",
			"KIBASHIP_OPERATOR_DEFAULT_PORT=3000",
			"-n", namespace)
		_, _ = utils.Run(cmd)
	})

	AfterAll(func() {
		By("Cleaning up test namespace")
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(ctx, ns)
	})

	Context("ApplicationDomain Lifecycle", func() {
		var testProject *platformv1alpha1.Project
		var testApplication *platformv1alpha1.Application

		BeforeEach(func() {
			By("Creating test project")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-project-domain",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440000",
						validation.LabelResourceSlug:  "test-project-domain",
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
					Name:      "project-test-project-domain-app-frontend-kibaship-com",
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
			// Delete application domains
			domainList := &platformv1alpha1.ApplicationDomainList{}
			_ = k8sClient.List(ctx, domainList, client.InNamespace(testNamespace))
			for _, domain := range domainList.Items {
				_ = k8sClient.Delete(ctx, &domain)
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

		It("should automatically create default domain for GitRepository application", func() {
			By("Verifying ApplicationDomain was created automatically")
			var domains platformv1alpha1.ApplicationDomainList
			Eventually(func() int {
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}
				return len(domains.Items)
			}, time.Minute*2, time.Second*5).Should(BeNumerically(">=", 1))

			domain := domains.Items[0]

			By("Verifying domain properties")
			Expect(domain.Spec.ApplicationRef.Name).To(Equal(testApplication.Name))
			Expect(domain.Spec.Type).To(Equal(platformv1alpha1.ApplicationDomainTypeDefault))
			Expect(domain.Spec.Default).To(BeTrue())
			Expect(domain.Spec.Port).To(Equal(int32(3000)))
			Expect(domain.Spec.TLSEnabled).To(BeTrue())
			Expect(domain.Spec.Domain).NotTo(BeEmpty())

			By("Verifying domain labels")
			Expect(domain.Labels).To(HaveKeyWithValue("platform.kibaship.com/application", testApplication.Name))
			Expect(domain.Labels).To(HaveKeyWithValue("platform.kibaship.com/domain-type", "default"))
			Expect(domain.Labels).To(HaveKeyWithValue(validation.LabelApplicationUUID, testApplication.Labels[validation.LabelResourceUUID]))
			Expect(domain.Labels).To(HaveKeyWithValue(validation.LabelProjectUUID, testApplication.Labels[validation.LabelProjectUUID]))

			// Domain should have its own UUID
			domainUUID := domain.Labels[validation.LabelResourceUUID]
			Expect(domainUUID).NotTo(BeEmpty())
			Expect(domainUUID).NotTo(Equal(testApplication.Labels[validation.LabelResourceUUID]))

			By("Verifying domain status becomes ready")
			Eventually(func() string {
				var updatedDomain platformv1alpha1.ApplicationDomain
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&domain), &updatedDomain)
				if err != nil {
					return ""
				}
				return string(updatedDomain.Status.Phase)
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))
		})

		It("should create custom domain when ApplicationDomain resource is created", func() {
			By("Creating custom ApplicationDomain")
			customDomain := &platformv1alpha1.ApplicationDomain{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-frontend-domain",
					Namespace: testNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:        "550e8400-e29b-41d4-a716-446655440002",
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
					Domain:     "custom.frontend.test.kibaship.com",
					Default:    false,
					Port:       8080,
					TLSEnabled: true,
				},
			}

			Expect(k8sClient.Create(ctx, customDomain)).To(Succeed())

			By("Verifying custom domain becomes ready")
			Eventually(func() string {
				var domain platformv1alpha1.ApplicationDomain
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(customDomain), &domain)
				if err != nil {
					return ""
				}
				return string(domain.Status.Phase)
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Verifying multiple domains exist for the application")
			var domains platformv1alpha1.ApplicationDomainList
			err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(domains.Items)).To(BeNumerically(">=", 2))

			// Should have one default and one custom domain
			defaultDomainCount := 0
			customDomainCount := 0
			for _, domain := range domains.Items {
				if domain.Spec.Default {
					defaultDomainCount++
				} else {
					customDomainCount++
				}
			}
			Expect(defaultDomainCount).To(Equal(1))
			Expect(customDomainCount).To(BeNumerically(">=", 1))
		})

		It("should enforce domain validation rules", func() {
			By("Attempting to create domain with invalid domain name")
			invalidDomain := &platformv1alpha1.ApplicationDomain{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-domain-test",
					Namespace: testNamespace,
				},
				Spec: platformv1alpha1.ApplicationDomainSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: testApplication.Name,
					},
					Type:       platformv1alpha1.ApplicationDomainTypeCustom,
					Domain:     "invalid..domain.com", // Invalid double dots
					Default:    false,
					Port:       8080,
					TLSEnabled: true,
				},
			}

			err := k8sClient.Create(ctx, invalidDomain)
			Expect(err).To(HaveOccurred(), "Should reject invalid domain format")

			By("Attempting to create domain with invalid application reference")
			invalidAppRef := &platformv1alpha1.ApplicationDomain{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-app-ref-test",
					Namespace: testNamespace,
				},
				Spec: platformv1alpha1.ApplicationDomainSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: "non-existent-application",
					},
					Type:       platformv1alpha1.ApplicationDomainTypeCustom,
					Domain:     "test.domain.com",
					Default:    false,
					Port:       8080,
					TLSEnabled: true,
				},
			}

			err = k8sClient.Create(ctx, invalidAppRef)
			if err == nil {
				// If creation succeeded, it should fail during reconciliation
				Eventually(func() string {
					var domain platformv1alpha1.ApplicationDomain
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(invalidAppRef), &domain)
					if err != nil {
						return "NotFound"
					}
					return string(domain.Status.Phase)
				}, time.Minute*1, time.Second*5).Should(Equal("Failed"))
			}
		})

		It("should clean up domains when application is deleted", func() {
			By("Getting initial domain count")
			var initialDomains platformv1alpha1.ApplicationDomainList
			err := k8sClient.List(ctx, &initialDomains, client.InNamespace(testNamespace))
			Expect(err).NotTo(HaveOccurred())
			initialCount := len(initialDomains.Items)

			By("Deleting the application")
			Expect(k8sClient.Delete(ctx, testApplication)).To(Succeed())

			By("Verifying domains are cleaned up")
			Eventually(func() int {
				var domains platformv1alpha1.ApplicationDomainList
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return initialCount // Return initial count on error to continue waiting
				}

				// Count domains still referencing the deleted application
				remainingDomains := 0
				for _, domain := range domains.Items {
					if domain.Spec.ApplicationRef.Name == testApplication.Name {
						remainingDomains++
					}
				}
				return remainingDomains
			}, time.Minute*2, time.Second*5).Should(Equal(0))

			// Set to nil so AfterEach doesn't try to delete again
			testApplication = nil
		})
	})

	Context("ApplicationDomain Edge Cases", func() {
		It("should handle concurrent domain operations gracefully", func() {
			By("Creating multiple applications concurrently")
			applications := make([]*platformv1alpha1.Application, 3)

			for i := 0; i < 3; i++ {
				app := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("concurrent-app-%d", i),
						Namespace: testNamespace,
						Labels: map[string]string{
							validation.LabelResourceUUID: fmt.Sprintf("550e8400-e29b-41d4-a716-44665544000%d", i),
							validation.LabelResourceSlug: fmt.Sprintf("concurrent-app-%d", i),
							validation.LabelProjectUUID:  "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						Type: platformv1alpha1.ApplicationTypeGitRepository,
						ProjectRef: corev1.LocalObjectReference{
							Name: "non-existent-project", // This will cause creation but domain creation should still work
						},
						GitRepository: &platformv1alpha1.GitRepositoryConfig{
							Repository: fmt.Sprintf("https://github.com/test/app-%d", i),
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

			By("Verifying all applications eventually get domains created")
			Eventually(func() int {
				var domains platformv1alpha1.ApplicationDomainList
				err := k8sClient.List(ctx, &domains, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}

				domainCount := 0
				for _, domain := range domains.Items {
					for _, app := range applications {
						if domain.Spec.ApplicationRef.Name == app.Name {
							domainCount++
						}
					}
				}
				return domainCount
			}, time.Minute*3, time.Second*5).Should(BeNumerically(">=", 3))

			By("Cleaning up concurrent test applications")
			for _, app := range applications {
				_ = k8sClient.Delete(ctx, app)
			}
		})
	})
})
