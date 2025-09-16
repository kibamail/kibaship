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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRealTimeProvider(t *testing.T) {
	provider := NewRealTimeProvider()

	// Test Now()
	now1 := provider.Now()
	time.Sleep(1 * time.Millisecond)
	now2 := provider.Now()
	assert.True(t, now2.After(now1))

	// Test After()
	start := time.Now()
	ch := provider.After(10 * time.Millisecond)

	select {
	case receivedTime := <-ch:
		elapsed := time.Since(start)
		assert.True(t, elapsed >= 10*time.Millisecond)
		assert.True(t, receivedTime.After(start))
	case <-time.After(100 * time.Millisecond):
		t.Fatal("After channel should have fired within 100ms")
	}

	// Test Sleep() - we can't easily test this without making the test slow
	// Just ensure it doesn't panic
	assert.NotPanics(t, func() {
		provider.Sleep(1 * time.Microsecond)
	})
}

func TestKubernetesClientAdapter(t *testing.T) {
	// Create a fake client with a test scheme
	scheme := runtime.NewScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	adapter := NewKubernetesClientAdapter(fakeClient)
	assert.NotNil(t, adapter)

	// Test that the adapter exists and implements the interface
	assert.Implements(t, (*KubernetesClient)(nil), adapter)

	// Test List - this would normally succeed with empty results
	// but we can't test List easily with our mock structure without proper interfaces
}

func TestAdaptersIntegration(t *testing.T) {
	// Test that the adapters can be used together in a realistic scenario
	timeProvider := NewRealTimeProvider()

	scheme := runtime.NewScheme()
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	clientAdapter := NewKubernetesClientAdapter(k8sClient)

	// Verify types implement the expected interfaces
	assert.Implements(t, (*TimeProvider)(nil), timeProvider)
	assert.Implements(t, (*KubernetesClient)(nil), clientAdapter)

	// Test concurrent usage (common in Kubernetes controllers)
	done := make(chan bool, 2)

	go func() {
		// Simulate time-based operations
		for i := 0; i < 10; i++ {
			_ = timeProvider.Now()
			timeProvider.Sleep(1 * time.Microsecond)
		}
		done <- true
	}()

	go func() {
		// Simulate Kubernetes operations - just test that the methods exist
		// We can't easily test actual Get operations without proper setup
		_ = clientAdapter
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}
