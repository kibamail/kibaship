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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedisClient_XAdd(t *testing.T) {
	tests := []struct {
		name          string
		stream        string
		values        map[string]interface{}
		expectedError string
	}{
		{
			name:   "successful add",
			stream: "test-stream",
			values: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:          "empty stream name",
			stream:        "",
			values:        map[string]interface{}{"key": "value"},
			expectedError: "stream name cannot be empty",
		},
		{
			name:          "empty values",
			stream:        "test-stream",
			values:        map[string]interface{}{},
			expectedError: "values cannot be empty",
		},
		{
			name:          "nil values",
			stream:        "test-stream",
			values:        nil,
			expectedError: "values cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTestRedisClient("localhost:6379", "password")

			entryID, err := client.XAdd(context.Background(), tt.stream, tt.values)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, entryID)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, entryID)
				// Test implementation returns a fixed entry ID
				assert.Equal(t, "1703123456789-0", entryID)
			}
		})
	}
}

func TestRedisClient_Ping(t *testing.T) {
	tests := []struct {
		name          string
		address       string
		password      string
		expectedError string
	}{
		{
			name:     "successful ping",
			address:  "localhost:6379",
			password: "password",
		},
		{
			name:          "empty address",
			address:       "",
			password:      "password",
			expectedError: "Redis address is not configured",
		},
		{
			name:          "empty password",
			address:       "localhost:6379",
			password:      "",
			expectedError: "Redis password is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTestRedisClient(tt.address, tt.password)

			err := client.Ping(context.Background())

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisClient_Close(t *testing.T) {
	client := NewTestRedisClient("localhost:6379", "password")

	err := client.Close()
	assert.NoError(t, err)

	// After closing, should not be able to ping
	err = client.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis: client is closed")
}

func TestNewRedisClient(t *testing.T) {
	address := "localhost:6379"
	password := "test-password"

	client := NewTestRedisClient(address, password)
	assert.NotNil(t, client)

	// Test that the client is properly configured
	err := client.Ping(context.Background())
	assert.NoError(t, err)
}
