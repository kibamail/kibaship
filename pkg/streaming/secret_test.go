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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
)

func TestSecretManager_GetValkeyPassword(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*mockKubernetesClient)
		expectedPass  string
		expectedError string
	}{
		{
			name: "successful password retrieval",
			setupMocks: func(k8sClient *mockKubernetesClient) {
				secret := &corev1.Secret{
					Data: map[string][]byte{
						"password": []byte("test-password"),
					},
				}
				k8sClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					obj := args.Get(2).(*corev1.Secret)
					*obj = *secret
				})
			},
			expectedPass: "test-password",
		},
		{
			name: "secret not found",
			setupMocks: func(k8sClient *mockKubernetesClient) {
				k8sClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("secret not found"))
			},
			expectedError: "failed to get Valkey secret",
		},
		{
			name: "multiple fields in secret",
			setupMocks: func(k8sClient *mockKubernetesClient) {
				secret := &corev1.Secret{
					Data: map[string][]byte{
						"field1": []byte("value1"),
						"field2": []byte("value2"),
					},
				}
				k8sClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					obj := args.Get(2).(*corev1.Secret)
					*obj = *secret
				})
			},
			expectedError: "expected exactly one field in secret, got 2 fields",
		},
		{
			name: "empty password",
			setupMocks: func(k8sClient *mockKubernetesClient) {
				secret := &corev1.Secret{
					Data: map[string][]byte{
						"password": []byte(""),
					},
				}
				k8sClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					obj := args.Get(2).(*corev1.Secret)
					*obj = *secret
				})
			},
			expectedError: "secret contains empty password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := &mockKubernetesClient{}
			config := &Config{
				Namespace:        "test-namespace",
				ValkeySecretName: "test-secret",
			}

			tt.setupMocks(k8sClient)

			manager := NewSecretManager(k8sClient, config)

			password, err := manager.GetValkeyPassword(context.Background())

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, password)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPass, password)
			}

			k8sClient.AssertExpectations(t)
		})
	}
}
