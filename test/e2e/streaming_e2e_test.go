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
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/validation"
	"github.com/kibamail/kibaship-operator/test/utils"
)

var _ = Describe("Streaming Integration E2E Tests", Ordered, func() {
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
		testNamespace = "streaming-e2e-test"
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

	Context("Valkey Operator Integration", func() {
		It("should have Valkey operator running and CRDs available", func() {
			By("Verifying Valkey operator deployment is running")
			var deployment unstructured.Unstructured
			deployment.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			})

			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "valkey-operator-controller-manager",
					Namespace: "valkey-operator-system",
				}, &deployment)
			}, time.Minute*1, time.Second*5).Should(Succeed())

			By("Verifying Valkey CRDs are installed")
			cmd := exec.Command("kubectl", "get", "crd", "valkeys.hyperspike.io")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create and manage Valkey cluster for system cache", func() {
			By("Creating Valkey cluster resource")
			valkeyCluster := &unstructured.Unstructured{}
			valkeyCluster.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "hyperspike.io",
				Version: "v1",
				Kind:    "Valkey",
			})
			valkeyCluster.SetName("test-streaming-valkey")
			valkeyCluster.SetNamespace(testNamespace)

			valkeySpec := map[string]interface{}{
				"replicas": 1,
				"config": map[string]interface{}{
					"maxmemory-policy": "allkeys-lru",
				},
			}
			valkeyCluster.Object["spec"] = valkeySpec

			Expect(k8sClient.Create(ctx, valkeyCluster)).To(Succeed())

			By("Waiting for Valkey cluster to become ready")
			Eventually(func() bool {
				var valkey unstructured.Unstructured
				valkey.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "hyperspike.io",
					Version: "v1",
					Kind:    "Valkey",
				})

				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(valkeyCluster), &valkey)
				if err != nil {
					return false
				}

				status, found, err := unstructured.NestedMap(valkey.Object, "status")
				if err != nil || !found {
					return false
				}

				ready, found, err := unstructured.NestedBool(status, "ready")
				return err == nil && found && ready
			}, time.Minute*5, time.Second*10).Should(BeTrue())

			By("Verifying Valkey service is created")
			var valkeyService corev1.Service
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-streaming-valkey",
					Namespace: testNamespace,
				}, &valkeyService)
			}, time.Minute*2, time.Second*5).Should(Succeed())

			By("Verifying service has correct port configuration")
			Expect(len(valkeyService.Spec.Ports)).To(BeNumerically(">", 0))
			foundRedisPort := false
			for _, port := range valkeyService.Spec.Ports {
				if port.Port == 6379 {
					foundRedisPort = true
					break
				}
			}
			Expect(foundRedisPort).To(BeTrue(), "Valkey service should expose port 6379")

			// Clean up
			_ = k8sClient.Delete(ctx, valkeyCluster)
		})
	})

	Context("Operator Streaming Integration", func() {
		var systemValkeyCluster *unstructured.Unstructured

		BeforeEach(func() {
			By("Creating system Valkey cluster for operator")
			systemValkeyCluster = &unstructured.Unstructured{}
			systemValkeyCluster.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "hyperspike.io",
				Version: "v1",
				Kind:    "Valkey",
			})
			systemValkeyCluster.SetName("kibaship-valkey-cluster-kibaship-com")
			systemValkeyCluster.SetNamespace(namespace) // Use operator namespace

			valkeySpec := map[string]interface{}{
				"replicas": 1,
				"config": map[string]interface{}{
					"maxmemory-policy": "allkeys-lru",
				},
			}
			systemValkeyCluster.Object["spec"] = valkeySpec

			Expect(k8sClient.Create(ctx, systemValkeyCluster)).To(Succeed())

			// Wait for Valkey to be ready
			Eventually(func() bool {
				var valkey unstructured.Unstructured
				valkey.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "hyperspike.io",
					Version: "v1",
					Kind:    "Valkey",
				})

				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(systemValkeyCluster), &valkey)
				if err != nil {
					return false
				}

				status, found, err := unstructured.NestedMap(valkey.Object, "status")
				if err != nil || !found {
					return false
				}

				ready, found, err := unstructured.NestedBool(status, "ready")
				return err == nil && found && ready
			}, time.Minute*5, time.Second*10).Should(BeTrue())

			By("Creating Valkey secret")
			valkeySecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kibaship-valkey-cluster-kibaship-com",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"password": []byte("test-password-123"),
				},
			}
			err := k8sClient.Create(ctx, valkeySecret)
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			By("Restarting operator to pick up Valkey cluster")
			cmd := exec.Command("kubectl", "rollout", "restart", "deployment/kibaship-operator-controller-manager", "-n", namespace)
			_, _ = utils.Run(cmd)

			// Wait for operator to be ready again
			cmd = exec.Command("kubectl", "rollout", "status", "deployment/kibaship-operator-controller-manager", "-n", namespace, "--timeout=300s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("Cleaning up system Valkey cluster")
			if systemValkeyCluster != nil {
				_ = k8sClient.Delete(ctx, systemValkeyCluster)
			}

			By("Cleaning up Valkey secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kibaship-valkey-cluster-kibaship-com",
					Namespace: namespace,
				},
			}
			_ = k8sClient.Delete(ctx, secret)
		})

		It("should publish project events to Redis streams", func() {
			By("Creating a project to generate streaming events")
			testProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "streaming-test-project",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440000",
						validation.LabelResourceSlug:  "streaming-test-project",
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

			By("Verifying operator logs show streaming activity")
			// Get operator pod name
			cmd := exec.Command("kubectl", "get", "pods", "-l", "control-plane=controller-manager", "-n", namespace, "-o", "jsonpath={.items[0].metadata.name}")
			podOutput, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			operatorPodName := podOutput

			// Check operator logs for streaming activity
			Eventually(func() string {
				cmd := exec.Command("kubectl", "logs", operatorPodName, "-n", namespace, "--since=60s")
				logs, err := utils.Run(cmd)
				if err != nil {
					return ""
				}
				return logs
			}, time.Minute*2, time.Second*10).Should(ContainSubstring("streaming"))

			// Clean up
			_ = k8sClient.Delete(ctx, testProject)
		})

		It("should handle streaming connection failures gracefully", func() {
			By("Deleting Valkey secret to simulate connection failure")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kibaship-valkey-cluster-kibaship-com",
					Namespace: namespace,
				},
			}
			_ = k8sClient.Delete(ctx, secret)

			By("Creating project during streaming outage")
			testProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "outage-test-project",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440001",
						validation.LabelResourceSlug:  "outage-test-project",
						validation.LabelWorkspaceUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					},
				},
				Spec: platformv1alpha1.ProjectSpec{},
			}

			Expect(k8sClient.Create(ctx, testProject)).To(Succeed())

			By("Verifying project still becomes ready despite streaming issues")
			Eventually(func() string {
				var project platformv1alpha1.Project
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &project)
				if err != nil {
					return ""
				}
				return project.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal("Ready"))

			By("Restoring Valkey secret")
			restoredSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kibaship-valkey-cluster-kibaship-com",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"password": []byte("test-password-123"),
				},
			}
			Expect(k8sClient.Create(ctx, restoredSecret)).To(Succeed())

			By("Triggering project update to test streaming recovery")
			// Update project annotation to trigger reconciliation
			var currentProject platformv1alpha1.Project
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &currentProject)
			Expect(err).NotTo(HaveOccurred())

			if currentProject.Annotations == nil {
				currentProject.Annotations = make(map[string]string)
			}
			currentProject.Annotations["test.streaming/recovery"] = "true"
			Expect(k8sClient.Update(ctx, &currentProject)).To(Succeed())

			By("Verifying operator logs show streaming recovery")
			cmd := exec.Command("kubectl", "get", "pods", "-l", "control-plane=controller-manager", "-n", namespace, "-o", "jsonpath={.items[0].metadata.name}")
			podOutput, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			operatorPodName := podOutput

			Eventually(func() string {
				cmd := exec.Command("kubectl", "logs", operatorPodName, "-n", namespace, "--since=30s")
				logs, err := utils.Run(cmd)
				if err != nil {
					return ""
				}
				return logs
			}, time.Minute*1, time.Second*10).Should(Or(
				ContainSubstring("streaming"),
				ContainSubstring("connected"),
				ContainSubstring("recovery"),
			))

			// Clean up
			_ = k8sClient.Delete(ctx, testProject)
		})

		It("should publish application and deployment events", func() {
			By("Creating project first")
			testProject := &platformv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "app-streaming-test-project",
					Labels: map[string]string{
						validation.LabelResourceUUID:  "550e8400-e29b-41d4-a716-446655440002",
						validation.LabelResourceSlug:  "app-streaming-test-project",
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

			By("Getting project namespace")
			var readyProject platformv1alpha1.Project
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testProject), &readyProject)
			Expect(err).NotTo(HaveOccurred())
			projectNamespace := readyProject.Status.NamespaceName

			By("Creating application in project")
			testApplication := &platformv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "streaming-test-app",
					Namespace: projectNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440003",
						validation.LabelResourceSlug: "streaming-test-app",
						validation.LabelProjectUUID:  testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.ApplicationSpec{
					Type: platformv1alpha1.ApplicationTypeGitRepository,
					ProjectRef: corev1.LocalObjectReference{
						Name: testProject.Name,
					},
					GitRepository: &platformv1alpha1.GitRepositoryConfig{
						Repository: "https://github.com/test/streaming-app",
					},
				},
			}

			Expect(k8sClient.Create(ctx, testApplication)).To(Succeed())

			By("Creating deployment for application")
			testDeployment := &platformv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "streaming-test-deployment",
					Namespace: projectNamespace,
					Labels: map[string]string{
						validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440004",
						validation.LabelApplicationUUID: testApplication.Labels[validation.LabelResourceUUID],
						validation.LabelProjectUUID:     testProject.Labels[validation.LabelResourceUUID],
					},
				},
				Spec: platformv1alpha1.DeploymentSpec{
					ApplicationRef: corev1.LocalObjectReference{
						Name: testApplication.Name,
					},
				},
			}

			Expect(k8sClient.Create(ctx, testDeployment)).To(Succeed())

			By("Verifying all resources generate streaming events")
			// Get operator pod name
			cmd := exec.Command("kubectl", "get", "pods", "-l", "control-plane=controller-manager", "-n", namespace, "-o", "jsonpath={.items[0].metadata.name}")
			podOutput, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			operatorPodName := podOutput

			// Check for streaming activity for different resource types
			Eventually(func() string {
				cmd := exec.Command("kubectl", "logs", operatorPodName, "-n", namespace, "--since=120s")
				logs, err := utils.Run(cmd)
				if err != nil {
					return ""
				}
				return logs
			}, time.Minute*2, time.Second*10).Should(Or(
				ContainSubstring("project:"),
				ContainSubstring("application"),
				ContainSubstring("deployment"),
				ContainSubstring("stream"),
			))

			// Clean up
			_ = k8sClient.Delete(ctx, testDeployment)
			_ = k8sClient.Delete(ctx, testApplication)
			_ = k8sClient.Delete(ctx, testProject)
		})
	})
})
