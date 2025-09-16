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
	"fmt"
)

// testRedisClient implements RedisClient interface for testing without real Redis
type testRedisClient struct {
	address  string
	password string
	closed   bool
}

// NewTestRedisClient creates a test Redis client that doesn't require a real Redis instance
func NewTestRedisClient(address, password string) RedisClient {
	return &testRedisClient{
		address:  address,
		password: password,
	}
}

// XAdd adds an entry to a Redis stream (test implementation)
func (r *testRedisClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	if r.closed {
		return "", fmt.Errorf("redis: client is closed")
	}
	if stream == "" {
		return "", fmt.Errorf("stream name cannot be empty")
	}
	if len(values) == 0 {
		return "", fmt.Errorf("values cannot be empty")
	}

	// Mock response - in real implementation, this would be the stream entry ID
	return "1703123456789-0", nil
}

// Ping tests the connection (test implementation)
func (r *testRedisClient) Ping(ctx context.Context) error {
	if r.closed {
		return fmt.Errorf("redis: client is closed")
	}
	if r.address == "" {
		return fmt.Errorf("Redis address is not configured")
	}
	if r.password == "" {
		return fmt.Errorf("Redis password is not configured")
	}

	// Mock successful ping
	return nil
}

// Close closes the client (test implementation)
func (r *testRedisClient) Close() error {
	r.closed = true
	return nil
}
