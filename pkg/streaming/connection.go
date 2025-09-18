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
	"sync"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ValkeyClientFactory creates Valkey cluster clients
type ValkeyClientFactory func(seedAddress, password string, config *Config) (ValkeyClient, error)

// clusterConnectionManager implements ConnectionManager with cluster awareness
type clusterConnectionManager struct {
	config         *Config
	client         ValkeyClient
	connected      bool
	clusterHealthy bool
	mutex          sync.RWMutex
	clientFactory  ValkeyClientFactory
}

// NewConnectionManager creates a new cluster-aware connection manager
func NewConnectionManager(config *Config) ConnectionManager {
	return &clusterConnectionManager{
		config:        config,
		clientFactory: NewValkeyClusterClient,
	}
}

// NewConnectionManagerWithFactory creates a new connection manager with custom client factory
func NewConnectionManagerWithFactory(config *Config, clientFactory ValkeyClientFactory) ConnectionManager {
	return &clusterConnectionManager{
		config:        config,
		clientFactory: clientFactory,
	}
}

// InitializeCluster establishes connection to Valkey cluster with auto-discovery
func (c *clusterConnectionManager) InitializeCluster(ctx context.Context, seedAddress, password string) error {
	log := logf.FromContext(ctx).WithName("cluster-connection-manager")

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected && c.client != nil {
		log.Info("Already connected to Valkey cluster")
		return nil
	}

	log.Info("Initializing Valkey cluster connection with auto-discovery",
		"seedAddress", seedAddress,
		"clusterEnabled", c.config.ClusterEnabled)

	// Create cluster client with auto-discovery
	valkeyClient, err := c.clientFactory(seedAddress, password, c.config)
	if err != nil {
		return fmt.Errorf("failed to create Valkey cluster client: %w", err)
	}

	// Test the connection
	err = valkeyClient.Ping(ctx)
	if err != nil {
		_ = valkeyClient.Close()
		return fmt.Errorf("failed to ping Valkey cluster: %w", err)
	}

	// Verify cluster mode and discover nodes
	if c.config.ClusterEnabled {
		clusterInfo, err := valkeyClient.ClusterNodes(ctx)
		if err != nil {
			log.V(1).Info("Warning: Could not get cluster nodes info, proceeding anyway", "error", err)
		} else {
			nodeCount := len(strings.Split(clusterInfo, "\n")) - 1 // -1 for empty last line
			log.Info("Cluster topology discovered", "nodeCount", nodeCount)
		}
	}

	c.client = valkeyClient
	c.connected = true
	c.clusterHealthy = true

	log.Info("Successfully connected to Valkey cluster with auto-discovery")
	return nil
}

// IsConnected returns connection status
func (c *clusterConnectionManager) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.client != nil
}

// GetClient returns the Valkey client for stream operations
func (c *clusterConnectionManager) GetClient() ValkeyClient {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.client
}

// IsClusterHealthy returns cluster health status
func (c *clusterConnectionManager) IsClusterHealthy() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected || c.client == nil {
		return false
	}

	// Perform health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.client.Ping(ctx)
	healthy := err == nil

	// Update health status
	c.mutex.RUnlock()
	c.mutex.Lock()
	c.clusterHealthy = healthy
	c.mutex.Unlock()
	c.mutex.RLock()

	return healthy
}

// Close closes the connection
func (c *clusterConnectionManager) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		c.connected = false
		c.clusterHealthy = false
		return err
	}

	c.connected = false
	c.clusterHealthy = false
	return nil
}
