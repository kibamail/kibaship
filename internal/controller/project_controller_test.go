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
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		project := &platformv1alpha1.Project{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Project")
			err := k8sClient.Get(ctx, typeNamespacedName, project)
			if err != nil && errors.IsNotFound(err) {
				resource := &platformv1alpha1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
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
			By("Reconciling the created resource")
			controllerReconciler := &ProjectReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation when platform.kibaship.com/uuid label is missing", func() {
			By("Creating a project without the required UUID label")
			invalidResource := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-resource",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/workspace-uuid": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, invalidResource)).To(Succeed())

			controllerReconciler := &ProjectReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "invalid-resource",
					Namespace: "default",
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
					Name:      "invalid-uuid-resource",
					Namespace: "default",
					Labels: map[string]string{
						"platform.kibaship.com/uuid":           "invalid-uuid",
						"platform.kibaship.com/workspace-uuid": "also-invalid",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}
			Expect(k8sClient.Create(ctx, invalidResource)).To(Succeed())

			controllerReconciler := &ProjectReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "invalid-uuid-resource",
					Namespace: "default",
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be a valid UUID"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, invalidResource)).To(Succeed())
		})
	})
})
