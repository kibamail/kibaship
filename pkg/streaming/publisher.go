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
	"sync"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// projectStreamPublisher implements ProjectStreamPublisher
type projectStreamPublisher struct {
	connectionManager ConnectionManager
	timeProvider      TimeProvider
	config            *Config
	sequenceNumbers   map[string]int64 // projectUUID -> sequence number
	mutex             sync.Mutex
}

// NewProjectStreamPublisher creates a new project stream publisher
func NewProjectStreamPublisher(
	connectionManager ConnectionManager, timeProvider TimeProvider, config *Config,
) ProjectStreamPublisher {
	return &projectStreamPublisher{
		connectionManager: connectionManager,
		timeProvider:      timeProvider,
		config:            config,
		sequenceNumbers:   make(map[string]int64),
	}
}

// PublishEvent publishes an event to the project's stream
func (p *projectStreamPublisher) PublishEvent(ctx context.Context, event *ResourceEvent) error {
	log := logf.FromContext(ctx).WithName("stream-publisher")

	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	if event.ProjectUUID == "" {
		return fmt.Errorf("event must have a project UUID")
	}

	// Check connection
	if !p.connectionManager.IsConnected() {
		return fmt.Errorf("not connected to Valkey cluster")
	}

	client := p.connectionManager.GetClient()
	if client == nil {
		return fmt.Errorf("redis client is not available")
	}

	// Enrich event with sequence number and timestamp
	p.enrichEvent(event)

	// Generate stream name
	streamName := p.generateStreamName(event.ProjectUUID)

	// Convert event to stream values
	values, err := p.eventToStreamValues(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	log.V(1).Info("Publishing event to stream",
		"stream", streamName,
		"eventType", event.Operation,
		"resourceType", event.ResourceType,
		"resourceUUID", event.ResourceUUID)

	// Publish to stream
	entryID, err := client.XAdd(ctx, streamName, values)
	if err != nil {
		return fmt.Errorf("failed to publish event to stream %s: %w", streamName, err)
	}

	log.Info("Successfully published event to stream",
		"stream", streamName,
		"entryID", entryID,
		"eventID", event.EventID)

	return nil
}

// PublishBatch publishes multiple events in batch
func (p *projectStreamPublisher) PublishBatch(ctx context.Context, events []*ResourceEvent) error {
	log := logf.FromContext(ctx).WithName("stream-publisher")

	if len(events) == 0 {
		return nil
	}

	log.Info("Publishing batch of events", "count", len(events))

	// Group events by project for efficient streaming
	projectEvents := make(map[string][]*ResourceEvent)
	for _, event := range events {
		if event == nil || event.ProjectUUID == "" {
			log.V(1).Info("Skipping invalid event in batch")
			continue
		}
		projectEvents[event.ProjectUUID] = append(projectEvents[event.ProjectUUID], event)
	}

	// Publish events for each project
	var lastError error
	successCount := 0

	for projectUUID, projectEventList := range projectEvents {
		for _, event := range projectEventList {
			err := p.PublishEvent(ctx, event)
			if err != nil {
				log.Error(err, "Failed to publish event in batch",
					"projectUUID", projectUUID,
					"eventID", event.EventID)
				lastError = err
			} else {
				successCount++
			}
		}
	}

	log.Info("Completed batch publishing",
		"total", len(events),
		"successful", successCount,
		"failed", len(events)-successCount)

	return lastError
}

// enrichEvent adds sequence number and ensures timestamp
func (p *projectStreamPublisher) enrichEvent(event *ResourceEvent) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = p.timeProvider.Now()
	}

	// Increment sequence number for the project
	p.sequenceNumbers[event.ProjectUUID]++
	event.Metadata.SequenceNumber = p.sequenceNumbers[event.ProjectUUID]
}

// generateStreamName generates the stream name for a project with sharding support
func (p *projectStreamPublisher) generateStreamName(projectUUID string) string {
	if !p.config.StreamShardingEnabled {
		// Use hash tag to ensure all streams for a project go to same slot
		return fmt.Sprintf("{project:%s}:events", projectUUID)
	}

	// Check if this is a high-traffic project that needs sharding
	if p.isHighTrafficProject(projectUUID) {
		// Distribute across multiple shards for high-traffic projects
		shardIndex := p.getShardIndex(projectUUID)
		return fmt.Sprintf("{project:%s}:events:%d", projectUUID, shardIndex)
	}

	// Regular projects use single stream with hash tag
	return fmt.Sprintf("{project:%s}:events", projectUUID)
}

// isHighTrafficProject determines if a project should use stream sharding
func (p *projectStreamPublisher) isHighTrafficProject(projectUUID string) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check sequence number to determine traffic level
	sequenceNumber, exists := p.sequenceNumbers[projectUUID]
	if !exists {
		return false
	}

	// If sequence number exceeds threshold, consider it high traffic
	return sequenceNumber > p.config.HighTrafficThreshold
}

// getShardIndex returns a consistent shard index for a project
func (p *projectStreamPublisher) getShardIndex(projectUUID string) int {
	// Use simple hash to determine shard
	hash := 0
	for _, char := range projectUUID {
		hash = hash*31 + int(char)
	}
	return (hash % p.config.StreamShardsPerProject) + 1
}

// eventToStreamValues converts a ResourceEvent to Redis stream values
func (p *projectStreamPublisher) eventToStreamValues(event *ResourceEvent) (map[string]interface{}, error) {
	// Serialize the entire event as JSON for the stream
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	values := map[string]interface{}{
		"event_id":      event.EventID,
		"timestamp":     event.Timestamp.Unix(),
		"project_uuid":  event.ProjectUUID,
		"resource_type": string(event.ResourceType),
		"resource_uuid": event.ResourceUUID,
		"operation":     string(event.Operation),
		"event_data":    string(eventJSON),
		"sequence":      event.Metadata.SequenceNumber,
	}

	// Add optional fields if present
	if event.WorkspaceUUID != "" {
		values["workspace_uuid"] = event.WorkspaceUUID
	}
	if event.ResourceSlug != "" {
		values["resource_slug"] = event.ResourceSlug
	}
	if event.Namespace != "" {
		values["namespace"] = event.Namespace
	}
	if event.Metadata.ParentResourceUUID != "" {
		values["parent_resource_uuid"] = event.Metadata.ParentResourceUUID
	}

	return values, nil
}
