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

// testValkeyClient implements ValkeyClient interface for testing without real Valkey
type testValkeyClient struct {
	address  string
	password string
	closed   bool
}

// NewTestValkeyClient creates a test Valkey client that doesn't require a real Valkey instance
func NewTestValkeyClient(seedAddress, password string, config *Config) (ValkeyClient, error) {
	return &testValkeyClient{
		address:  seedAddress,
		password: password,
	}, nil
}

// XAdd adds an entry to a Valkey stream (test implementation)
func (r *testValkeyClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	if r.closed {
		return "", fmt.Errorf("valkey: client is closed")
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
func (r *testValkeyClient) Ping(ctx context.Context) error {
	if r.closed {
		return fmt.Errorf("valkey: client is closed")
	}
	if r.address == "" {
		return fmt.Errorf("Valkey address is not configured")
	}

	// Mock successful ping
	return nil
}

// ClusterNodes returns cluster node information (test implementation)
func (r *testValkeyClient) ClusterNodes(ctx context.Context) (string, error) {
	if r.closed {
		return "", fmt.Errorf("valkey: client is closed")
	}

	// Mock cluster nodes response
	return "node1:6379@16379 master - 0 1703123456789 0 connected 0-5460\nnode2:6379@16379 master - 0 1703123456789 0 connected 5461-10922\nnode3:6379@16379 master - 0 1703123456789 0 connected 10923-16383\n", nil
}

// Close closes the client (test implementation)
func (r *testValkeyClient) Close() error {
	r.closed = true
	return nil
}
