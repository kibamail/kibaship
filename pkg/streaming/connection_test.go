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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ConnectionManager", func() {
	var (
		config     *Config
		mockClient *mockRedisClient
		manager    ConnectionManager
	)

	BeforeEach(func() {
		config = &Config{
			Namespace:         "test-namespace",
			ValkeyServiceName: "test-service",
			ValkeyPort:        6379,
		}
		mockClient = &mockRedisClient{}
	})

	Describe("Connect", func() {
		Context("when connection is successful", func() {
			It("should connect without errors", func() {
				password := "test-password"
				clientFactory := func(address, passwordParam string) RedisClient {
					expectedAddress := "test-service.test-namespace.svc.cluster.local:6379"
					Expect(address).To(Equal(expectedAddress))
					Expect(passwordParam).To(Equal(password))
					return mockClient
				}

				manager = NewConnectionManagerWithFactory(config, clientFactory).(*connectionManager)
				mockClient.On("Ping", mock.Anything).Return(nil)

				err := manager.Connect(context.Background(), password)
				Expect(err).NotTo(HaveOccurred())
				Expect(manager.IsConnected()).To(BeTrue())
				Expect(manager.GetClient()).NotTo(BeNil())

				mockClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when ping fails", func() {
			It("should return an error", func() {
				password := "test-password"
				clientFactory := func(address, passwordParam string) RedisClient {
					expectedAddress := "test-service.test-namespace.svc.cluster.local:6379"
					Expect(address).To(Equal(expectedAddress))
					Expect(passwordParam).To(Equal(password))
					return mockClient
				}

				manager = NewConnectionManagerWithFactory(config, clientFactory).(*connectionManager)
				mockClient.On("Ping", mock.Anything).Return(errors.New("connection failed"))
				mockClient.On("Close").Return(nil)

				err := manager.Connect(context.Background(), password)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to ping Valkey cluster"))
				Expect(manager.IsConnected()).To(BeFalse())

				mockClient.AssertExpectations(GinkgoT())
			})
		})

		Context("with empty password", func() {
			It("should allow empty password", func() {
				password := ""
				clientFactory := func(address, passwordParam string) RedisClient {
					expectedAddress := "test-service.test-namespace.svc.cluster.local:6379"
					Expect(address).To(Equal(expectedAddress))
					Expect(passwordParam).To(Equal(""))
					return mockClient
				}

				manager = NewConnectionManagerWithFactory(config, clientFactory).(*connectionManager)
				mockClient.On("Ping", mock.Anything).Return(nil)

				err := manager.Connect(context.Background(), password)
				Expect(err).NotTo(HaveOccurred())
				Expect(manager.IsConnected()).To(BeTrue())
				Expect(manager.GetClient()).NotTo(BeNil())

				mockClient.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("IsConnected", func() {
		It("should return false initially", func() {
			manager := NewConnectionManager(config)
			Expect(manager.IsConnected()).To(BeFalse())
		})

		It("should return true after successful connection", func() {
			clientFactory := func(address, password string) RedisClient {
				return mockClient
			}

			manager = NewConnectionManagerWithFactory(config, clientFactory).(*connectionManager)
			mockClient.On("Ping", mock.Anything).Return(nil)

			err := manager.Connect(context.Background(), "password")
			Expect(err).NotTo(HaveOccurred())
			Expect(manager.IsConnected()).To(BeTrue())

			mockClient.AssertExpectations(GinkgoT())
		})
	})

	Describe("GetClient", func() {
		It("should return nil initially", func() {
			manager := NewConnectionManager(config)
			Expect(manager.GetClient()).To(BeNil())
		})

		It("should return client after successful connection", func() {
			clientFactory := func(address, password string) RedisClient {
				return mockClient
			}

			manager = NewConnectionManagerWithFactory(config, clientFactory).(*connectionManager)
			mockClient.On("Ping", mock.Anything).Return(nil)

			err := manager.Connect(context.Background(), "password")
			Expect(err).NotTo(HaveOccurred())
			Expect(manager.GetClient()).To(Equal(mockClient))

			mockClient.AssertExpectations(GinkgoT())
		})
	})

	Describe("Close", func() {
		It("should not error when not connected", func() {
			manager := NewConnectionManager(config)
			err := manager.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should close connection and clean up state", func() {
			clientFactory := func(address, password string) RedisClient {
				return mockClient
			}

			manager = NewConnectionManagerWithFactory(config, clientFactory).(*connectionManager)
			mockClient.On("Ping", mock.Anything).Return(nil)
			mockClient.On("Close").Return(nil)

			err := manager.Connect(context.Background(), "password")
			Expect(err).NotTo(HaveOccurred())

			// Now close
			err = manager.Close()
			Expect(err).NotTo(HaveOccurred())
			Expect(manager.IsConnected()).To(BeFalse())
			Expect(manager.GetClient()).To(BeNil())

			mockClient.AssertExpectations(GinkgoT())
		})
	})

	Describe("when already connected", func() {
		It("should not create new client on subsequent connections", func() {
			callCount := 0
			clientFactory := func(address, password string) RedisClient {
				callCount++
				return mockClient
			}

			manager = NewConnectionManagerWithFactory(config, clientFactory).(*connectionManager)
			mockClient.On("Ping", mock.Anything).Return(nil)

			// Connect first time
			err := manager.Connect(context.Background(), "password")
			Expect(err).NotTo(HaveOccurred())
			Expect(callCount).To(Equal(1))

			// Connect second time - should not create new client
			err = manager.Connect(context.Background(), "password")
			Expect(err).NotTo(HaveOccurred())
			Expect(callCount).To(Equal(1)) // Should not increment

			mockClient.AssertExpectations(GinkgoT())
		})
	})
})
