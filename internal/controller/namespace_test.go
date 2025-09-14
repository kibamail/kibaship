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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

var _ = Describe("NamespaceManager", func() {
	var (
		ctx              context.Context
		namespaceManager *NamespaceManager
		testProject      *platformv1alpha1.Project
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespaceManager = NewNamespaceManager(k8sClient)

		// Generate unique project name to avoid conflicts between tests
		uniqueID := time.Now().UnixNano()
		testProject = &platformv1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("test-namespace-project-%d", uniqueID),
				Labels: map[string]string{
					ProjectUUIDLabel:   "550e8400-e29b-41d4-a716-446655440010",
					WorkspaceUUIDLabel: "6ba7b810-9dad-11d1-80b4-00c04fd430d0",
				},
			},
			Spec: platformv1alpha1.ProjectSpec{},
		}
		Expect(k8sClient.Create(ctx, testProject)).To(Succeed())
	})

	AfterEach(func() {
		// Cleanup any namespaces that might have been created
		if testProject != nil {
			namespaceName := namespaceManager.GenerateNamespaceName(testProject.Name)
			namespace := &corev1.Namespace{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace); err == nil {
				k8sClient.Delete(ctx, namespace)
			}
		}

		// Cleanup test project
		if testProject != nil {
			k8sClient.Delete(ctx, testProject)
		}

		// Wait for cleanup to complete
		if testProject != nil {
			Eventually(func() bool {
				project := &platformv1alpha1.Project{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: testProject.Name}, project)
				return errors.IsNotFound(err)
			}).Should(BeTrue())
		}
	})

	Describe("GenerateNamespaceName", func() {
		It("should generate correct namespace name with prefix and suffix", func() {
			result := namespaceManager.GenerateNamespaceName("my-project")
			Expect(result).To(Equal("project-my-project-kibaship-com"))
		})

		It("should handle project names with hyphens", func() {
			result := namespaceManager.GenerateNamespaceName("my-test-project")
			Expect(result).To(Equal("project-my-test-project-kibaship-com"))
		})
	})

	Describe("CreateProjectNamespace", func() {
		It("should successfully create a namespace for a project", func() {
			By("Creating the namespace")
			namespace, err := namespaceManager.CreateProjectNamespace(ctx, testProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(namespace).NotTo(BeNil())

			By("Verifying namespace properties")
			expectedName := namespaceManager.GenerateNamespaceName(testProject.Name)
			Expect(namespace.Name).To(Equal(expectedName))

			By("Verifying namespace labels")
			Expect(namespace.Labels[ManagedByLabel]).To(Equal(ManagedByValue))
			Expect(namespace.Labels[ProjectNameLabel]).To(Equal(testProject.Name))
			Expect(namespace.Labels[ProjectUUIDLabel]).To(Equal("550e8400-e29b-41d4-a716-446655440010"))
			Expect(namespace.Labels[WorkspaceUUIDLabel]).To(Equal("6ba7b810-9dad-11d1-80b4-00c04fd430d0"))

			By("Verifying namespace annotations")
			Expect(namespace.Annotations["platform.kibaship.com/created-by"]).To(Equal("kibaship-operator"))
			Expect(namespace.Annotations["platform.kibaship.com/project"]).To(Equal(testProject.Name))

			By("Verifying namespace exists in cluster")
			retrievedNamespace := &corev1.Namespace{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: expectedName}, retrievedNamespace)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return existing namespace if it belongs to the same project", func() {
			By("Creating the namespace first time")
			namespace1, err := namespaceManager.CreateProjectNamespace(ctx, testProject)
			Expect(err).NotTo(HaveOccurred())

			By("Creating the namespace second time")
			namespace2, err := namespaceManager.CreateProjectNamespace(ctx, testProject)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying it's the same namespace")
			Expect(namespace1.Name).To(Equal(namespace2.Name))
			Expect(namespace1.UID).To(Equal(namespace2.UID))
		})

		It("should fail if namespace exists for different project", func() {
			By("Creating a namespace manually with different project UUID")
			conflictingNamespaceName := "project-conflict-test-kibaship-com"
			conflictingNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: conflictingNamespaceName,
					Labels: map[string]string{
						ManagedByLabel:   ManagedByValue,
						ProjectNameLabel: "conflict-test",
						ProjectUUIDLabel: "different-uuid-1234-5678-9abc-def012345678",
					},
				},
			}
			Expect(k8sClient.Create(ctx, conflictingNamespace)).To(Succeed())

			// Create a test project that would conflict
			conflictProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "conflict-test",
					Labels: map[string]string{
						ProjectUUIDLabel: "550e8400-e29b-41d4-a716-446655440099",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}

			By("Attempting to create namespace for conflicting project")
			_, err := namespaceManager.CreateProjectNamespace(ctx, conflictProject)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists but belongs to different project"))

			By("Cleaning up conflicting namespace")
			Expect(k8sClient.Delete(ctx, conflictingNamespace)).To(Succeed())
		})
	})

	Describe("GetProjectNamespace", func() {
		It("should retrieve existing project namespace", func() {
			By("Creating the namespace")
			createdNamespace, err := namespaceManager.CreateProjectNamespace(ctx, testProject)
			Expect(err).NotTo(HaveOccurred())

			By("Retrieving the namespace")
			retrievedNamespace, err := namespaceManager.GetProjectNamespace(ctx, testProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedNamespace.Name).To(Equal(createdNamespace.Name))
			Expect(retrievedNamespace.UID).To(Equal(createdNamespace.UID))
		})

		It("should return error if namespace doesn't exist", func() {
			// Use a project name that definitely won't have a namespace
			nonExistentProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "non-existent-project-12345",
					Labels: map[string]string{
						ProjectUUIDLabel: "550e8400-e29b-41d4-a716-446655440999",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}

			_, err := namespaceManager.GetProjectNamespace(ctx, nonExistentProject)
			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})

	Describe("IsProjectNamespaceUnique", func() {
		It("should return true for unique project name", func() {
			isUnique, err := namespaceManager.IsProjectNamespaceUnique(ctx, "unique-project-name", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(isUnique).To(BeTrue())
		})

		It("should return false if namespace exists for different project", func() {
			By("Creating a namespace for different project")
			existingNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceManager.GenerateNamespaceName("existing-project"),
					Labels: map[string]string{
						ManagedByLabel:   ManagedByValue,
						ProjectNameLabel: "existing-project",
						ProjectUUIDLabel: "different-uuid-1234-5678-9abc-def012345678",
					},
				},
			}
			Expect(k8sClient.Create(ctx, existingNamespace)).To(Succeed())

			By("Checking uniqueness")
			isUnique, err := namespaceManager.IsProjectNamespaceUnique(ctx, "existing-project", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(isUnique).To(BeFalse())

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, existingNamespace)).To(Succeed())
		})

		It("should return true if namespace exists for same project (exclude case)", func() {
			By("Creating the namespace")
			_, err := namespaceManager.CreateProjectNamespace(ctx, testProject)
			Expect(err).NotTo(HaveOccurred())

			By("Checking uniqueness with exclusion")
			isUnique, err := namespaceManager.IsProjectNamespaceUnique(ctx, testProject.Name, testProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(isUnique).To(BeTrue())
		})
	})

	Describe("ListProjectNamespaces", func() {
		It("should list all project-managed namespaces", func() {
			By("Creating multiple project namespaces")
			project1 := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "list-test-project-1",
					Labels: map[string]string{
						ProjectUUIDLabel: "550e8400-e29b-41d4-a716-446655440011",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, project1)).To(Succeed())

			project2 := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "list-test-project-2",
					Labels: map[string]string{
						ProjectUUIDLabel: "550e8400-e29b-41d4-a716-446655440012",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, project2)).To(Succeed())

			_, err := namespaceManager.CreateProjectNamespace(ctx, project1)
			Expect(err).NotTo(HaveOccurred())

			_, err = namespaceManager.CreateProjectNamespace(ctx, project2)
			Expect(err).NotTo(HaveOccurred())

			By("Listing project namespaces")
			projectValidator := NewProjectValidator(k8sClient)
			namespaces, err := projectValidator.ListProjectNamespaces(ctx)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying we found the created namespaces")
			var foundNames []string
			for _, ns := range namespaces {
				foundNames = append(foundNames, ns.Name)
			}
			Expect(foundNames).To(ContainElement("project-list-test-project-1-kibaship-com"))
			Expect(foundNames).To(ContainElement("project-list-test-project-2-kibaship-com"))

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, project1)).To(Succeed())
			Expect(k8sClient.Delete(ctx, project2)).To(Succeed())
		})
	})

	Describe("CreateProjectServiceAccount", func() {
		var testNamespace *corev1.Namespace

		BeforeEach(func() {
			// Create a test namespace first
			var err error
			testNamespace, err = namespaceManager.CreateProjectNamespace(ctx, testProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(testNamespace).NotTo(BeNil())
		})

		AfterEach(func() {
			// Clean up namespace (this will also clean up all namespace-scoped resources)
			if testNamespace != nil {
				err := k8sClient.Delete(ctx, testNamespace)
				if err != nil && !errors.IsNotFound(err) {
					// Ignore cleanup errors in tests
				}
			}
		})

		It("should create service account with all required resources", func() {
			By("Verifying service account was created")
			serviceAccount := &corev1.ServiceAccount{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      namespaceManager.generateServiceAccountName(testProject.Name),
				Namespace: testNamespace.Name,
			}, serviceAccount)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service account has correct labels")
			Expect(serviceAccount.Labels[ManagedByLabel]).To(Equal(ManagedByValue))
			Expect(serviceAccount.Labels[ProjectNameLabel]).To(Equal(testProject.Name))
			Expect(serviceAccount.Labels[ProjectUUIDLabel]).To(Equal("550e8400-e29b-41d4-a716-446655440010"))
			Expect(serviceAccount.Labels[WorkspaceUUIDLabel]).To(Equal("6ba7b810-9dad-11d1-80b4-00c04fd430d0"))

			By("Verifying service account has correct annotations")
			Expect(serviceAccount.Annotations["platform.kibaship.com/created-by"]).To(Equal("kibaship-operator"))
			Expect(serviceAccount.Annotations["platform.kibaship.com/project"]).To(Equal(testProject.Name))
		})

		It("should create admin role with all permissions", func() {
			By("Verifying role was created")
			role := &rbacv1.Role{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      namespaceManager.generateRoleName(testProject.Name),
				Namespace: testNamespace.Name,
			}, role)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying role has all permissions")
			Expect(role.Rules).To(HaveLen(1))
			rule := role.Rules[0]
			Expect(rule.APIGroups).To(ContainElement("*"))
			Expect(rule.Resources).To(ContainElement("*"))
			Expect(rule.Verbs).To(ContainElement("*"))

			By("Verifying role has correct labels")
			Expect(role.Labels[ManagedByLabel]).To(Equal(ManagedByValue))
			Expect(role.Labels[ProjectNameLabel]).To(Equal(testProject.Name))
			Expect(role.Labels[ProjectUUIDLabel]).To(Equal("550e8400-e29b-41d4-a716-446655440010"))
			Expect(role.Labels[WorkspaceUUIDLabel]).To(Equal("6ba7b810-9dad-11d1-80b4-00c04fd430d0"))
		})

		It("should create role binding connecting service account to role", func() {
			By("Verifying role binding was created")
			roleBinding := &rbacv1.RoleBinding{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      namespaceManager.generateRoleBindingName(testProject.Name),
				Namespace: testNamespace.Name,
			}, roleBinding)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying role binding has correct subject")
			Expect(roleBinding.Subjects).To(HaveLen(1))
			subject := roleBinding.Subjects[0]
			Expect(subject.Kind).To(Equal("ServiceAccount"))
			Expect(subject.Name).To(Equal(namespaceManager.generateServiceAccountName(testProject.Name)))
			Expect(subject.Namespace).To(Equal(testNamespace.Name))

			By("Verifying role binding has correct role reference")
			Expect(roleBinding.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
			Expect(roleBinding.RoleRef.Kind).To(Equal("Role"))
			Expect(roleBinding.RoleRef.Name).To(Equal(namespaceManager.generateRoleName(testProject.Name)))

			By("Verifying role binding has correct labels")
			Expect(roleBinding.Labels[ManagedByLabel]).To(Equal(ManagedByValue))
			Expect(roleBinding.Labels[ProjectNameLabel]).To(Equal(testProject.Name))
			Expect(roleBinding.Labels[ProjectUUIDLabel]).To(Equal("550e8400-e29b-41d4-a716-446655440010"))
			Expect(roleBinding.Labels[WorkspaceUUIDLabel]).To(Equal("6ba7b810-9dad-11d1-80b4-00c04fd430d0"))
		})

	})

	Describe("deleteServiceAccountResources", func() {
		var testNamespace *corev1.Namespace

		BeforeEach(func() {
			// Create a test namespace with service account resources
			var err error
			testNamespace, err = namespaceManager.CreateProjectNamespace(ctx, testProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(testNamespace).NotTo(BeNil())
		})

		AfterEach(func() {
			// Clean up namespace
			if testNamespace != nil {
				err := k8sClient.Delete(ctx, testNamespace)
				if err != nil && !errors.IsNotFound(err) {
					// Ignore cleanup errors in tests
				}
			}
		})

		It("should clean up all service account resources", func() {
			By("Verifying resources exist before cleanup")
			serviceAccount := &corev1.ServiceAccount{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      namespaceManager.generateServiceAccountName(testProject.Name),
				Namespace: testNamespace.Name,
			}, serviceAccount)
			Expect(err).NotTo(HaveOccurred())

			By("Cleaning up service account resources")
			namespaceManager.deleteServiceAccountResources(ctx, testNamespace, testProject)

			By("Verifying service account was deleted")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      namespaceManager.generateServiceAccountName(testProject.Name),
				Namespace: testNamespace.Name,
			}, serviceAccount)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			By("Verifying role was deleted")
			role := &rbacv1.Role{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      namespaceManager.generateRoleName(testProject.Name),
				Namespace: testNamespace.Name,
			}, role)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			By("Verifying role binding was deleted")
			roleBinding := &rbacv1.RoleBinding{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      namespaceManager.generateRoleBindingName(testProject.Name),
				Namespace: testNamespace.Name,
			}, roleBinding)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})

})
