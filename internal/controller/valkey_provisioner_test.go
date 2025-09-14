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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ValkeyProvisioner", func() {
	var (
		provisioner *ValkeyProvisioner
		testContext context.Context
	)

	BeforeEach(func() {
		provisioner = NewValkeyProvisioner(k8sClient)
		testContext = context.Background()
	})

	Describe("NewValkeyProvisioner", func() {
		It("should create a new ValkeyProvisioner instance", func() {
			p := NewValkeyProvisioner(k8sClient)
			Expect(p).NotTo(BeNil())
			Expect(p.Client).To(Equal(k8sClient))
		})
	})

	Describe("generateSystemValkeyClusterName", func() {
		It("should generate correct cluster name following conventions", func() {
			name := generateSystemValkeyClusterName()
			Expect(name).To(Equal("kibaship-valkey-cluster-kibaship-com"))
		})
	})

	Describe("generateSystemValkeyCluster", func() {
		It("should create Valkey cluster with correct configuration", func() {
			name := "test-valkey-cluster"
			namespace := "test-namespace"

			valkey := generateSystemValkeyCluster(name, namespace)

			Expect(valkey).NotTo(BeNil())
			Expect(valkey.GetAPIVersion()).To(Equal("hyperspike.io/v1"))
			Expect(valkey.GetKind()).To(Equal("Valkey"))
			Expect(valkey.GetName()).To(Equal(name))
			Expect(valkey.GetNamespace()).To(Equal(namespace))

			// Check labels
			labels := valkey.GetLabels()
			Expect(labels["app.kubernetes.io/name"]).To(Equal("kibaship-valkey-cluster"))
			Expect(labels["app.kubernetes.io/managed-by"]).To(Equal("kibaship"))
			Expect(labels["app.kubernetes.io/component"]).To(Equal("system-cache"))
			Expect(labels["app.kubernetes.io/part-of"]).To(Equal("kibaship-platform"))

			// Check annotations
			annotations := valkey.GetAnnotations()
			Expect(annotations["description"]).To(Equal("System Valkey cluster for KibaShip platform caching and session management"))
			Expect(annotations["platform.kibaship.com/role"]).To(Equal("system-cache"))

			// Check spec fields directly from the unstructured object
			nodes, found, err := unstructured.NestedInt64(valkey.Object, "spec", "nodes")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(nodes).To(Equal(int64(3)))

			replicas, found, err := unstructured.NestedInt64(valkey.Object, "spec", "replicas")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(replicas).To(Equal(int64(2)))

			tls, found, err := unstructured.NestedBool(valkey.Object, "spec", "tls")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(tls).To(BeFalse())

			prometheus, found, err := unstructured.NestedBool(valkey.Object, "spec", "prometheus")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(prometheus).To(BeTrue())

			volumePermissions, found, err := unstructured.NestedBool(valkey.Object, "spec", "volumePermissions")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(volumePermissions).To(BeTrue())
		})

		It("should set correct resource limits and requests", func() {
			name := "test-valkey-cluster"
			namespace := "test-namespace"

			valkey := generateSystemValkeyCluster(name, namespace)

			// Check resource limits
			cpuLimit, found, err := unstructured.NestedString(valkey.Object, "spec", "resources", "limits", "cpu")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(cpuLimit).To(Equal("100m"))

			memoryLimit, found, err := unstructured.NestedString(valkey.Object, "spec", "resources", "limits", "memory")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(memoryLimit).To(Equal("128Mi"))

			// Check resource requests
			cpuRequest, found, err := unstructured.NestedString(valkey.Object, "spec", "resources", "requests", "cpu")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(cpuRequest).To(Equal("100m"))

			memoryRequest, found, err := unstructured.NestedString(valkey.Object, "spec", "resources", "requests", "memory")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(memoryRequest).To(Equal("128Mi"))
		})

		It("should set correct storage configuration", func() {
			name := "test-valkey-cluster"
			namespace := "test-namespace"

			valkey := generateSystemValkeyCluster(name, namespace)

			// Check storage access modes
			accessModes, found, err := unstructured.NestedStringSlice(valkey.Object, "spec", "storage", "spec", "accessModes")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(accessModes).To(Equal([]string{"ReadWriteOnce"}))

			// Check storage class name
			storageClassName, found, err := unstructured.NestedString(valkey.Object, "spec", "storage", "spec", "storageClassName")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(storageClassName).To(Equal("storage-replica-1"))

			// Check storage size
			storageSize, found, err := unstructured.NestedString(valkey.Object, "spec", "storage", "spec", "resources", "requests", "storage")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(storageSize).To(Equal("1Gi"))
		})
	})

	Describe("getOperatorNamespace", func() {
		BeforeEach(func() {
			// Clear environment variables before each test
			os.Unsetenv("OPERATOR_NAMESPACE")
			os.Unsetenv("POD_NAMESPACE")
		})

		AfterEach(func() {
			// Clean up environment variables after each test
			os.Unsetenv("OPERATOR_NAMESPACE")
			os.Unsetenv("POD_NAMESPACE")
		})

		It("should return OPERATOR_NAMESPACE when set", func() {
			os.Setenv("OPERATOR_NAMESPACE", "test-operator-namespace")

			namespace, err := getOperatorNamespace()
			Expect(err).NotTo(HaveOccurred())
			Expect(namespace).To(Equal("test-operator-namespace"))
		})

		It("should return POD_NAMESPACE when OPERATOR_NAMESPACE not set", func() {
			os.Setenv("POD_NAMESPACE", "test-pod-namespace")

			namespace, err := getOperatorNamespace()
			Expect(err).NotTo(HaveOccurred())
			Expect(namespace).To(Equal("test-pod-namespace"))
		})

		It("should return default namespace when no environment variables set", func() {
			namespace, err := getOperatorNamespace()
			Expect(err).NotTo(HaveOccurred())
			Expect(namespace).To(Equal("kibaship-system"))
		})

		It("should prioritize OPERATOR_NAMESPACE over POD_NAMESPACE", func() {
			os.Setenv("OPERATOR_NAMESPACE", "operator-ns")
			os.Setenv("POD_NAMESPACE", "pod-ns")

			namespace, err := getOperatorNamespace()
			Expect(err).NotTo(HaveOccurred())
			Expect(namespace).To(Equal("operator-ns"))
		})
	})

	Describe("checkValkeyClusterExists", func() {
		It("should return false when cluster does not exist", func() {
			exists, err := provisioner.checkValkeyClusterExists(testContext, "non-existent-cluster", "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should return true when cluster exists", func() {
			// Create a test Valkey cluster
			testCluster := &unstructured.Unstructured{}
			testCluster.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "hyperspike.io",
				Version: "v1",
				Kind:    "Valkey",
			})
			testCluster.SetName("test-existing-cluster")
			testCluster.SetNamespace("default")

			err := k8sClient.Create(testContext, testCluster)
			Expect(err).NotTo(HaveOccurred())

			exists, err := provisioner.checkValkeyClusterExists(testContext, "test-existing-cluster", "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Clean up
			err = k8sClient.Delete(testContext, testCluster)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("createValkeyCluster", func() {
		It("should successfully create a Valkey cluster", func() {
			clusterName := "test-create-cluster"
			namespace := "default"

			err := provisioner.createValkeyCluster(testContext, clusterName, namespace)
			Expect(err).NotTo(HaveOccurred())

			// Verify the cluster was created
			valkey := &unstructured.Unstructured{}
			valkey.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "hyperspike.io",
				Version: "v1",
				Kind:    "Valkey",
			})

			err = k8sClient.Get(testContext, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, valkey)
			Expect(err).NotTo(HaveOccurred())
			Expect(valkey.GetName()).To(Equal(clusterName))
			Expect(valkey.GetNamespace()).To(Equal(namespace))

			// Clean up
			err = k8sClient.Delete(testContext, valkey)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ProvisionSystemValkeyCluster", func() {
		var testNamespace string

		BeforeEach(func() {
			testNamespace = "default"
			os.Setenv("OPERATOR_NAMESPACE", testNamespace)
		})

		AfterEach(func() {
			os.Unsetenv("OPERATOR_NAMESPACE")

			// Clean up any created Valkey clusters
			clusterName := generateSystemValkeyClusterName()
			valkey := &unstructured.Unstructured{}
			valkey.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "hyperspike.io",
				Version: "v1",
				Kind:    "Valkey",
			})

			err := k8sClient.Get(testContext, types.NamespacedName{
				Name:      clusterName,
				Namespace: testNamespace,
			}, valkey)
			if err == nil {
				k8sClient.Delete(testContext, valkey)
			}
		})

		It("should successfully provision a new Valkey cluster when none exists", func() {
			err := provisioner.ProvisionSystemValkeyCluster(testContext)
			Expect(err).NotTo(HaveOccurred())

			// Verify the cluster was created
			clusterName := generateSystemValkeyClusterName()
			valkey := &unstructured.Unstructured{}
			valkey.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "hyperspike.io",
				Version: "v1",
				Kind:    "Valkey",
			})

			err = k8sClient.Get(testContext, types.NamespacedName{
				Name:      clusterName,
				Namespace: testNamespace,
			}, valkey)
			Expect(err).NotTo(HaveOccurred())
			Expect(valkey.GetName()).To(Equal(clusterName))
			Expect(valkey.GetNamespace()).To(Equal(testNamespace))
		})

		It("should skip provisioning when cluster already exists", func() {
			// First provision
			err := provisioner.ProvisionSystemValkeyCluster(testContext)
			Expect(err).NotTo(HaveOccurred())

			// Second provision should not error and should skip
			err = provisioner.ProvisionSystemValkeyCluster(testContext)
			Expect(err).NotTo(HaveOccurred())

			// Verify only one cluster exists
			clusterName := generateSystemValkeyClusterName()
			valkey := &unstructured.Unstructured{}
			valkey.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "hyperspike.io",
				Version: "v1",
				Kind:    "Valkey",
			})

			err = k8sClient.Get(testContext, types.NamespacedName{
				Name:      clusterName,
				Namespace: testNamespace,
			}, valkey)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle namespace determination errors gracefully", func() {
			// Clear all namespace environment variables to force fallback to default
			originalOpNs := os.Getenv("OPERATOR_NAMESPACE")
			originalPodNs := os.Getenv("POD_NAMESPACE")

			os.Unsetenv("OPERATOR_NAMESPACE")
			os.Unsetenv("POD_NAMESPACE")

			// This should fall back to "kibaship-system" but that namespace doesn't exist in test env
			// So we expect this to fail with "namespace not found"
			err := provisioner.ProvisionSystemValkeyCluster(testContext)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))

			// Restore original env vars
			if originalOpNs != "" {
				os.Setenv("OPERATOR_NAMESPACE", originalOpNs)
			}
			if originalPodNs != "" {
				os.Setenv("POD_NAMESPACE", originalPodNs)
			}
		})
	})
})
