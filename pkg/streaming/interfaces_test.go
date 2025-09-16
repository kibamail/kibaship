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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestResourceEvent_Validation(t *testing.T) {
	tests := []struct {
		name    string
		event   *ResourceEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: &ResourceEvent{
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
			},
			wantErr: false,
		},
		{
			name: "missing event ID",
			event: &ResourceEvent{
				ProjectUUID:  uuid.New().String(),
				ResourceType: ResourceTypeProject,
				ResourceUUID: uuid.New().String(),
				Operation:    OperationCreate,
			},
			wantErr: true,
		},
		{
			name: "missing project UUID",
			event: &ResourceEvent{
				EventID:      uuid.New().String(),
				ResourceType: ResourceTypeProject,
				ResourceUUID: uuid.New().String(),
				Operation:    OperationCreate,
			},
			wantErr: true,
		},
		{
			name: "invalid resource type",
			event: &ResourceEvent{
				EventID:      uuid.New().String(),
				ProjectUUID:  uuid.New().String(),
				ResourceType: "invalid",
				ResourceUUID: uuid.New().String(),
				Operation:    OperationCreate,
			},
			wantErr: true,
		},
		{
			name: "invalid operation",
			event: &ResourceEvent{
				EventID:      uuid.New().String(),
				ProjectUUID:  uuid.New().String(),
				ResourceType: ResourceTypeProject,
				ResourceUUID: uuid.New().String(),
				Operation:    "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Namespace:         "default",
				ValkeyServiceName: "valkey-service",
				ValkeySecretName:  "valkey-secret",
				ValkeyPort:        6379,
				StartupTimeout:    5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing namespace",
			config: &Config{
				ValkeyServiceName: "valkey-service",
				ValkeySecretName:  "valkey-secret",
				ValkeyPort:        6379,
				StartupTimeout:    5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "missing service name",
			config: &Config{
				Namespace:        "default",
				ValkeySecretName: "valkey-secret",
				ValkeyPort:       6379,
				StartupTimeout:   5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &Config{
				Namespace:         "default",
				ValkeyServiceName: "valkey-service",
				ValkeySecretName:  "valkey-secret",
				ValkeyPort:        0,
				StartupTimeout:    5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: &Config{
				Namespace:         "default",
				ValkeyServiceName: "valkey-service",
				ValkeySecretName:  "valkey-secret",
				ValkeyPort:        6379,
				StartupTimeout:    0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewResourceEvent(t *testing.T) {
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

	assert.NotEmpty(t, event.EventID)
	assert.Equal(t, projectUUID, event.ProjectUUID)
	assert.Equal(t, workspaceUUID, event.WorkspaceUUID)
	assert.Equal(t, ResourceTypeApplication, event.ResourceType)
	assert.Equal(t, resourceUUID, event.ResourceUUID)
	assert.Equal(t, "test-app", event.ResourceSlug)
	assert.Equal(t, "default", event.Namespace)
	assert.Equal(t, OperationUpdate, event.Operation)
	assert.False(t, event.Timestamp.IsZero())
	assert.Equal(t, int64(0), event.Metadata.SequenceNumber) // Should be set by publisher

	// Validate the created event
	assert.NoError(t, event.Validate())
}

func TestNewResourceEventFromK8sResource(t *testing.T) {
	tests := []struct {
		name          string
		k8sResource   client.Object
		expectedError bool
	}{
		{
			name: "successful serialization with ConfigMap",
			k8sResource: &corev1.ConfigMap{
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
			},
			expectedError: false,
		},
		{
			name: "successful serialization with Secret",
			k8sResource: &corev1.Secret{
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
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				tt.k8sResource,
			)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, event)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, event)

				// Verify basic event fields
				assert.NotEmpty(t, event.EventID)
				assert.NotZero(t, event.Timestamp)
				assert.Equal(t, projectUUID, event.ProjectUUID)
				assert.Equal(t, workspaceUUID, event.WorkspaceUUID)
				assert.Equal(t, resourceType, event.ResourceType)
				assert.Equal(t, resourceUUID, event.ResourceUUID)
				assert.Equal(t, resourceSlug, event.ResourceSlug)
				assert.Equal(t, namespace, event.Namespace)
				assert.Equal(t, operation, event.Operation)

				// Verify that the Kubernetes resource is in the payload
				assert.NotNil(t, event.Payload)
				assert.Contains(t, event.Payload, "resource")

				resourceData, ok := event.Payload["resource"].(map[string]interface{})
				assert.True(t, ok, "resource should be a map")
				assert.NotEmpty(t, resourceData)

				// Verify specific fields based on resource type
				if configMap, ok := tt.k8sResource.(*corev1.ConfigMap); ok {
					metadata, exists := resourceData["metadata"].(map[string]interface{})
					assert.True(t, exists)
					assert.Equal(t, configMap.Name, metadata["name"])
					assert.Equal(t, configMap.Namespace, metadata["namespace"])

					data, exists := resourceData["data"].(map[string]interface{})
					assert.True(t, exists)
					assert.Equal(t, "value1", data["key1"])
					assert.Equal(t, "value2", data["key2"])
				}

				if secret, ok := tt.k8sResource.(*corev1.Secret); ok {
					metadata, exists := resourceData["metadata"].(map[string]interface{})
					assert.True(t, exists)
					assert.Equal(t, secret.Name, metadata["name"])
					assert.Equal(t, secret.Namespace, metadata["namespace"])

					data, exists := resourceData["data"].(map[string]interface{})
					assert.True(t, exists)
					// Note: Kubernetes secrets data is base64 encoded when marshaled to JSON
					assert.Contains(t, data, "password")
				}
			}
		})
	}
}
