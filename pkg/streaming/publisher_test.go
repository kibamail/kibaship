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
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ProjectStreamPublisher", func() {
	var (
		connMgr      *mockConnectionManager
		valkeyClient *mockValkeyClient
		timeProvider *mockTimeProvider
		config       *Config
		publisher    ProjectStreamPublisher
	)

	BeforeEach(func() {
		connMgr = &mockConnectionManager{}
		valkeyClient = &mockValkeyClient{}
		timeProvider = &mockTimeProvider{}
		config = &Config{
			StreamShardingEnabled:  true,
			StreamShardsPerProject: 4,
			HighTrafficThreshold:   1000,
		}
		publisher = NewProjectStreamPublisher(connMgr, timeProvider, config)
	})

	Describe("PublishEvent", func() {
		Context("when publishing successfully", func() {
			It("should publish the event without errors", func() {
				event := &ResourceEvent{
					EventID:      uuid.New().String(),
					ProjectUUID:  uuid.New().String(),
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
					Timestamp:    time.Now(),
				}

				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(valkeyClient)
				valkeyClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("1703123456789-0", nil)

				err := publisher.PublishEvent(context.Background(), event)
				Expect(err).NotTo(HaveOccurred())

				connMgr.AssertExpectations(GinkgoT())
				valkeyClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when event is nil", func() {
			It("should return an error", func() {
				err := publisher.PublishEvent(context.Background(), nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("event cannot be nil"))
			})
		})

		Context("when project UUID is missing", func() {
			It("should return an error", func() {
				event := &ResourceEvent{
					EventID:      uuid.New().String(),
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
				}

				err := publisher.PublishEvent(context.Background(), event)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("event must have a project UUID"))
			})
		})

		Context("when not connected to Valkey", func() {
			It("should return an error", func() {
				event := &ResourceEvent{
					EventID:      uuid.New().String(),
					ProjectUUID:  uuid.New().String(),
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
				}

				connMgr.On("IsConnected").Return(false)

				err := publisher.PublishEvent(context.Background(), event)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not connected to Valkey cluster"))

				connMgr.AssertExpectations(GinkgoT())
			})
		})

		Context("when Redis client is not available", func() {
			It("should return an error", func() {
				event := &ResourceEvent{
					EventID:      uuid.New().String(),
					ProjectUUID:  uuid.New().String(),
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
				}

				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(nil)

				err := publisher.PublishEvent(context.Background(), event)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Redis client is not available"))

				connMgr.AssertExpectations(GinkgoT())
			})
		})

		Context("when Redis operation fails", func() {
			It("should return an error", func() {
				event := &ResourceEvent{
					EventID:      uuid.New().String(),
					ProjectUUID:  uuid.New().String(),
					ResourceType: ResourceTypeApplication,
					ResourceUUID: uuid.New().String(),
					Operation:    OperationCreate,
					Timestamp:    time.Now(),
				}

				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(valkeyClient)
				valkeyClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("redis connection failed"))

				err := publisher.PublishEvent(context.Background(), event)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to publish event to stream"))

				connMgr.AssertExpectations(GinkgoT())
				valkeyClient.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("PublishBatch", func() {
		var (
			projectUUID1 string
			projectUUID2 string
		)

		BeforeEach(func() {
			projectUUID1 = uuid.New().String()
			projectUUID2 = uuid.New().String()
		})

		Context("with empty batch", func() {
			It("should handle empty batch without errors", func() {
				events := []*ResourceEvent{}

				err := publisher.PublishBatch(context.Background(), events)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with successful batch", func() {
			It("should publish all events", func() {
				events := []*ResourceEvent{
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
				}

				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(valkeyClient)
				valkeyClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("1703123456789-0", nil).Times(2)

				err := publisher.PublishBatch(context.Background(), events)
				Expect(err).NotTo(HaveOccurred())

				connMgr.AssertExpectations(GinkgoT())
				valkeyClient.AssertExpectations(GinkgoT())
			})
		})

		Context("with invalid events in batch", func() {
			It("should process only valid events", func() {
				events := []*ResourceEvent{
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
				}

				// Only one valid event should be processed
				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(valkeyClient)
				valkeyClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("1703123456789-0", nil).Once()

				err := publisher.PublishBatch(context.Background(), events)
				Expect(err).NotTo(HaveOccurred())

				connMgr.AssertExpectations(GinkgoT())
				valkeyClient.AssertExpectations(GinkgoT())
			})
		})

		Context("with partial failure", func() {
			It("should return error when some events fail", func() {
				events := []*ResourceEvent{
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
				}

				connMgr.On("IsConnected").Return(true)
				connMgr.On("GetClient").Return(valkeyClient)
				// First call succeeds, second fails
				valkeyClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("1703123456789-0", nil).Once()
				valkeyClient.On("XAdd", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("redis error")).Once()

				err := publisher.PublishBatch(context.Background(), events)
				Expect(err).To(HaveOccurred())

				connMgr.AssertExpectations(GinkgoT())
				valkeyClient.AssertExpectations(GinkgoT())
			})
		})
	})
})
