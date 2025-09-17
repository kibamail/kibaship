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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ConnectionManager", func() {
	var (
		config     *Config
		mockClient *mockValkeyClient
		manager    ConnectionManager
	)

	BeforeEach(func() {
		config = &Config{
			Namespace:         "test-namespace",
			ValkeyServiceName: "test-service",
			ValkeyPort:        6379,
			ClusterEnabled:    true,
			ConnectionTimeout: 30 * time.Second,
			RequestTimeout:    10 * time.Second,
		}
		mockClient = &mockValkeyClient{}
	})

	Describe("InitializeCluster", func() {
		Context("when connection is successful", func() {
			It("should connect without errors", func() {
				password := "test-password"
				seedAddress := "test-service.test-namespace.svc.cluster.local"
				clientFactory := func(seedAddr, passwordParam string, configParam *Config) (ValkeyClient, error) {
					Expect(seedAddr).To(Equal(seedAddress))
					Expect(passwordParam).To(Equal(password))
					return mockClient, nil
				}

				manager = NewConnectionManagerWithFactory(config, clientFactory).(*clusterConnectionManager)
				mockClient.On("Ping", mock.Anything).Return(nil)
				mockClient.On("ClusterNodes", mock.Anything).Return("node1:6379 master\nnode2:6379 master\n", nil)

				err := manager.InitializeCluster(context.Background(), seedAddress, password)
				Expect(err).NotTo(HaveOccurred())
				Expect(manager.IsConnected()).To(BeTrue())
				Expect(manager.GetClient()).NotTo(BeNil())

				mockClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when ping fails", func() {
			It("should return an error", func() {
				password := "test-password"
				seedAddress := "test-service.test-namespace.svc.cluster.local"
				clientFactory := func(seedAddr, passwordParam string, configParam *Config) (ValkeyClient, error) {
					return mockClient, nil
				}

				manager = NewConnectionManagerWithFactory(config, clientFactory).(*clusterConnectionManager)
				mockClient.On("Ping", mock.Anything).Return(errors.New("connection failed"))
				mockClient.On("Close").Return(nil)

				err := manager.InitializeCluster(context.Background(), seedAddress, password)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to ping Valkey cluster"))
				Expect(manager.IsConnected()).To(BeFalse())

				mockClient.AssertExpectations(GinkgoT())
			})
		})

		Context("with empty password", func() {
			It("should allow empty password", func() {
				password := ""
				seedAddress := "test-service.test-namespace.svc.cluster.local"
				clientFactory := func(seedAddr, passwordParam string, configParam *Config) (ValkeyClient, error) {
					Expect(seedAddr).To(Equal(seedAddress))
					Expect(passwordParam).To(Equal(""))
					return mockClient, nil
				}

				manager = NewConnectionManagerWithFactory(config, clientFactory).(*clusterConnectionManager)
				mockClient.On("Ping", mock.Anything).Return(nil)
				mockClient.On("ClusterNodes", mock.Anything).Return("node1:6379 master\n", nil)

				err := manager.InitializeCluster(context.Background(), seedAddress, password)
				Expect(err).NotTo(HaveOccurred())
				Expect(manager.IsConnected()).To(BeTrue())

				mockClient.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("IsClusterHealthy", func() {
		Context("when connected", func() {
			It("should return health status based on ping", func() {
				manager = NewConnectionManager(config).(*clusterConnectionManager)
				manager.(*clusterConnectionManager).client = mockClient
				manager.(*clusterConnectionManager).connected = true

				mockClient.On("Ping", mock.Anything).Return(nil)
				Expect(manager.IsClusterHealthy()).To(BeTrue())

				mockClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when not connected", func() {
			It("should return false", func() {
				manager = NewConnectionManager(config).(*clusterConnectionManager)
				Expect(manager.IsClusterHealthy()).To(BeFalse())
			})
		})
	})

	Describe("Close", func() {
		Context("when connected", func() {
			It("should close connection", func() {
				manager = NewConnectionManager(config).(*clusterConnectionManager)
				manager.(*clusterConnectionManager).client = mockClient
				manager.(*clusterConnectionManager).connected = true

				mockClient.On("Close").Return(nil)
				err := manager.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(manager.IsConnected()).To(BeFalse())

				mockClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when not connected", func() {
			It("should not error", func() {
				manager = NewConnectionManager(config)
				err := manager.Close()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
