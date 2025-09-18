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
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValkeyReadyGate provides a simple blocking interface to wait for Valkey cluster readiness
type ValkeyReadyGate interface {
	// WaitForReady blocks until Valkey cluster is ready or timeout is reached
	WaitForReady(ctx context.Context) error
}

// ValkeyReadinessMonitor monitors Valkey cluster availability (DEPRECATED: use ValkeyReadyGate)
type ValkeyReadinessMonitor interface {
	// WaitForReady waits for Valkey cluster to become ready within timeout
	WaitForReady(ctx context.Context, timeout time.Duration) error
	// IsReady checks if Valkey cluster is currently ready
	IsReady(ctx context.Context) bool
}

// SecretManager manages Valkey authentication credentials
type SecretManager interface {
	// GetValkeyPassword retrieves the password from the Valkey secret
	GetValkeyPassword(ctx context.Context) (string, error)
}

// ConnectionManager manages Valkey cluster connections
type ConnectionManager interface {
	// InitializeCluster establishes connection to Valkey cluster with auto-discovery
	InitializeCluster(ctx context.Context, seedAddress, password string) error
	// IsConnected returns connection status
	IsConnected() bool
	// GetClient returns the Valkey client for stream operations
	GetClient() ValkeyClient
	// IsClusterHealthy returns cluster health status
	IsClusterHealthy() bool
	// Close closes the connection
	Close() error
}

// ValkeyClient abstracts Valkey cluster operations for testing
type ValkeyClient interface {
	// XAdd adds an entry to a Valkey stream
	XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error)
	// Ping tests the connection
	Ping(ctx context.Context) error
	// ClusterNodes returns cluster node information
	ClusterNodes(ctx context.Context) (string, error)
	// Close closes the client
	Close() error
}

// ProjectStreamPublisher publishes events to project-specific streams
type ProjectStreamPublisher interface {
	// PublishEvent publishes an event to the project's stream
	PublishEvent(ctx context.Context, event *ResourceEvent) error
	// PublishBatch publishes multiple events in batch
	PublishBatch(ctx context.Context, events []*ResourceEvent) error
}

// StartupSequenceController orchestrates the startup sequence
type StartupSequenceController interface {
	// Initialize runs the complete startup sequence
	Initialize(ctx context.Context) error
	// IsReady returns true if streaming is ready
	IsReady() bool
	// Shutdown gracefully shuts down the streaming components
	Shutdown(ctx context.Context) error
}

// KubernetesClient abstracts Kubernetes operations for testing
type KubernetesClient interface {
	// Get retrieves a Kubernetes object
	Get(ctx context.Context, key client.ObjectKey, obj client.Object) error
	// List lists Kubernetes objects
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	// Watch watches for changes to Kubernetes objects
	Watch(ctx context.Context, list client.ObjectList, opts ...client.ListOption) (watch.Interface, error)
}

// TimeProvider abstracts time operations for testing
type TimeProvider interface {
	// Now returns current time
	Now() time.Time
	// Sleep pauses execution
	Sleep(duration time.Duration)
	// After returns a channel that fires after duration
	After(duration time.Duration) <-chan time.Time
}

// ResourceEvent represents a resource status event
type ResourceEvent struct {
	EventID       string                 `json:"event_id"`
	Timestamp     time.Time              `json:"timestamp"`
	ProjectUUID   string                 `json:"project_uuid"`
	WorkspaceUUID string                 `json:"workspace_uuid"`
	ResourceType  ResourceType           `json:"resource_type"`
	ResourceUUID  string                 `json:"resource_uuid"`
	ResourceSlug  string                 `json:"resource_slug"`
	Operation     OperationType          `json:"operation"`
	Namespace     string                 `json:"namespace"`
	Payload       map[string]interface{} `json:"payload"`
	Metadata      EventMetadata          `json:"metadata"`
}

// ResourceType represents the type of Kubernetes resource
type ResourceType string

const (
	ResourceTypeProject           ResourceType = "Project"
	ResourceTypeApplication       ResourceType = "Application"
	ResourceTypeDeployment        ResourceType = "Deployment"
	ResourceTypeApplicationDomain ResourceType = "ApplicationDomain"
)

// OperationType represents the operation performed on the resource
type OperationType string

const (
	OperationCreate OperationType = "Create"
	OperationUpdate OperationType = "Update"
	OperationDelete OperationType = "Delete"
	OperationFailed OperationType = "Failed"
	OperationReady  OperationType = "Ready"
)

// EventMetadata contains additional event context
type EventMetadata struct {
	ReconciliationID   string `json:"reconciliation_id"`
	ControllerVersion  string `json:"controller_version"`
	SequenceNumber     int64  `json:"sequence_number"`
	ParentResourceUUID string `json:"parent_resource_uuid,omitempty"`
}

// Config holds streaming configuration
type Config struct {
	ValkeyServiceName string
	ValkeySecretName  string
	ValkeyPort        int
	Namespace         string
	StartupTimeout    time.Duration
	BatchSize         int
	BatchTimeout      time.Duration
	RetryAttempts     int
	// Cluster configuration
	ClusterEnabled    bool
	ConnectionTimeout time.Duration
	RequestTimeout    time.Duration
	// Stream sharding configuration
	StreamShardingEnabled  bool
	StreamShardsPerProject int
	// High traffic project configuration
	HighTrafficThreshold int64
}

// Validate validates the resource event
func (r *ResourceEvent) Validate() error {
	if r.EventID == "" {
		return fmt.Errorf("event ID is required")
	}
	if r.ProjectUUID == "" {
		return fmt.Errorf("project UUID is required")
	}
	if r.ResourceType == "" || (r.ResourceType != ResourceTypeProject &&
		r.ResourceType != ResourceTypeApplication && r.ResourceType != ResourceTypeDeployment &&
		r.ResourceType != ResourceTypeApplicationDomain) {
		return fmt.Errorf("valid resource type is required")
	}
	if r.Operation == "" || (r.Operation != OperationCreate && r.Operation != OperationUpdate &&
		r.Operation != OperationDelete && r.Operation != OperationFailed && r.Operation != OperationReady) {
		return fmt.Errorf("valid operation is required")
	}
	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if c.ValkeyServiceName == "" {
		return fmt.Errorf("valkey service name is required")
	}
	if c.ValkeyPort <= 0 {
		return fmt.Errorf("valid Valkey port is required")
	}
	if c.StartupTimeout <= 0 {
		return fmt.Errorf("startup timeout must be positive")
	}
	return nil
}

// NewResourceEvent creates a new resource event
func NewResourceEvent(
	projectUUID, workspaceUUID string, resourceType ResourceType, resourceUUID, resourceSlug, namespace string,
	operation OperationType,
) *ResourceEvent {
	return &ResourceEvent{
		EventID:       uuid.New().String(),
		Timestamp:     time.Now(),
		ProjectUUID:   projectUUID,
		WorkspaceUUID: workspaceUUID,
		ResourceType:  resourceType,
		ResourceUUID:  resourceUUID,
		ResourceSlug:  resourceSlug,
		Namespace:     namespace,
		Operation:     operation,
		Payload:       make(map[string]interface{}),
		Metadata:      EventMetadata{},
	}
}

// NewResourceEventFromK8sResource creates a new resource event with the full Kubernetes resource in the payload
func NewResourceEventFromK8sResource(
	projectUUID, workspaceUUID string, resourceType ResourceType, resourceUUID, resourceSlug, namespace string,
	operation OperationType, k8sResource client.Object,
) (*ResourceEvent, error) {
	event := NewResourceEvent(projectUUID, workspaceUUID, resourceType, resourceUUID, resourceSlug, namespace, operation)

	// Serialize the Kubernetes resource to JSON
	resourceJSON, err := json.Marshal(k8sResource)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize Kubernetes resource: %w", err)
	}

	// Parse JSON into map for payload
	var resourceMap map[string]interface{}
	if err := json.Unmarshal(resourceJSON, &resourceMap); err != nil {
		return nil, fmt.Errorf("failed to parse Kubernetes resource JSON: %w", err)
	}

	// Set the resource in payload
	event.Payload["resource"] = resourceMap

	return event, nil
}

// DefaultConfig returns default streaming configuration
func DefaultConfig(namespace string) *Config {
	return &Config{
		ValkeyServiceName:      "kibaship-valkey-cluster-kibaship-com",
		ValkeySecretName:       "kibaship-valkey-cluster-kibaship-com",
		ValkeyPort:             6379,
		Namespace:              namespace,
		StartupTimeout:         5 * time.Minute,
		BatchSize:              100,
		BatchTimeout:           5 * time.Second,
		RetryAttempts:          3,
		ClusterEnabled:         true,
		ConnectionTimeout:      30 * time.Second,
		RequestTimeout:         10 * time.Second,
		StreamShardingEnabled:  true,
		StreamShardsPerProject: 4,
		HighTrafficThreshold:   1000,
	}
}
