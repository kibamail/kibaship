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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStartupSequenceController_Initialize(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*mockValkeyReadinessMonitor, *mockSecretManager, *mockConnectionManager)
		expectedError string
	}{
		{
			name: "successful initialization",
			setupMocks: func(readiness *mockValkeyReadinessMonitor, secret *mockSecretManager, conn *mockConnectionManager) {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
				secret.On("GetValkeyPassword", mock.Anything).Return("test-password", nil)
				conn.On("Connect", mock.Anything, "test-password").Return(nil)
				conn.On("IsConnected").Return(true)
			},
		},
		{
			name: "valkey readiness fails",
			setupMocks: func(readiness *mockValkeyReadinessMonitor, secret *mockSecretManager, conn *mockConnectionManager) {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(errors.New("timeout"))
			},
			expectedError: "Valkey cluster failed to become ready",
		},
		{
			name: "secret retrieval fails",
			setupMocks: func(readiness *mockValkeyReadinessMonitor, secret *mockSecretManager, conn *mockConnectionManager) {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
				secret.On("GetValkeyPassword", mock.Anything).Return("", errors.New("secret not found"))
			},
			expectedError: "Valkey authentication secret not found",
		},
		{
			name: "connection fails",
			setupMocks: func(readiness *mockValkeyReadinessMonitor, secret *mockSecretManager, conn *mockConnectionManager) {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
				secret.On("GetValkeyPassword", mock.Anything).Return("test-password", nil)
				conn.On("Connect", mock.Anything, "test-password").Return(errors.New("connection failed"))
			},
			expectedError: "failed to establish connection to Valkey cluster",
		},
		{
			name: "empty password retrieved from secret",
			setupMocks: func(readiness *mockValkeyReadinessMonitor, secret *mockSecretManager, conn *mockConnectionManager) {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
				secret.On("GetValkeyPassword", mock.Anything).Return("", nil)
			},
			expectedError: "Valkey authentication secret empty after cluster ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readiness := &mockValkeyReadinessMonitor{}
			secret := &mockSecretManager{}
			conn := &mockConnectionManager{}
			config := &Config{
				StartupTimeout: 5 * time.Minute,
			}

			tt.setupMocks(readiness, secret, conn)

			controller := NewStartupSequenceController(readiness, secret, conn, config)

			err := controller.Initialize(context.Background())

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.True(t, controller.IsReady())
			}

			readiness.AssertExpectations(t)
			secret.AssertExpectations(t)
			conn.AssertExpectations(t)
		})
	}
}

func TestStartupSequenceController_IsReady(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*mockConnectionManager)
		initialized bool
		expected    bool
	}{
		{
			name: "ready when initialized and connected",
			setupMocks: func(conn *mockConnectionManager) {
				conn.On("IsConnected").Return(true)
			},
			initialized: true,
			expected:    true,
		},
		{
			name: "not ready when not initialized",
			setupMocks: func(conn *mockConnectionManager) {
				// No expectations - IsConnected should not be called when not initialized
			},
			initialized: false,
			expected:    false,
		},
		{
			name: "not ready when not connected",
			setupMocks: func(conn *mockConnectionManager) {
				conn.On("IsConnected").Return(false)
			},
			initialized: true,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readiness := &mockValkeyReadinessMonitor{}
			secret := &mockSecretManager{}
			conn := &mockConnectionManager{}
			config := &Config{}

			tt.setupMocks(conn)

			controller := NewStartupSequenceController(readiness, secret, conn, config).(*startupSequenceController)

			// Set the ready state directly for testing
			controller.mutex.Lock()
			controller.ready = tt.initialized
			controller.mutex.Unlock()

			result := controller.IsReady()
			assert.Equal(t, tt.expected, result)

			conn.AssertExpectations(t)
		})
	}
}

func TestStartupSequenceController_Shutdown(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*mockConnectionManager)
		expectedError string
	}{
		{
			name: "successful shutdown",
			setupMocks: func(conn *mockConnectionManager) {
				conn.On("Close").Return(nil)
			},
		},
		{
			name: "shutdown with connection error",
			setupMocks: func(conn *mockConnectionManager) {
				conn.On("Close").Return(errors.New("close failed"))
			},
			expectedError: "close failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readiness := &mockValkeyReadinessMonitor{}
			secret := &mockSecretManager{}
			conn := &mockConnectionManager{}
			config := &Config{}

			tt.setupMocks(conn)

			controller := NewStartupSequenceController(readiness, secret, conn, config).(*startupSequenceController)

			// Initialize as ready
			controller.mutex.Lock()
			controller.ready = true
			controller.mutex.Unlock()

			err := controller.Shutdown(context.Background())

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Should be marked as not ready after shutdown
			assert.False(t, controller.IsReady())

			conn.AssertExpectations(t)
		})
	}
}

func TestStartupSequenceController_Integration(t *testing.T) {
	// Full integration test with successful flow
	readiness := &mockValkeyReadinessMonitor{}
	secret := &mockSecretManager{}
	conn := &mockConnectionManager{}
	config := &Config{
		StartupTimeout: 5 * time.Minute,
	}

	// Setup successful initialization flow
	readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
	secret.On("GetValkeyPassword", mock.Anything).Return("test-password", nil)
	conn.On("Connect", mock.Anything, "test-password").Return(nil)
	conn.On("IsConnected").Return(true)
	conn.On("Close").Return(nil)

	controller := NewStartupSequenceController(readiness, secret, conn, config)

	// Initially not ready
	assert.False(t, controller.IsReady())

	// Initialize
	err := controller.Initialize(context.Background())
	assert.NoError(t, err)

	// Should be ready after initialization
	assert.True(t, controller.IsReady())

	// Shutdown
	err = controller.Shutdown(context.Background())
	assert.NoError(t, err)

	// Should not be ready after shutdown
	assert.False(t, controller.IsReady())

	readiness.AssertExpectations(t)
	secret.AssertExpectations(t)
	conn.AssertExpectations(t)
}

func TestStartupSequenceController_SimplifiedFlow(t *testing.T) {
	// Test the simplified startup flow without secret watching
	readiness := &mockValkeyReadinessMonitor{}
	secret := &mockSecretManager{}
	conn := &mockConnectionManager{}
	config := &Config{
		StartupTimeout: 5 * time.Minute,
	}

	// Setup mocks for the complete simplified flow
	readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
	secret.On("GetValkeyPassword", mock.Anything).Return("test-password", nil)
	conn.On("Connect", mock.Anything, "test-password").Return(nil)
	conn.On("IsConnected").Return(true)

	controller := NewStartupSequenceController(readiness, secret, conn, config)

	// Initialize should succeed without secret watching
	err := controller.Initialize(context.Background())
	assert.NoError(t, err)

	// Should be ready
	assert.True(t, controller.IsReady())

	// Verify that no secret watching was attempted (all expectations met)
	readiness.AssertExpectations(t)
	secret.AssertExpectations(t) // Only GetValkeyPassword should have been called
	conn.AssertExpectations(t)
}
