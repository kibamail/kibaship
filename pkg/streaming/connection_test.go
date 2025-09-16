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
)

func TestConnectionManager_Connect(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		pingError     error
		expectedError string
	}{
		{
			name:     "successful connection",
			password: "test-password",
		},
		{
			name:          "ping fails",
			password:      "test-password",
			pingError:     errors.New("connection failed"),
			expectedError: "failed to ping Valkey cluster",
		},
		{
			name:     "empty password allowed",
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Namespace:         "test-namespace",
				ValkeyServiceName: "test-service",
				ValkeyPort:        6379,
			}

			mockClient := &mockRedisClient{}
			clientFactory := func(address, password string) RedisClient {
				expectedAddress := "test-service.test-namespace.svc.cluster.local:6379"
				assert.Equal(t, expectedAddress, address)
				assert.Equal(t, tt.password, password)
				return mockClient
			}

			manager := NewConnectionManagerWithFactory(config, clientFactory).(*connectionManager)

			mockClient.On("Ping", mock.Anything).Return(tt.pingError)
			if tt.pingError != nil {
				mockClient.On("Close").Return(nil)
			}

			err := manager.Connect(context.Background(), tt.password)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.False(t, manager.IsConnected())
			} else {
				assert.NoError(t, err)
				assert.True(t, manager.IsConnected())
				assert.NotNil(t, manager.GetClient())
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestConnectionManager_IsConnected(t *testing.T) {
	config := &Config{
		Namespace:         "test-namespace",
		ValkeyServiceName: "test-service",
		ValkeyPort:        6379,
	}

	manager := NewConnectionManager(config)

	// Initially not connected
	assert.False(t, manager.IsConnected())

	// Simulate connection
	mockClient := &mockRedisClient{}
	clientFactory := func(address, password string) RedisClient {
		return mockClient
	}

	manager = NewConnectionManagerWithFactory(config, clientFactory)

	mockClient.On("Ping", mock.Anything).Return(nil)

	err := manager.Connect(context.Background(), "password")
	assert.NoError(t, err)
	assert.True(t, manager.IsConnected())

	mockClient.AssertExpectations(t)
}

func TestConnectionManager_GetClient(t *testing.T) {
	config := &Config{
		Namespace:         "test-namespace",
		ValkeyServiceName: "test-service",
		ValkeyPort:        6379,
	}

	manager := NewConnectionManager(config)

	// Initially no client
	assert.Nil(t, manager.GetClient())

	// After connection, client should be available
	mockClient := &mockRedisClient{}
	clientFactory := func(address, password string) RedisClient {
		return mockClient
	}

	manager = NewConnectionManagerWithFactory(config, clientFactory)

	mockClient.On("Ping", mock.Anything).Return(nil)

	err := manager.Connect(context.Background(), "password")
	assert.NoError(t, err)
	assert.Equal(t, mockClient, manager.GetClient())

	mockClient.AssertExpectations(t)
}

func TestConnectionManager_Close(t *testing.T) {
	config := &Config{
		Namespace:         "test-namespace",
		ValkeyServiceName: "test-service",
		ValkeyPort:        6379,
	}

	manager := NewConnectionManager(config)

	// Close when not connected should not error
	err := manager.Close()
	assert.NoError(t, err)

	// Connect first
	mockClient := &mockRedisClient{}
	clientFactory := func(address, password string) RedisClient {
		return mockClient
	}

	manager = NewConnectionManagerWithFactory(config, clientFactory)

	mockClient.On("Ping", mock.Anything).Return(nil)
	mockClient.On("Close").Return(nil)

	err = manager.Connect(context.Background(), "password")
	assert.NoError(t, err)

	// Now close
	err = manager.Close()
	assert.NoError(t, err)
	assert.False(t, manager.IsConnected())
	assert.Nil(t, manager.GetClient())

	mockClient.AssertExpectations(t)
}

func TestConnectionManager_AlreadyConnected(t *testing.T) {
	config := &Config{
		Namespace:         "test-namespace",
		ValkeyServiceName: "test-service",
		ValkeyPort:        6379,
	}

	mockClient := &mockRedisClient{}
	callCount := 0
	clientFactory := func(address, password string) RedisClient {
		callCount++
		return mockClient
	}

	manager := NewConnectionManagerWithFactory(config, clientFactory)

	mockClient.On("Ping", mock.Anything).Return(nil)

	// Connect first time
	err := manager.Connect(context.Background(), "password")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Connect second time - should not create new client
	err = manager.Connect(context.Background(), "password")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount) // Should not increment

	mockClient.AssertExpectations(t)
}
