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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/validation"
)

var _ = Describe("Tekton Integration", func() {
	var (
		ctx              context.Context
		namespaceManager *NamespaceManager
		testProject      *platformv1alpha1.Project
		testNamespace    *corev1.Namespace
		tektonNamespace  *corev1.Namespace
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespaceManager = NewNamespaceManager(k8sClient)

		// Create tekton-pipelines namespace
		tektonNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: TektonNamespace,
			},
		}
		err := k8sClient.Create(ctx, tektonNamespace)
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		// Create test project
		testProject = &platformv1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tekton-test-project",
				Labels: map[string]string{
					validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440030",
					validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430d0",
				},
			},
			Spec: platformv1alpha1.ProjectSpec{},
		}
		Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

		// Create project namespace with all resources including Tekton integration
		testNamespace, err = namespaceManager.CreateProjectNamespace(ctx, testProject)
		Expect(err).NotTo(HaveOccurred())
		Expect(testNamespace).NotTo(BeNil())
	})

	AfterEach(func() {
		// Clean up test namespace
		if testNamespace != nil {
			err := k8sClient.Delete(ctx, testNamespace)
			if err != nil && !errors.IsNotFound(err) {
				GinkgoWriter.Printf("Failed to clean up namespace: %v\n", err)
			}
		}

		// Clean up test project
		if testProject != nil {
			err := k8sClient.Delete(ctx, testProject)
			if err != nil && !errors.IsNotFound(err) {
				GinkgoWriter.Printf("Failed to clean up project: %v\n", err)
			}
		}

		// Don't delete tekton namespace - it's shared across tests and will be cleaned up by test env

		// Wait for cleanup to complete
		if testProject != nil {
			Eventually(func() bool {
				project := &platformv1alpha1.Project{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: testProject.Name}, project)
				return errors.IsNotFound(err)
			}).Should(BeTrue())
		}
	})

	Describe("Role Creation", func() {
		It("should create tekton-tasks-reader role in tekton-pipelines namespace", func() {
			By("Verifying the Tekton role was created")
			role := &rbacv1.Role{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      TektonRoleName,
				Namespace: TektonNamespace,
			}, role)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the role has correct permissions")
			Expect(role.Rules).To(HaveLen(1))
			rule := role.Rules[0]
			Expect(rule.APIGroups).To(ContainElement("tekton.dev"))
			Expect(rule.Resources).To(ContainElement("tasks"))
			Expect(rule.Verbs).To(ContainElements("get", "list", "watch"))

			By("Verifying the role has correct labels and annotations")
			Expect(role.Labels[ManagedByLabel]).To(Equal(ManagedByValue))
			Expect(role.Annotations["platform.kibaship.com/created-by"]).To(Equal("kibaship"))
		})

		It("should not create duplicate tekton-tasks-reader role", func() {
			By("Creating another project to test role reuse")
			secondProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tekton-test-second-project",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440031",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430d0",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, secondProject)).To(Succeed())

			By("Creating namespace for second project")
			secondNamespace, err := namespaceManager.CreateProjectNamespace(ctx, secondProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(secondNamespace).NotTo(BeNil())

			By("Verifying only one Tekton role exists")
			roleList := &rbacv1.RoleList{}
			err = k8sClient.List(ctx, roleList, client.InNamespace(TektonNamespace), client.MatchingLabels{
				ManagedByLabel: ManagedByValue,
			})
			Expect(err).NotTo(HaveOccurred())

			tektonRoles := 0
			for _, role := range roleList.Items {
				if role.Name == TektonRoleName {
					tektonRoles++
				}
			}
			Expect(tektonRoles).To(Equal(1), "Should have exactly one tekton-tasks-reader role")

			By("Cleaning up second project resources")
			namespaceManager.deleteServiceAccountResources(ctx, secondNamespace, secondProject)
			_ = k8sClient.Delete(ctx, secondNamespace)
			_ = k8sClient.Delete(ctx, secondProject)
		})
	})

	Describe("Role Binding Creation", func() {
		It("should create role binding from project service account to tekton role", func() {
			By("Verifying the Tekton role binding was created")
			roleBindingName := namespaceManager.generateTektonRoleBindingName(testProject.Labels[validation.LabelResourceUUID])
			roleBinding := &rbacv1.RoleBinding{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      roleBindingName,
				Namespace: TektonNamespace,
			}, roleBinding)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the role binding has correct subject")
			Expect(roleBinding.Subjects).To(HaveLen(1))
			subject := roleBinding.Subjects[0]
			Expect(subject.Kind).To(Equal("ServiceAccount"))
			Expect(subject.Name).To(Equal(namespaceManager.generateServiceAccountName(testProject.Labels[validation.LabelResourceUUID])))
			Expect(subject.Namespace).To(Equal(testNamespace.Name))

			By("Verifying the role binding has correct role reference")
			Expect(roleBinding.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
			Expect(roleBinding.RoleRef.Kind).To(Equal("Role"))
			Expect(roleBinding.RoleRef.Name).To(Equal(TektonRoleName))

			By("Verifying the role binding has correct labels")
			Expect(roleBinding.Labels[ManagedByLabel]).To(Equal(ManagedByValue))
			Expect(roleBinding.Labels[ProjectNameLabel]).To(Equal(testProject.Name))
		})
	})

	Describe("Resource Cleanup", func() {
		It("should clean up Tekton role binding when deleting project resources", func() {
			By("Verifying the Tekton role binding exists before cleanup")
			roleBindingName := namespaceManager.generateTektonRoleBindingName(testProject.Labels[validation.LabelResourceUUID])
			roleBinding := &rbacv1.RoleBinding{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      roleBindingName,
				Namespace: TektonNamespace,
			}, roleBinding)
			Expect(err).NotTo(HaveOccurred())

			By("Cleaning up service account resources")
			namespaceManager.deleteServiceAccountResources(ctx, testNamespace, testProject)

			By("Verifying the Tekton role binding was deleted")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      roleBindingName,
				Namespace: TektonNamespace,
			}, roleBinding)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})
