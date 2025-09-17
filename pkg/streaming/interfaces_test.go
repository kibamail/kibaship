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

package streaming

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ResourceEvent", func() {
	Describe("Validation", func() {
		Context("with valid event", func() {
			It("should validate successfully", func() {
				event := &ResourceEvent{
					EventID:       uuid.New().String(),
					ProjectUUID:   uuid.New().String(),
					WorkspaceUUID: uuid.New().String(),
					ResourceType:  ResourceTypeProject,
					ResourceUUID:  uuid.New().String(),
					Operation:     OperationCreate,
					Timestamp:     time.Now(),
					Metadata: EventMetadata{
						SequenceNumber: 1,
					},
				}

				err := event.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with missing event ID", func() {
			It("should return validation error", func() {
				event := &ResourceEvent{
					ProjectUUID:  uuid.New().String(),
					ResourceType: ResourceTypeProject,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
				}

				err := event.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with missing project UUID", func() {
			It("should return validation error", func() {
				event := &ResourceEvent{
					EventID:      uuid.New().String(),
					ResourceType: ResourceTypeProject,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
				}

				err := event.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with invalid resource type", func() {
			It("should return validation error", func() {
				event := &ResourceEvent{
					EventID:      uuid.New().String(),
					ProjectUUID:  uuid.New().String(),
					ResourceType: "invalid",
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
				}

				err := event.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with invalid operation", func() {
			It("should return validation error", func() {
				event := &ResourceEvent{
					EventID:      uuid.New().String(),
					ProjectUUID:  uuid.New().String(),
					ResourceType: ResourceTypeProject,
					ResourceUUID: uuid.New().String(),
					Operation:    "invalid",
				}

				err := event.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

var _ = Describe("Config", func() {
	Describe("Validation", func() {
		Context("with valid config", func() {
			It("should validate successfully", func() {
				config := &Config{
					Namespace:         "default",
					ValkeyServiceName: "valkey-service",
					ValkeySecretName:  "valkey-secret",
					ValkeyPort:        6379,
					StartupTimeout:    5 * time.Minute,
				}

				err := config.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with missing namespace", func() {
			It("should return validation error", func() {
				config := &Config{
					ValkeyServiceName: "valkey-service",
					ValkeySecretName:  "valkey-secret",
					ValkeyPort:        6379,
					StartupTimeout:    5 * time.Minute,
				}

				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with missing service name", func() {
			It("should return validation error", func() {
				config := &Config{
					Namespace:        "default",
					ValkeySecretName: "valkey-secret",
					ValkeyPort:       6379,
					StartupTimeout:   5 * time.Minute,
				}

				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with invalid port", func() {
			It("should return validation error", func() {
				config := &Config{
					Namespace:         "default",
					ValkeyServiceName: "valkey-service",
					ValkeySecretName:  "valkey-secret",
					ValkeyPort:        0,
					StartupTimeout:    5 * time.Minute,
				}

				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with zero timeout", func() {
			It("should return validation error", func() {
				config := &Config{
					Namespace:         "default",
					ValkeyServiceName: "valkey-service",
					ValkeySecretName:  "valkey-secret",
					ValkeyPort:        6379,
					StartupTimeout:    0,
				}

				err := config.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

var _ = Describe("NewResourceEvent", func() {
	It("should create a valid resource event", func() {
		projectUUID := uuid.New().String()
		workspaceUUID := uuid.New().String()
		resourceUUID := uuid.New().String()

		event := NewResourceEvent(
			projectUUID,
			workspaceUUID,
			ResourceTypeApplication,
			resourceUUID,
			"test-app",
			"default",
			OperationUpdate,
		)

		Expect(event.EventID).NotTo(BeEmpty())
		Expect(event.ProjectUUID).To(Equal(projectUUID))
		Expect(event.WorkspaceUUID).To(Equal(workspaceUUID))
		Expect(event.ResourceType).To(Equal(ResourceTypeApplication))
		Expect(event.ResourceUUID).To(Equal(resourceUUID))
		Expect(event.ResourceSlug).To(Equal("test-app"))
		Expect(event.Namespace).To(Equal("default"))
		Expect(event.Operation).To(Equal(OperationUpdate))
		Expect(event.Timestamp.IsZero()).To(BeFalse())
		Expect(event.Metadata.SequenceNumber).To(Equal(int64(0))) // Should be set by publisher

		// Validate the created event
		err := event.Validate()
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("NewResourceEventFromK8sResource", func() {
	Context("with ConfigMap resource", func() {
		It("should create event with serialized ConfigMap", func() {
			k8sResource := &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			}

			projectUUID := "proj-123"
			workspaceUUID := "workspace-456"
			resourceType := ResourceTypeProject
			resourceUUID := "res-789"
			resourceSlug := "test-resource"
			namespace := "test-namespace"
			operation := OperationCreate

			event, err := NewResourceEventFromK8sResource(
				projectUUID,
				workspaceUUID,
				resourceType,
				resourceUUID,
				resourceSlug,
				namespace,
				operation,
				k8sResource,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(event).NotTo(BeNil())

			// Verify basic event fields
			Expect(event.EventID).NotTo(BeEmpty())
			Expect(event.Timestamp.IsZero()).To(BeFalse())
			Expect(event.ProjectUUID).To(Equal(projectUUID))
			Expect(event.WorkspaceUUID).To(Equal(workspaceUUID))
			Expect(event.ResourceType).To(Equal(resourceType))
			Expect(event.ResourceUUID).To(Equal(resourceUUID))
			Expect(event.ResourceSlug).To(Equal(resourceSlug))
			Expect(event.Namespace).To(Equal(namespace))
			Expect(event.Operation).To(Equal(operation))

			// Verify that the Kubernetes resource is in the payload
			Expect(event.Payload).NotTo(BeNil())
			Expect(event.Payload).To(HaveKey("resource"))

			resourceData, ok := event.Payload["resource"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "resource should be a map")
			Expect(resourceData).NotTo(BeEmpty())

			// Verify specific ConfigMap fields
			metadata, exists := resourceData["metadata"].(map[string]interface{})
			Expect(exists).To(BeTrue())
			Expect(metadata["name"]).To(Equal(k8sResource.Name))
			Expect(metadata["namespace"]).To(Equal(k8sResource.Namespace))

			data, exists := resourceData["data"].(map[string]interface{})
			Expect(exists).To(BeTrue())
			Expect(data["key1"]).To(Equal("value1"))
			Expect(data["key2"]).To(Equal("value2"))
		})
	})

	Context("with Secret resource", func() {
		It("should create event with serialized Secret", func() {
			k8sResource := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			}

			projectUUID := "proj-123"
			workspaceUUID := "workspace-456"
			resourceType := ResourceTypeProject
			resourceUUID := "res-789"
			resourceSlug := "test-resource"
			namespace := "test-namespace"
			operation := OperationCreate

			event, err := NewResourceEventFromK8sResource(
				projectUUID,
				workspaceUUID,
				resourceType,
				resourceUUID,
				resourceSlug,
				namespace,
				operation,
				k8sResource,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(event).NotTo(BeNil())

			// Verify basic event fields
			Expect(event.EventID).NotTo(BeEmpty())
			Expect(event.Timestamp.IsZero()).To(BeFalse())
			Expect(event.ProjectUUID).To(Equal(projectUUID))
			Expect(event.WorkspaceUUID).To(Equal(workspaceUUID))
			Expect(event.ResourceType).To(Equal(resourceType))
			Expect(event.ResourceUUID).To(Equal(resourceUUID))
			Expect(event.ResourceSlug).To(Equal(resourceSlug))
			Expect(event.Namespace).To(Equal(namespace))
			Expect(event.Operation).To(Equal(operation))

			// Verify that the Kubernetes resource is in the payload
			Expect(event.Payload).NotTo(BeNil())
			Expect(event.Payload).To(HaveKey("resource"))

			resourceData, ok := event.Payload["resource"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "resource should be a map")
			Expect(resourceData).NotTo(BeEmpty())

			// Verify specific Secret fields
			metadata, exists := resourceData["metadata"].(map[string]interface{})
			Expect(exists).To(BeTrue())
			Expect(metadata["name"]).To(Equal(k8sResource.Name))
			Expect(metadata["namespace"]).To(Equal(k8sResource.Namespace))

			data, exists := resourceData["data"].(map[string]interface{})
			Expect(exists).To(BeTrue())
			// Note: Kubernetes secrets data is base64 encoded when marshaled to JSON
			Expect(data).To(HaveKey("password"))
		})
	})
})
