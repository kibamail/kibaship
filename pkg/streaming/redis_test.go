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

const (
	testStreamName = "test-stream"
	testPassword   = "test-password"
)

var _ = Describe("ValkeyClient", func() {
	var client ValkeyClient

	BeforeEach(func() {
		client, _ = NewTestValkeyClient("localhost:6379", "password", &Config{})
	})

	Describe("XAdd", func() {
		Context("when adding successfully", func() {
			It("should add entry to stream", func() {
				stream := testStreamName
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
				stream := testStreamName
				values := map[string]interface{}{}

				entryID, err := client.XAdd(context.Background(), stream, values)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("values cannot be empty"))
				Expect(entryID).To(BeEmpty())
			})
		})

		Context("with nil values", func() {
			It("should return an error", func() {
				stream := testStreamName
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
				client, _ := NewTestValkeyClient("localhost:6379", "password", &Config{})
				err := client.Ping(context.Background())
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with empty address", func() {
			It("should return an error", func() {
				client, _ := NewTestValkeyClient("", "password", &Config{})
				err := client.Ping(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("valkey address is not configured"))
			})
		})

		Context("with empty password", func() {
			It("should allow empty password", func() {
				client, _ := NewTestValkeyClient("localhost:6379", "", &Config{})
				err := client.Ping(context.Background())
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Close", func() {
		It("should close client and prevent further operations", func() {
			client, _ := NewTestValkeyClient("localhost:6379", "password", &Config{})

			err := client.Close()
			Expect(err).NotTo(HaveOccurred())

			// After closing, should not be able to ping
			err = client.Ping(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("valkey: client is closed"))
		})
	})

	Describe("NewValkeyClient", func() {
		It("should create a properly configured client", func() {
			address := "localhost:6379"
			password := testPassword

			client, _ := NewTestValkeyClient(address, password, &Config{})
			Expect(client).NotTo(BeNil())

			// Test that the client is properly configured
			err := client.Ping(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
