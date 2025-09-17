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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RedisClient", func() {
	var client RedisClient

	BeforeEach(func() {
		client = NewTestRedisClient("localhost:6379", "password")
	})

	Describe("XAdd", func() {
		Context("when adding successfully", func() {
			It("should add entry to stream", func() {
				stream := "test-stream"
				values := map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				}

				entryID, err := client.XAdd(context.Background(), stream, values)
				Expect(err).NotTo(HaveOccurred())
				Expect(entryID).NotTo(BeEmpty())
				// Test implementation returns a fixed entry ID
				Expect(entryID).To(Equal("1703123456789-0"))
			})
		})

		Context("with empty stream name", func() {
			It("should return an error", func() {
				stream := ""
				values := map[string]interface{}{"key": "value"}

				entryID, err := client.XAdd(context.Background(), stream, values)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("stream name cannot be empty"))
				Expect(entryID).To(BeEmpty())
			})
		})

		Context("with empty values", func() {
			It("should return an error", func() {
				stream := "test-stream"
				values := map[string]interface{}{}

				entryID, err := client.XAdd(context.Background(), stream, values)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("values cannot be empty"))
				Expect(entryID).To(BeEmpty())
			})
		})

		Context("with nil values", func() {
			It("should return an error", func() {
				stream := "test-stream"
				var values map[string]interface{}

				entryID, err := client.XAdd(context.Background(), stream, values)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("values cannot be empty"))
				Expect(entryID).To(BeEmpty())
			})
		})
	})

	Describe("Ping", func() {
		Context("when ping is successful", func() {
			It("should ping without errors", func() {
				client := NewTestRedisClient("localhost:6379", "password")
				err := client.Ping(context.Background())
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with empty address", func() {
			It("should return an error", func() {
				client := NewTestRedisClient("", "password")
				err := client.Ping(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Redis address is not configured"))
			})
		})

		Context("with empty password", func() {
			It("should return an error", func() {
				client := NewTestRedisClient("localhost:6379", "")
				err := client.Ping(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Redis password is not configured"))
			})
		})
	})

	Describe("Close", func() {
		It("should close client and prevent further operations", func() {
			client := NewTestRedisClient("localhost:6379", "password")

			err := client.Close()
			Expect(err).NotTo(HaveOccurred())

			// After closing, should not be able to ping
			err = client.Ping(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("redis: client is closed"))
		})
	})

	Describe("NewRedisClient", func() {
		It("should create a properly configured client", func() {
			address := "localhost:6379"
			password := "test-password"

			client := NewTestRedisClient(address, password)
			Expect(client).NotTo(BeNil())

			// Test that the client is properly configured
			err := client.Ping(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
