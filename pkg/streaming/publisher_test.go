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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProjectStreamPublisher_PublishEvent(t *testing.T) {
	tests := []struct {
		name          string
		event         *ResourceEvent
		setupMocks    func(*mockConnectionManager, *mockRedisClient)
		expectedError string
	}{
		{
			name: "successful publish",
			event: &ResourceEvent{
				EventID:      uuid.New().String(),
				ProjectUUID:  uuid.New().String(),
				ResourceType: ResourceTypeApplication,
				ResourceUUID: uuid.New().String(),
				Operation:    OperationCreate,
				Timestamp:    time.Now(),
			},
			setupMocks: func(connMgr *mockConnectionManager, redisClient *mockRedisClient) {
				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(redisClient)
				redisClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("1703123456789-0", nil)
			},
		},
		{
			name:          "nil event",
			event:         nil,
			expectedError: "event cannot be nil",
			setupMocks:    func(*mockConnectionManager, *mockRedisClient) {},
		},
		{
			name: "missing project UUID",
			event: &ResourceEvent{
				EventID:      uuid.New().String(),
				ResourceType: ResourceTypeApplication,
				ResourceUUID: uuid.New().String(),
				Operation:    OperationCreate,
			},
			expectedError: "event must have a project UUID",
			setupMocks:    func(*mockConnectionManager, *mockRedisClient) {},
		},
		{
			name: "not connected",
			event: &ResourceEvent{
				EventID:      uuid.New().String(),
				ProjectUUID:  uuid.New().String(),
				ResourceType: ResourceTypeApplication,
				ResourceUUID: uuid.New().String(),
				Operation:    OperationCreate,
			},
			setupMocks: func(connMgr *mockConnectionManager, redisClient *mockRedisClient) {
				connMgr.On("IsConnected").Return(false)
			},
			expectedError: "not connected to Valkey cluster",
		},
		{
			name: "client not available",
			event: &ResourceEvent{
				EventID:      uuid.New().String(),
				ProjectUUID:  uuid.New().String(),
				ResourceType: ResourceTypeApplication,
				ResourceUUID: uuid.New().String(),
				Operation:    OperationCreate,
			},
			setupMocks: func(connMgr *mockConnectionManager, redisClient *mockRedisClient) {
				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(nil)
			},
			expectedError: "Redis client is not available",
		},
		{
			name: "redis error",
			event: &ResourceEvent{
				EventID:      uuid.New().String(),
				ProjectUUID:  uuid.New().String(),
				ResourceType: ResourceTypeApplication,
				ResourceUUID: uuid.New().String(),
				Operation:    OperationCreate,
				Timestamp:    time.Now(),
			},
			setupMocks: func(connMgr *mockConnectionManager, redisClient *mockRedisClient) {
				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(redisClient)
				redisClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("redis connection failed"))
			},
			expectedError: "failed to publish event to stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connMgr := &mockConnectionManager{}
			redisClient := &mockRedisClient{}
			timeProvider := &mockTimeProvider{}
			config := &Config{}

			tt.setupMocks(connMgr, redisClient)

			if tt.event != nil && tt.event.Timestamp.IsZero() {
				timeProvider.On("Now").Return(time.Now())
			}

			publisher := NewProjectStreamPublisher(connMgr, timeProvider, config)

			err := publisher.PublishEvent(context.Background(), tt.event)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			connMgr.AssertExpectations(t)
			redisClient.AssertExpectations(t)
		})
	}
}

func TestProjectStreamPublisher_PublishBatch(t *testing.T) {
	projectUUID1 := uuid.New().String()
	projectUUID2 := uuid.New().String()

	tests := []struct {
		name       string
		events     []*ResourceEvent
		setupMocks func(*mockConnectionManager, *mockRedisClient)
		wantError  bool
	}{
		{
			name:   "empty batch",
			events: []*ResourceEvent{},
			setupMocks: func(*mockConnectionManager, *mockRedisClient) {
				// No setup needed for empty batch
			},
		},
		{
			name: "successful batch",
			events: []*ResourceEvent{
				{
					EventID:      uuid.New().String(),
					ProjectUUID:  projectUUID1,
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
					Timestamp:    time.Now(),
				},
				{
					EventID:      uuid.New().String(),
					ProjectUUID:  projectUUID2,
					ResourceType: ResourceTypeProject,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationUpdate,
					Timestamp:    time.Now(),
				},
			},
			setupMocks: func(connMgr *mockConnectionManager, redisClient *mockRedisClient) {
				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(redisClient)
				redisClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("1703123456789-0", nil).Times(2)
			},
		},
		{
			name: "batch with invalid events",
			events: []*ResourceEvent{
				nil, // Invalid event
				{
					EventID:      uuid.New().String(),
					ProjectUUID:  "", // Missing project UUID
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
				},
				{
					EventID:      uuid.New().String(),
					ProjectUUID:  projectUUID1,
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
					Timestamp:    time.Now(),
				},
			},
			setupMocks: func(connMgr *mockConnectionManager, redisClient *mockRedisClient) {
				// Only one valid event should be processed
				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(redisClient)
				redisClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("1703123456789-0", nil).Once()
			},
		},
		{
			name: "partial failure",
			events: []*ResourceEvent{
				{
					EventID:      uuid.New().String(),
					ProjectUUID:  projectUUID1,
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
					Timestamp:    time.Now(),
				},
				{
					EventID:      uuid.New().String(),
					ProjectUUID:  projectUUID2,
					ResourceType: ResourceTypeProject,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationUpdate,
					Timestamp:    time.Now(),
				},
			},
			setupMocks: func(connMgr *mockConnectionManager, redisClient *mockRedisClient) {
				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(redisClient)
				// First call succeeds, second fails
				redisClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("1703123456789-0", nil).Once()
				redisClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("redis error")).Once()
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connMgr := &mockConnectionManager{}
			redisClient := &mockRedisClient{}
			timeProvider := &mockTimeProvider{}
			config := &Config{}

			tt.setupMocks(connMgr, redisClient)

			publisher := NewProjectStreamPublisher(connMgr, timeProvider, config)

			err := publisher.PublishBatch(context.Background(), tt.events)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			connMgr.AssertExpectations(t)
			redisClient.AssertExpectations(t)
		})
	}
}

func TestProjectStreamPublisher_enrichEvent(t *testing.T) {
	connMgr := &mockConnectionManager{}
	timeProvider := &mockTimeProvider{}
	config := &Config{}

	publisher := NewProjectStreamPublisher(connMgr, timeProvider, config).(*projectStreamPublisher)

	projectUUID := uuid.New().String()
	event := &ResourceEvent{
		EventID:      uuid.New().String(),
		ProjectUUID:  projectUUID,
		ResourceType: ResourceTypeApplication,
		ResourceUUID: uuid.New().String(),
		Operation:    OperationCreate,
		// No timestamp set
	}

	now := time.Now()
	timeProvider.On("Now").Return(now)

	publisher.enrichEvent(event)

	// Check timestamp was set
	assert.Equal(t, now, event.Timestamp)

	// Check sequence number was set
	assert.Equal(t, int64(1), event.Metadata.SequenceNumber)

	// Publish another event for the same project
	event2 := &ResourceEvent{
		EventID:      uuid.New().String(),
		ProjectUUID:  projectUUID,
		ResourceType: ResourceTypeApplication,
		ResourceUUID: uuid.New().String(),
		Operation:    OperationUpdate,
		Timestamp:    time.Now(), // Already set
	}

	originalTimestamp := event2.Timestamp
	publisher.enrichEvent(event2)

	// Timestamp should not be overridden
	assert.Equal(t, originalTimestamp, event2.Timestamp)

	// Sequence number should increment
	assert.Equal(t, int64(2), event2.Metadata.SequenceNumber)

	timeProvider.AssertExpectations(t)
}

func TestProjectStreamPublisher_generateStreamName(t *testing.T) {
	connMgr := &mockConnectionManager{}
	timeProvider := &mockTimeProvider{}
	config := &Config{}

	publisher := NewProjectStreamPublisher(connMgr, timeProvider, config).(*projectStreamPublisher)

	projectUUID := uuid.New().String()
	streamName := publisher.generateStreamName(projectUUID)

	expected := "project:" + projectUUID + ":events"
	assert.Equal(t, expected, streamName)
}

func TestProjectStreamPublisher_eventToStreamValues(t *testing.T) {
	connMgr := &mockConnectionManager{}
	timeProvider := &mockTimeProvider{}
	config := &Config{}

	publisher := NewProjectStreamPublisher(connMgr, timeProvider, config).(*projectStreamPublisher)

	event := &ResourceEvent{
		EventID:       uuid.New().String(),
		ProjectUUID:   uuid.New().String(),
		WorkspaceUUID: uuid.New().String(),
		ResourceType:  ResourceTypeApplication,
		ResourceUUID:  uuid.New().String(),
		ResourceSlug:  "test-app",
		Namespace:     "default",
		Operation:     OperationCreate,
		Timestamp:     time.Unix(1703123456, 0),
		Metadata: EventMetadata{
			SequenceNumber:     5,
			ParentResourceUUID: uuid.New().String(),
		},
	}

	values, err := publisher.eventToStreamValues(event)
	assert.NoError(t, err)

	// Check all expected fields are present
	assert.Equal(t, event.EventID, values["event_id"])
	assert.Equal(t, int64(1703123456), values["timestamp"])
	assert.Equal(t, event.ProjectUUID, values["project_uuid"])
	assert.Equal(t, event.WorkspaceUUID, values["workspace_uuid"])
	assert.Equal(t, string(event.ResourceType), values["resource_type"])
	assert.Equal(t, event.ResourceUUID, values["resource_uuid"])
	assert.Equal(t, event.ResourceSlug, values["resource_slug"])
	assert.Equal(t, event.Namespace, values["namespace"])
	assert.Equal(t, string(event.Operation), values["operation"])
	assert.Equal(t, int64(5), values["sequence"])
	assert.Equal(t, event.Metadata.ParentResourceUUID, values["parent_resource_uuid"])
	assert.Contains(t, values, "event_data")
}
