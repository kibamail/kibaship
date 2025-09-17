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

var _ = Describe("StartupSequenceController", func() {
	var (
		readiness  *mockValkeyReadinessMonitor
		secret     *mockSecretManager
		conn       *mockConnectionManager
		config     *Config
		controller StartupSequenceController
	)

	BeforeEach(func() {
		readiness = &mockValkeyReadinessMonitor{}
		secret = &mockSecretManager{}
		conn = &mockConnectionManager{}
		config = &Config{
			StartupTimeout:    5 * time.Minute,
			ValkeyServiceName: "test-service",
			Namespace:         "test-namespace",
		}
		controller = NewStartupSequenceController(readiness, secret, conn, config)
	})

	Describe("Initialize", func() {
		Context("when initialization is successful", func() {
			It("should initialize without errors", func() {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
				secret.On("GetValkeyPassword", mock.Anything).Return("test-password", nil)
				conn.On("InitializeCluster", mock.Anything, mock.AnythingOfType("string"), "test-password").Return(nil)
				conn.On("IsConnected").Return(true)

				err := controller.Initialize(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(controller.IsReady()).To(BeTrue())

				readiness.AssertExpectations(GinkgoT())
				secret.AssertExpectations(GinkgoT())
				conn.AssertExpectations(GinkgoT())
			})
		})

		Context("when Valkey readiness fails", func() {
			It("should return an error", func() {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(errors.New("timeout"))

				err := controller.Initialize(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Valkey cluster failed to become ready"))

				readiness.AssertExpectations(GinkgoT())
			})
		})

		Context("when secret retrieval fails", func() {
			It("should return an error", func() {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
				secret.On("GetValkeyPassword", mock.Anything).Return("", errors.New("secret not found"))

				err := controller.Initialize(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Valkey authentication secret not found"))

				readiness.AssertExpectations(GinkgoT())
				secret.AssertExpectations(GinkgoT())
			})
		})

		Context("when connection fails", func() {
			It("should return an error", func() {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
				secret.On("GetValkeyPassword", mock.Anything).Return("test-password", nil)
				conn.On("InitializeCluster", mock.Anything, mock.AnythingOfType("string"), "test-password").Return(errors.New("connection failed"))

				err := controller.Initialize(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to initialize Valkey cluster connection"))

				readiness.AssertExpectations(GinkgoT())
				secret.AssertExpectations(GinkgoT())
				conn.AssertExpectations(GinkgoT())
			})
		})

		Context("when empty password is retrieved from secret", func() {
			It("should return an error", func() {
				readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
				secret.On("GetValkeyPassword", mock.Anything).Return("", nil)

				err := controller.Initialize(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Valkey authentication secret empty after cluster ready"))

				readiness.AssertExpectations(GinkgoT())
				secret.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("IsReady", func() {
		Context("when initialized and connected", func() {
			It("should return true", func() {
				conn.On("IsConnected").Return(true)

				controller := NewStartupSequenceController(readiness, secret, conn, config).(*startupSequenceController)

				// Set the ready state directly for testing
				controller.mutex.Lock()
				controller.ready = true
				controller.mutex.Unlock()

				result := controller.IsReady()
				Expect(result).To(BeTrue())

				conn.AssertExpectations(GinkgoT())
			})
		})

		Context("when not initialized", func() {
			It("should return false", func() {
				controller := NewStartupSequenceController(readiness, secret, conn, config).(*startupSequenceController)

				// Set the ready state directly for testing
				controller.mutex.Lock()
				controller.ready = false
				controller.mutex.Unlock()

				result := controller.IsReady()
				Expect(result).To(BeFalse())

				// No expectations - IsConnected should not be called when not initialized
			})
		})

		Context("when initialized but not connected", func() {
			It("should return false", func() {
				conn.On("IsConnected").Return(false)

				controller := NewStartupSequenceController(readiness, secret, conn, config).(*startupSequenceController)

				// Set the ready state directly for testing
				controller.mutex.Lock()
				controller.ready = true
				controller.mutex.Unlock()

				result := controller.IsReady()
				Expect(result).To(BeFalse())

				conn.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Shutdown", func() {
		Context("when shutdown is successful", func() {
			It("should shutdown without errors", func() {
				conn.On("Close").Return(nil)

				controller := NewStartupSequenceController(readiness, secret, conn, config).(*startupSequenceController)

				// Initialize as ready
				controller.mutex.Lock()
				controller.ready = true
				controller.mutex.Unlock()

				err := controller.Shutdown(context.Background())
				Expect(err).NotTo(HaveOccurred())

				// Should be marked as not ready after shutdown
				Expect(controller.IsReady()).To(BeFalse())

				conn.AssertExpectations(GinkgoT())
			})
		})

		Context("when shutdown encounters connection error", func() {
			It("should return an error", func() {
				conn.On("Close").Return(errors.New("close failed"))

				controller := NewStartupSequenceController(readiness, secret, conn, config).(*startupSequenceController)

				// Initialize as ready
				controller.mutex.Lock()
				controller.ready = true
				controller.mutex.Unlock()

				err := controller.Shutdown(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("close failed"))

				// Should be marked as not ready after shutdown
				Expect(controller.IsReady()).To(BeFalse())

				conn.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Integration", func() {
		It("should handle full lifecycle successfully", func() {
			// Setup successful initialization flow
			readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
			secret.On("GetValkeyPassword", mock.Anything).Return("test-password", nil)
			conn.On("InitializeCluster", mock.Anything, mock.AnythingOfType("string"), "test-password").Return(nil)
			conn.On("IsConnected").Return(true)
			conn.On("Close").Return(nil)

			// Initially not ready
			Expect(controller.IsReady()).To(BeFalse())

			// Initialize
			err := controller.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Should be ready after initialization
			Expect(controller.IsReady()).To(BeTrue())

			// Shutdown
			err = controller.Shutdown(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Should not be ready after shutdown
			Expect(controller.IsReady()).To(BeFalse())

			readiness.AssertExpectations(GinkgoT())
			secret.AssertExpectations(GinkgoT())
			conn.AssertExpectations(GinkgoT())
		})
	})

	Describe("SimplifiedFlow", func() {
		It("should handle simplified startup flow without secret watching", func() {
			// Setup mocks for the complete simplified flow
			readiness.On("WaitForReady", mock.Anything, 5*time.Minute).Return(nil)
			secret.On("GetValkeyPassword", mock.Anything).Return("test-password", nil)
			conn.On("InitializeCluster", mock.Anything, mock.AnythingOfType("string"), "test-password").Return(nil)
			conn.On("IsConnected").Return(true)

			// Initialize should succeed without secret watching
			err := controller.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Should be ready
			Expect(controller.IsReady()).To(BeTrue())

			// Verify that no secret watching was attempted (all expectations met)
			readiness.AssertExpectations(GinkgoT())
			secret.AssertExpectations(GinkgoT()) // Only GetValkeyPassword should have been called
			conn.AssertExpectations(GinkgoT())
		})
	})
})
