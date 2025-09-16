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
	"time"

	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// mockKubernetesClient implements KubernetesClient for testing
type mockKubernetesClient struct {
	mock.Mock
}

func (m *mockKubernetesClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

func (m *mockKubernetesClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *mockKubernetesClient) Watch(ctx context.Context, list client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	args := m.Called(ctx, list, opts)
	return args.Get(0).(watch.Interface), args.Error(1)
}

// mockTimeProvider implements TimeProvider for testing
type mockTimeProvider struct {
	mock.Mock
}

func (m *mockTimeProvider) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *mockTimeProvider) Sleep(duration time.Duration) {
	m.Called(duration)
}

func (m *mockTimeProvider) After(duration time.Duration) <-chan time.Time {
	args := m.Called(duration)
	return args.Get(0).(<-chan time.Time)
}

// mockRedisClient implements RedisClient for testing
type mockRedisClient struct {
	mock.Mock
}

func (m *mockRedisClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	args := m.Called(ctx, stream, values)
	return args.String(0), args.Error(1)
}

func (m *mockRedisClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// mockConnectionManager implements ConnectionManager for testing
type mockConnectionManager struct {
	mock.Mock
}

func (m *mockConnectionManager) Connect(ctx context.Context, password string) error {
	args := m.Called(ctx, password)
	return args.Error(0)
}

func (m *mockConnectionManager) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockConnectionManager) GetClient() RedisClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(RedisClient)
}

func (m *mockConnectionManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

// mockValkeyReadinessMonitor implements ValkeyReadinessMonitor for testing
type mockValkeyReadinessMonitor struct {
	mock.Mock
}

func (m *mockValkeyReadinessMonitor) WaitForReady(ctx context.Context, timeout time.Duration) error {
	args := m.Called(ctx, timeout)
	return args.Error(0)
}

func (m *mockValkeyReadinessMonitor) IsReady(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

// mockSecretManager implements SecretManager for testing
type mockSecretManager struct {
	mock.Mock
}

func (m *mockSecretManager) GetValkeyPassword(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

// MockValkeyCluster represents a Valkey cluster resource for testing
type MockValkeyCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            MockValkeyClusterStatus `json:"status,omitempty"`
}

func (m *MockValkeyCluster) DeepCopyObject() runtime.Object {
	return m
}

type MockValkeyClusterStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
