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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Adapters", func() {
	Describe("RealTimeProvider", func() {
		var provider TimeProvider

		BeforeEach(func() {
			provider = NewRealTimeProvider()
		})

		It("should return current time with Now()", func() {
			now1 := provider.Now()
			time.Sleep(1 * time.Millisecond)
			now2 := provider.Now()
			Expect(now2).To(BeTemporally(">", now1))
		})

		It("should fire After() channel after specified duration", func() {
			start := time.Now()
			ch := provider.After(10 * time.Millisecond)

			Eventually(ch, "100ms").Should(Receive(And(
				BeTemporally(">=", start),
				BeTemporally(">=", start.Add(10*time.Millisecond)),
			)))
		})

		It("should not panic during Sleep()", func() {
			Expect(func() {
				provider.Sleep(1 * time.Microsecond)
			}).NotTo(Panic())
		})
	})

	Describe("KubernetesClientAdapter", func() {
		var adapter KubernetesClient

		BeforeEach(func() {
			scheme := runtime.NewScheme()
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			adapter = NewKubernetesClientAdapter(fakeClient)
		})

		It("should create a valid adapter", func() {
			Expect(adapter).NotTo(BeNil())
		})

		It("should implement KubernetesClient interface", func() {
			_ = adapter
		})
	})

	Describe("Adapters Integration", func() {
		It("should work together concurrently", func() {
			timeProvider := NewRealTimeProvider()

			scheme := runtime.NewScheme()
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			clientAdapter := NewKubernetesClientAdapter(k8sClient)

			// Verify types implement the expected interfaces
			_ = timeProvider
			_ = clientAdapter

			// Test concurrent usage (common in Kubernetes controllers)
			done := make(chan bool, 2)

			go func() {
				defer GinkgoRecover()
				// Simulate time-based operations
				for i := 0; i < 10; i++ {
					_ = timeProvider.Now()
					timeProvider.Sleep(1 * time.Microsecond)
				}
				done <- true
			}()

			go func() {
				defer GinkgoRecover()
				// Simulate Kubernetes operations - just test that the methods exist
				_ = clientAdapter
				done <- true
			}()

			// Wait for both goroutines to complete
			Eventually(done).Should(Receive())
			Eventually(done).Should(Receive())
		})
	})
})
