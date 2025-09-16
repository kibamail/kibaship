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
	"sync"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// RedisClientFactory creates Redis clients
type RedisClientFactory func(address, password string) RedisClient

// connectionManager implements ConnectionManager
type connectionManager struct {
	config        *Config
	client        RedisClient
	connected     bool
	mutex         sync.RWMutex
	clientFactory RedisClientFactory
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(config *Config) ConnectionManager {
	return &connectionManager{
		config:        config,
		clientFactory: NewRedisClient,
	}
}

// NewConnectionManagerWithFactory creates a new connection manager with custom client factory
func NewConnectionManagerWithFactory(config *Config, clientFactory RedisClientFactory) ConnectionManager {
	return &connectionManager{
		config:        config,
		clientFactory: clientFactory,
	}
}

// Connect establishes connection to Valkey cluster
func (c *connectionManager) Connect(ctx context.Context, password string) error {
	log := logf.FromContext(ctx).WithName("connection-manager")

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected && c.client != nil {
		log.Info("Already connected to Valkey")
		return nil
	}

	// Build connection address
	address := fmt.Sprintf("%s.%s.svc.cluster.local:%d",
		c.config.ValkeyServiceName,
		c.config.Namespace,
		c.config.ValkeyPort)

	log.Info("Connecting to Valkey cluster", "address", address)

	// Create Redis client
	redisClient := c.clientFactory(address, password)
	if redisClient == nil {
		return fmt.Errorf("failed to create Redis client")
	}

	// Test the connection
	err := redisClient.Ping(ctx)
	if err != nil {
		redisClient.Close()
		return fmt.Errorf("failed to ping Valkey cluster: %w", err)
	}

	c.client = redisClient
	c.connected = true

	log.Info("Successfully connected to Valkey cluster")
	return nil
}

// IsConnected returns connection status
func (c *connectionManager) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.client != nil
}

// GetClient returns the Redis client for stream operations
func (c *connectionManager) GetClient() RedisClient {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.client
}

// Close closes the connection
func (c *connectionManager) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		c.connected = false
		return err
	}

	c.connected = false
	return nil
}
