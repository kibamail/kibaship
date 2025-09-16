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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

var _ = Describe("Project Lifecycle E2E Tests", Ordered, func() {
	var k8sClient client.Client
	ctx := context.Background()

	BeforeAll(func() {
		By("Setting up kubernetes client")
		cfg, err := config.GetConfig()
		Expect(err).NotTo(HaveOccurred())

		err = platformv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Project Creation and Initialization", func() {
		var testProject *platformv1alpha1.Project
		var expectedNamespaceName string

		AfterEach(func() {
			By("Cleaning up project and associated resources")
			if testProject != nil {
				// Delete the project
				_ = k8sClient.Delete(ctx, testProject)

				// Wait for project namespace to be deleted
				if expectedNamespaceName != "" {
					Eventually(func() bool {
						var ns corev1.Namespace
						err := k8sClient.Get(ctx, types.NamespacedName{Name: expectedNamespaceName}, &ns)
						return errors.IsNotFound(err)
					}, time.Minute*2, time.Second*5).Should(BeTrue())
				}
			}
		})

		It("should create project with full namespace and RBAC setup", func() {
			By("Creating a new project")
			testProject = &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project-lifecycle",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440000",
						validation.LabelResourceSlug:  "test-project-lifecycle",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}

			Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

			By("Waiting for project to become ready")
			Eventually(func() string {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				if err != nil {
					return ""
				}
				return project.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			By("Verifying project status contains namespace name")
			var updatedProject platformv1alpha1.Project
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &updatedProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedProject.Status.NamespaceName).To(ContainSubstring("project-"))
			Expect(updatedProject.Status.NamespaceName).To(ContainSubstring("-kibaship-com"))
			expectedNamespaceName = updatedProject.Status.NamespaceName

			By("Verifying dedicated namespace was created")
			var projectNamespace corev1.Namespace
			err = k8sClient.Get(ctx, types.NamespacedName{Name: expectedNamespaceName}, &projectNamespace)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying namespace has correct labels")
			Expect(projectNamespace.Labels).To(HaveKeyWithValue("platform.kibaship.com/managed-by", "kibaship-operator"))
			Expect(projectNamespace.Labels).To(HaveKeyWithValue("platform.kibaship.com/project", testProject.Name))
			Expect(projectNamespace.Labels).To(HaveKeyWithValue(validation.LabelResourceUUID, testProject.Labels[validation.LabelResourceUUID]))
			Expect(projectNamespace.Labels).To(HaveKeyWithValue(validation.LabelWorkspaceUUID, testProject.Labels[validation.LabelWorkspaceUUID]))

			By("Verifying namespace has correct annotations")
			Expect(projectNamespace.Annotations).To(HaveKeyWithValue("platform.kibaship.com/created-by", "kibaship-operator"))
			Expect(projectNamespace.Annotations).To(HaveKeyWithValue("platform.kibaship.com/project", testProject.Name))

			By("Verifying service account was created")
			var serviceAccount corev1.ServiceAccount
			serviceAccountName := fmt.Sprintf("project-%s-sa", testProject.Name)
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      serviceAccountName,
				Namespace: expectedNamespaceName,
			}, &serviceAccount)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying admin role was created")
			var adminRole rbacv1.Role
			adminRoleName := fmt.Sprintf("project-%s-admin", testProject.Name)
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      adminRoleName,
				Namespace: expectedNamespaceName,
			}, &adminRole)
			Expect(err).NotTo(HaveOccurred())

			// Verify role has comprehensive permissions
			Expect(len(adminRole.Rules)).To(BeNumerically(">", 0))
			foundAllResourcesRule := false
			for _, rule := range adminRole.Rules {
				if len(rule.Resources) > 0 && rule.Resources[0] == "*" {
					foundAllResourcesRule = true
					Expect(rule.Verbs).To(ContainElements("*"))
				}
			}
			Expect(foundAllResourcesRule).To(BeTrue(), "Admin role should have wildcard permissions")

			By("Verifying role binding was created")
			var roleBinding rbacv1.RoleBinding
			roleBindingName := fmt.Sprintf("project-%s-admin-binding", testProject.Name)
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      roleBindingName,
				Namespace: expectedNamespaceName,
			}, &roleBinding)
			Expect(err).NotTo(HaveOccurred())

			// Verify role binding connects service account to role
			Expect(roleBinding.Subjects).To(HaveLen(1))
			Expect(roleBinding.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(roleBinding.Subjects[0].Name).To(Equal(serviceAccountName))
			Expect(roleBinding.RoleRef.Kind).To(Equal("Role"))
			Expect(roleBinding.RoleRef.Name).To(Equal(adminRoleName))

			By("Verifying Tekton integration was created")
			var tektonRoleBinding rbacv1.RoleBinding
			tektonBindingName := fmt.Sprintf("project-%s-tekton-binding", testProject.Name)
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      tektonBindingName,
				Namespace: expectedNamespaceName,
			}, &tektonRoleBinding)
			Expect(err).NotTo(HaveOccurred())

			// Verify tekton role binding
			Expect(tektonRoleBinding.Subjects).To(HaveLen(1))
			Expect(tektonRoleBinding.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(tektonRoleBinding.Subjects[0].Name).To(Equal(serviceAccountName))
			Expect(tektonRoleBinding.Subjects[0].Namespace).To(Equal(expectedNamespaceName))
		})

		It("should handle project validation failures gracefully", func() {
			By("Attempting to create project without required UUID label")
			invalidProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalid-project-no-uuid",
					Labels: map[string]string{
						validation.LabelResourceSlug:  "invalid-project",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
						// Missing LabelResourceUUID
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}

			err := k8sClient.Create(ctx, invalidProject)
			if err == nil {
				// If creation succeeded, it should fail during reconciliation
				Eventually(func() string {
					var project platformv1alpha1.Project
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(invalidProject), &project)
					if err != nil {
						return "NotFound"
					}
					return project.Status.Phase
				}, time.Minute*1, time.Second*5).Should(Equal("Failed"))

				// Clean up
				_ = k8sClient.Delete(ctx, invalidProject)
			} else {
				// Should fail at creation time due to webhook validation
				Expect(err).To(HaveOccurred())
			}

			By("Attempting to create project with invalid UUID format")
			invalidUUIDProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalid-project-bad-uuid",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "invalid-uuid-format",
						validation.LabelResourceSlug:  "invalid-project",
						validation.LabelWorkspaceUUID: "also-invalid-uuid",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}

			err = k8sClient.Create(ctx, invalidUUIDProject)
			if err == nil {
				// If creation succeeded, it should fail during reconciliation
				Eventually(func() string {
					var project platformv1alpha1.Project
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(invalidUUIDProject), &project)
					if err != nil {
						return "NotFound"
					}
					return project.Status.Phase
				}, time.Minute*1, time.Second*5).Should(Equal("Failed"))

				// Clean up
				_ = k8sClient.Delete(ctx, invalidUUIDProject)
			} else {
				// Should fail at creation time due to webhook validation
				Expect(err).To(HaveOccurred())
			}
		})

		It("should handle namespace conflicts correctly", func() {
			By("Creating first project")
			firstProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "conflict-project-1",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440001",
						validation.LabelResourceSlug:  "conflict-project",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}

			Expect(k8sClient.Create(ctx, firstProject)).To(Succeed())

			Eventually(func() string {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(firstProject), &project)
				if err != nil {
					return ""
				}
				return project.Status.Phase
			}, time.Minute*2, time.Second*5).Should(Equal("Ready"))

			By("Attempting to create conflicting project with same slug")
			conflictProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "conflict-project-2",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440002",
						validation.LabelResourceSlug:  "conflict-project", // Same slug
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}

			err := k8sClient.Create(ctx, conflictProject)
			if err == nil {
				// If creation succeeded, it should fail during reconciliation due to namespace conflict
				Eventually(func() string {
					var project platformv1alpha1.Project
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(conflictProject), &project)
					if err != nil {
						return "NotFound"
					}
					return project.Status.Phase
				}, time.Minute*1, time.Second*5).Should(Equal("Failed"))

				// Clean up
				_ = k8sClient.Delete(ctx, conflictProject)
			}

			// Clean up first project
			_ = k8sClient.Delete(ctx, firstProject)
		})
	})

	Context("Project Deletion and Cleanup", func() {
		It("should perform complete cleanup when project is deleted", func() {
			By("Creating project with associated resources")
			testProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deletion-test-project",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440003",
						validation.LabelResourceSlug:  "deletion-test-project",
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

			// Get namespace name
			var readyProject platformv1alpha1.Project
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &readyProject)
			Expect(err).NotTo(HaveOccurred())
			projectNamespaceName := readyProject.Status.NamespaceName

			By("Creating application in project namespace")
			testApplication := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-for-deletion",
					Namespace: projectNamespaceName,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440004",
						validation.LabelResourceSlug: "test-app",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Provider:     "github.com",
						Repository:   "test/app",
						PublicAccess: true,
					},
				},
			}

			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			By("Verifying namespace and resources exist before deletion")
			var projectNamespace corev1.Namespace
			err = k8sClient.Get(ctx, types.NamespacedName{Name: projectNamespaceName}, &projectNamespace)
			Expect(err).NotTo(HaveOccurred())

			var serviceAccount corev1.ServiceAccount
			serviceAccountName := fmt.Sprintf("project-%s-sa", testProject.Name)
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      serviceAccountName,
				Namespace: projectNamespaceName,
			}, &serviceAccount)
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the project")
			Expect(k8sClient.Delete(ctx, testProject)).To(Succeed())

			By("Verifying project finalizer prevents immediate deletion")
			// Project should still exist but have deletion timestamp
			Eventually(func() bool {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				if err != nil {
					return false
				}
				return !project.DeletionTimestamp.IsZero()
			}, time.Second*30, time.Second*5).Should(BeTrue())

			By("Verifying namespace is eventually deleted")
			Eventually(func() bool {
				var ns corev1.Namespace
				err := k8sClient.Get(ctx, types.NamespacedName{Name: projectNamespaceName}, &ns)
				return errors.IsNotFound(err)
			}, time.Minute*3, time.Second*5).Should(BeTrue())

			By("Verifying project is eventually deleted")
			Eventually(func() bool {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				return errors.IsNotFound(err)
			}, time.Minute*3, time.Second*5).Should(BeTrue())

			By("Verifying associated applications are also deleted")
			Eventually(func() bool {
				var app platformv1alpha1.Application
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testApplication), &app)
				return errors.IsNotFound(err)
			}, time.Minute*1, time.Second*5).Should(BeTrue())
		})

		It("should handle deletion of project with multiple dependent resources", func() {
			By("Creating project")
			testProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "multi-resource-deletion-test",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440005",
						validation.LabelResourceSlug:  "multi-resource-deletion-test",
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

			var readyProject platformv1alpha1.Project
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &readyProject)
			Expect(err).NotTo(HaveOccurred())
			projectNamespaceName := readyProject.Status.NamespaceName

			By("Creating multiple applications")
			applications := make([]*platformv1alpha1.Application, 3)
			for i := 0; i < 3; i++ {
				app := &platformv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("multi-app-%d", i),
						Namespace: projectNamespaceName,
						Labels: map[string]string{
							validation.LabelResourceUUID: fmt.Sprintf("550e8400-e29b-41d4-a716-44665544001%d", i),
							validation.LabelResourceSlug: fmt.Sprintf("multi-app-%d", i),
							validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
						},
					},
					Spec: platformv1alpha1.ApplicationSpec{
						Type: platformv1alpha1.ApplicationTypeGitRepository,
						ProjectRef: corev1.LocalObjectReference{
							Name: testProject.Name,
						},
						GitRepository: &platformv1alpha1.GitRepositoryConfig{
							Provider:     "github.com",
							Repository:   fmt.Sprintf("test/app-%d", i),
							PublicAccess: true,
						},
					},
				}
				applications[i] = app
				Expect(k8sClient.Create(ctx, app)).To(Succeed())
			}

			By("Creating additional resources in namespace")
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: projectNamespaceName,
				},
				Data: map[string]string{
					"key": "value",
				},
			}
			Expect(k8sClient.Create(ctx, configMap)).To(Succeed())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: projectNamespaceName,
				},
				Data: map[string][]byte{
					"password": []byte("secret"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Deleting the project")
			Expect(k8sClient.Delete(ctx, testProject)).To(Succeed())

			By("Verifying all resources are eventually cleaned up")
			Eventually(func() bool {
				var ns corev1.Namespace
				err := k8sClient.Get(ctx, types.NamespacedName{Name: projectNamespaceName}, &ns)
				return errors.IsNotFound(err)
			}, time.Minute*5, time.Second*5).Should(BeTrue())

			Eventually(func() bool {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				return errors.IsNotFound(err)
			}, time.Minute*3, time.Second*5).Should(BeTrue())

			By("Verifying all applications are cleaned up")
			for _, app := range applications {
				Eventually(func() bool {
					var application platformv1alpha1.Application
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), &application)
					return errors.IsNotFound(err)
				}, time.Minute*1, time.Second*5).Should(BeTrue())
			}
		})
	})
})
