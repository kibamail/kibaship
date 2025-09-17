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
	"strings"

	"github.com/redis/go-redis/v9"
)

// valkeyClusterClient implements ValkeyClient interface using go-redis cluster client
type valkeyClusterClient struct {
	client *redis.ClusterClient
}

// NewValkeyClusterClient creates a new Valkey cluster client with auto-discovery
func NewValkeyClusterClient(seedAddress, password string, config *Config) (ValkeyClient, error) {
	// Parse seed address to get host and port
	var addresses []string
	if strings.Contains(seedAddress, ":") {
		addresses = []string{seedAddress}
	} else {
		addresses = []string{fmt.Sprintf("%s:%d", seedAddress, config.ValkeyPort)}
	}

	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:          addresses,
		Password:       password,
		DialTimeout:    config.ConnectionTimeout,
		ReadTimeout:    config.RequestTimeout,
		WriteTimeout:   config.RequestTimeout,
		PoolSize:       100,
		MinIdleConns:   10,
		MaxRetries:     config.RetryAttempts,
		RouteRandomly:  true,
		RouteByLatency: true,
		ReadOnly:       false,
		MaxRedirects:   8,
	})

	// Test cluster connectivity
	ctx := context.Background()
	if err := clusterClient.Ping(ctx).Err(); err != nil {
		clusterClient.Close()
		return nil, fmt.Errorf("failed to connect to Valkey cluster: %w", err)
	}

	return &valkeyClusterClient{
		client: clusterClient,
	}, nil
}

// XAdd adds an entry to a Valkey stream
func (v *valkeyClusterClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	if stream == "" {
		return "", fmt.Errorf("stream name cannot be empty")
	}
	if len(values) == 0 {
		return "", fmt.Errorf("values cannot be empty")
	}

	result := v.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	})

	return result.Result()
}

// Ping tests the connection to Valkey cluster
func (v *valkeyClusterClient) Ping(ctx context.Context) error {
	return v.client.Ping(ctx).Err()
}

// ClusterNodes returns cluster node information
func (v *valkeyClusterClient) ClusterNodes(ctx context.Context) (string, error) {
	result := v.client.ClusterNodes(ctx)
	return result.Result()
}

// Close closes the Valkey cluster client connection
func (v *valkeyClusterClient) Close() error {
	return v.client.Close()
}
