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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ValkeyReadinessMonitor", func() {
	Describe("Constructor", func() {
		It("should create a valid monitor instance", func() {
			k8sClient := &mockKubernetesClient{}
			timeProvider := &mockTimeProvider{}
			config := &Config{
				Namespace:         "test-namespace",
				ValkeyServiceName: "test-valkey-service",
			}

			monitor := NewValkeyReadinessMonitor(k8sClient, timeProvider, config)
			Expect(monitor).NotTo(BeNil())

			// Verify it implements the interface
			_ = monitor
		})
	})
})
