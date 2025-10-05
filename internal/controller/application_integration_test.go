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

package controller

import (
	"context"
	"testing"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/validation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplicationDomainCreation(t *testing.T) {
	// Set up operator configuration
	err := SetOperatorConfig("test.kibaship.com")
	if err != nil {
		t.Fatalf("Failed to set operator config: %v", err)
	}

	// Create a fake Kubernetes client
	scheme := runtime.NewScheme()
	_ = platformv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create test project first
	testProject := &platformv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: "test-namespace",
			Labels: map[string]string{
				validation.LabelResourceUUID: "550e8400-e29b-41d4-a716-446655440000",
				validation.LabelResourceSlug: "test-project",
			},
		},
		Spec: platformv1alpha1.ProjectSpec{},
	}

	// Create test environment
	testEnvironment := &platformv1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "environment-production-kibaship-com",
			Namespace: "test-namespace",
			Labels: map[string]string{
				validation.LabelResourceUUID: "env-550e8400-e29b-41d4-a716-446655440000",
				validation.LabelResourceSlug: "production",
				validation.LabelProjectUUID:  "550e8400-e29b-41d4-a716-446655440000",
			},
		},
		Spec: platformv1alpha1.EnvironmentSpec{
			ProjectRef: corev1.LocalObjectReference{Name: "test-project"},
		},
	}

	// Create test GitRepository application
	testApp := &platformv1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-test-project-app-frontend-kibaship-com",
			Namespace: "test-namespace",
			Labels: map[string]string{
				validation.LabelResourceUUID:    "550e8400-e29b-41d4-a716-446655440001",
				validation.LabelResourceSlug:    "frontend",
				validation.LabelProjectUUID:     "550e8400-e29b-41d4-a716-446655440000",
				validation.LabelEnvironmentUUID: "env-550e8400-e29b-41d4-a716-446655440000",
			},
		},
		Spec: platformv1alpha1.ApplicationSpec{
			Type: platformv1alpha1.ApplicationTypeGitRepository,
			EnvironmentRef: corev1.LocalObjectReference{
				Name: "environment-production-kibaship-com",
			},
			GitRepository: &platformv1alpha1.GitRepositoryConfig{
				Repository: "https://github.com/test/frontend",
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(testProject, testEnvironment, testApp).
		Build()

	// Create the reconciler
	reconciler := &ApplicationReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()

	// Test the handleApplicationDomains method
	err = reconciler.handleApplicationDomains(ctx, testApp)
	if err != nil {
		t.Fatalf("handleApplicationDomains failed: %v", err)
	}

	// Verify that an ApplicationDomain was created
	var domains platformv1alpha1.ApplicationDomainList
	err = client.List(ctx, &domains)
	if err != nil {
		t.Fatalf("Failed to list ApplicationDomains: %v", err)
	}

	if len(domains.Items) != 1 {
		t.Fatalf("Expected 1 ApplicationDomain, got %d", len(domains.Items))
	}

	domain := domains.Items[0]

	// Verify the ApplicationDomain properties
	if domain.Spec.ApplicationRef.Name != testApp.Name {
		t.Errorf("Expected ApplicationRef.Name to be %s, got %s", testApp.Name, domain.Spec.ApplicationRef.Name)
	}

	if domain.Spec.Type != platformv1alpha1.ApplicationDomainTypeDefault {
		t.Errorf("Expected Type to be %s, got %s", platformv1alpha1.ApplicationDomainTypeDefault, domain.Spec.Type)
	}

	if !domain.Spec.Default {
		t.Error("Expected Default to be true")
	}

	if domain.Spec.Port != 3000 {
		t.Errorf("Expected Port to be 3000, got %d", domain.Spec.Port)
	}

	if !domain.Spec.TLSEnabled {
		t.Error("Expected TLSEnabled to be true")
	}

	// Verify the domain follows the expected pattern
	if domain.Spec.Domain == "" {
		t.Error("Expected Domain to be set")
	}

	// Verify labels
	expectedLabels := map[string]string{
		ApplicationDomainLabelApplication: testApp.Name,
		ApplicationDomainLabelDomainType:  "default",
		validation.LabelApplicationUUID:   testApp.Labels[validation.LabelResourceUUID],
		validation.LabelProjectUUID:       testApp.Labels[validation.LabelProjectUUID],
	}

	for key, expectedValue := range expectedLabels {
		if actualValue, exists := domain.Labels[key]; !exists || actualValue != expectedValue {
			t.Errorf("Expected label %s to be %s, got %s", key, expectedValue, actualValue)
		}
	}

	// Verify the domain has its own unique UUID (different from application)
	domainUUID := domain.Labels[validation.LabelResourceUUID]
	if domainUUID == "" {
		t.Error("Expected ApplicationDomain to have its own UUID")
	}
	if domainUUID == testApp.Labels[validation.LabelResourceUUID] {
		t.Error("ApplicationDomain should have its own UUID, not the same as Application")
	}

	// Test that running it again doesn't create a duplicate
	err = reconciler.handleApplicationDomains(ctx, testApp)
	if err != nil {
		t.Fatalf("handleApplicationDomains failed on second run: %v", err)
	}

	// Verify still only one domain exists
	var domainsAfterSecondRun platformv1alpha1.ApplicationDomainList
	err = client.List(ctx, &domainsAfterSecondRun)
	if err != nil {
		t.Fatalf("Failed to list ApplicationDomains after second run: %v", err)
	}

	if len(domainsAfterSecondRun.Items) != 1 {
		t.Errorf("Expected 1 ApplicationDomain after second run, got %d", len(domainsAfterSecondRun.Items))
	}
}

func TestApplicationDomainSkipsNonGitRepository(t *testing.T) {
	// Set up operator configuration
	err := SetOperatorConfig("test.kibaship.com")
	if err != nil {
		t.Fatalf("Failed to set operator config: %v", err)
	}

	// Create a fake Kubernetes client
	scheme := runtime.NewScheme()
	_ = platformv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create test DockerImage application (not GitRepository)
	testApp := &platformv1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-test-project-app-backend-kibaship-com",
			Namespace: "test-namespace",
		},
		Spec: platformv1alpha1.ApplicationSpec{
			Type: platformv1alpha1.ApplicationTypeDockerImage, // Not GitRepository
			DockerImage: &platformv1alpha1.DockerImageConfig{
				Image: "nginx:latest",
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(testApp).
		Build()

	// Create the reconciler
	reconciler := &ApplicationReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()

	// Test the handleApplicationDomains method
	err = reconciler.handleApplicationDomains(ctx, testApp)
	if err != nil {
		t.Fatalf("handleApplicationDomains failed: %v", err)
	}

	// Verify that no ApplicationDomain was created
	var domains platformv1alpha1.ApplicationDomainList
	err = client.List(ctx, &domains)
	if err != nil {
		t.Fatalf("Failed to list ApplicationDomains: %v", err)
	}

	if len(domains.Items) != 0 {
		t.Fatalf("Expected 0 ApplicationDomains for non-GitRepository application, got %d", len(domains.Items))
	}
}
