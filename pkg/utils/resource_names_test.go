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

package utils

import (
	"testing"
)

func TestGetProjectResourceName(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	expected := "project-550e8400-e29b-41d4-a716-446655440000"
	result := GetProjectResourceName(uuid)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetEnvironmentResourceName(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440001"
	expected := "environment-550e8400-e29b-41d4-a716-446655440001"
	result := GetEnvironmentResourceName(uuid)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetApplicationResourceName(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440002"
	expected := "application-550e8400-e29b-41d4-a716-446655440002"
	result := GetApplicationResourceName(uuid)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetDeploymentResourceName(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440003"
	expected := "deployment-550e8400-e29b-41d4-a716-446655440003"
	result := GetDeploymentResourceName(uuid)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetApplicationDomainResourceName(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440004"
	expected := "domain-550e8400-e29b-41d4-a716-446655440004"
	result := GetApplicationDomainResourceName(uuid)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetValkeyResourceName(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440007"
	expected := "valkey-550e8400-e29b-41d4-a716-446655440007"
	result := GetValkeyResourceName(uuid)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TODO: Database resource name tests removed - will be reimplemented
func TestGetValkeyClusterResourceName(t *testing.T) {
	t.Skip("Valkey cluster resource name test removed - TODO: implement new test")
}

func TestGetMySQLResourceName(t *testing.T) {
	t.Skip("MySQL resource name test removed - TODO: implement new test")
}

func TestGetMySQLClusterResourceName(t *testing.T) {
	slug := "0f240b15edba7750862e14"
	expected := "mc-0f240b15edba7750862e14"
	result := GetMySQLClusterResourceName(slug)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetServiceName(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440009"
	expected := "service-550e8400-e29b-41d4-a716-446655440009"
	result := GetServiceName(uuid)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetKubernetesDeploymentName(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440010"
	expected := "deployment-550e8400-e29b-41d4-a716-446655440010"
	result := GetKubernetesDeploymentName(uuid)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}
