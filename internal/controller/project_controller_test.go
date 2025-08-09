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

var _ = Describe("Project Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name: resourceName,
		}
		project := &platformv1alpha1.Project{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Project")
			err := k8sClient.Get(ctx, typeNamespacedName, project)
			if err != nil && errors.IsNotFound(err) {
				resource := &platformv1alpha1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: resourceName,
						Labels: map[string]string{
							"platform.kibaship.com/uuid":           "550e8400-e29b-41d4-a716-446655440000",
							"platform.kibaship.com/workspace-uuid": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
						},
					},
					Spec: platformv1alpha1.ProjectSpec{},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &platformv1alpha1.Project{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Project")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource - first time (adds finalizer)")
			controllerReconciler := NewProjectReconciler(k8sClient, k8sClient.Scheme())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Reconciling again to complete initialization")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying project status is Ready")
			updatedProject := &platformv1alpha1.Project{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedProject.Status.Phase).To(Equal("Ready"))
			Expect(updatedProject.Status.NamespaceName).To(ContainSubstring("kibaship-project-"))

			By("Verifying that a namespace was created for the project")
			expectedNamespaceName := NamespacePrefix + resourceName
			namespace := &corev1.Namespace{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: expectedNamespaceName}, namespace)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying namespace has correct labels")
			Expect(namespace.Labels[ManagedByLabel]).To(Equal(ManagedByValue))
			Expect(namespace.Labels[ProjectNameLabel]).To(Equal(resourceName))
			Expect(namespace.Labels[ProjectUUIDLabel]).To(Equal("550e8400-e29b-41d4-a716-446655440000"))
			Expect(namespace.Labels[WorkspaceUUIDLabel]).To(Equal("6ba7b810-9dad-11d1-80b4-00c04fd430c8"))

			By("Verifying namespace has correct annotations")
			Expect(namespace.Annotations["platform.kibaship.com/created-by"]).To(Equal("kibaship-operator"))
			Expect(namespace.Annotations["platform.kibaship.com/project"]).To(Equal(resourceName))

			By("Cleaning up the created namespace")
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		})

		It("should fail validation when platform.kibaship.com/uuid label is missing", func() {
			By("Creating a project without the required UUID label")
			invalidResource := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalid-resource",
					Labels: map[string]string{
						"platform.kibaship.com/workspace-uuid": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, invalidResource)).To(Succeed())

			controllerReconciler := NewProjectReconciler(k8sClient, k8sClient.Scheme())

			// First reconcile adds finalizer
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "invalid-resource",
				},
			})
			Expect(err).NotTo(HaveOccurred()) // First reconcile just adds finalizer

			// Second reconcile should fail validation
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "invalid-resource",
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("required label 'platform.kibaship.com/uuid' is missing"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, invalidResource)).To(Succeed())
		})

		It("should fail validation when UUID labels have invalid format", func() {
			By("Creating a project with invalid UUID format")
			invalidResource := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalid-uuid-resource",
					Labels: map[string]string{
						"platform.kibaship.com/uuid":           "invalid-uuid",
						"platform.kibaship.com/workspace-uuid": "also-invalid",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, invalidResource)).To(Succeed())

			controllerReconciler := NewProjectReconciler(k8sClient, k8sClient.Scheme())

			// First reconcile adds finalizer
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "invalid-uuid-resource",
				},
			})
			Expect(err).NotTo(HaveOccurred()) // First reconcile just adds finalizer

			// Second reconcile should fail validation
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "invalid-uuid-resource",
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be a valid UUID"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, invalidResource)).To(Succeed())
		})

		It("should fail when project name conflicts with existing namespace", func() {
			By("Creating a conflicting namespace first")
			conflictingNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: NamespacePrefix + "conflicting-project",
					Labels: map[string]string{
						ManagedByLabel:   ManagedByValue,
						ProjectNameLabel: "conflicting-project",
						ProjectUUIDLabel: "different-uuid-1234-5678-9abc-def012345678",
					},
				},
			}
			Expect(k8sClient.Create(ctx, conflictingNamespace)).To(Succeed())

			By("Attempting to create a project with conflicting name")
			conflictingProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "conflicting-project",
					Labels: map[string]string{
						ProjectUUIDLabel:   "550e8400-e29b-41d4-a716-446655440001",
						WorkspaceUUIDLabel: "6ba7b810-9dad-11d1-80b4-00c04fd430c9",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, conflictingProject)).To(Succeed())

			controllerReconciler := NewProjectReconciler(k8sClient, k8sClient.Scheme())

			// First reconcile adds finalizer
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "conflicting-project",
				},
			})
			Expect(err).NotTo(HaveOccurred()) // First reconcile just adds finalizer

			// Second reconcile should fail due to namespace conflict
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "conflicting-project",
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("conflicting namespace"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, conflictingProject)).To(Succeed())
			Expect(k8sClient.Delete(ctx, conflictingNamespace)).To(Succeed())
		})

		It("should create namespace and add finalizer to project", func() {
			By("Creating a project")
			testProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "owner-ref-project",
					Labels: map[string]string{
						ProjectUUIDLabel:   "550e8400-e29b-41d4-a716-446655440002",
						WorkspaceUUIDLabel: "6ba7b810-9dad-11d1-80b4-00c04fd430ca",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

			By("Reconciling the project - first time (adds finalizer)")
			controllerReconciler := NewProjectReconciler(k8sClient, k8sClient.Scheme())
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "owner-ref-project",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying project has finalizer after first reconcile")
			updatedProject := &platformv1alpha1.Project{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "owner-ref-project"}, updatedProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedProject.Finalizers).To(ContainElement(ProjectFinalizerName))

			By("Reconciling again to complete initialization")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "owner-ref-project",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying namespace was created")
			namespaceName := NamespacePrefix + "owner-ref-project"
			namespace := &corev1.Namespace{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying project status is Ready")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "owner-ref-project"}, updatedProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedProject.Status.Phase).To(Equal("Ready"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, testProject)).To(Succeed())
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		})
	})
})
