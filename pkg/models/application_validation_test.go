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

package models

import (
	"testing"
)

func TestValidateDockerfileBuild(t *testing.T) {
	tests := []struct {
		name          string
		config        *DockerfileBuildConfig
		expectErrors  bool
		errorContains string
	}{
		{
			name: "valid config with Dockerfile path",
			config: &DockerfileBuildConfig{
				DockerfilePath: "Dockerfile",
				BuildContext:   ".",
			},
			expectErrors: false,
		},
		{
			name: "valid config with nested Dockerfile path",
			config: &DockerfileBuildConfig{
				DockerfilePath: "app/Dockerfile",
				BuildContext:   "app",
			},
			expectErrors: false,
		},
		{
			name: "empty DockerfilePath",
			config: &DockerfileBuildConfig{
				DockerfilePath: "",
				BuildContext:   ".",
			},
			expectErrors:  true,
			errorContains: "DockerfilePath is required",
		},
		{
			name: "DockerfilePath with path traversal",
			config: &DockerfileBuildConfig{
				DockerfilePath: "../Dockerfile",
				BuildContext:   ".",
			},
			expectErrors:  true,
			errorContains: "cannot contain '..'",
		},
		{
			name: "DockerfilePath with absolute path",
			config: &DockerfileBuildConfig{
				DockerfilePath: "/etc/Dockerfile",
				BuildContext:   ".",
			},
			expectErrors:  true,
			errorContains: "must be a relative path",
		},
		{
			name: "BuildContext with path traversal",
			config: &DockerfileBuildConfig{
				DockerfilePath: "Dockerfile",
				BuildContext:   "../",
			},
			expectErrors:  true,
			errorContains: "cannot contain '..'",
		},
		{
			name: "BuildContext with absolute path",
			config: &DockerfileBuildConfig{
				DockerfilePath: "Dockerfile",
				BuildContext:   "/app",
			},
			expectErrors:  true,
			errorContains: "must be a relative path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateDockerfileBuild(tt.config)

			if tt.expectErrors && len(errors) == 0 {
				t.Errorf("expected errors but got none")
			}

			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("expected no errors but got: %v", errors)
			}

			if tt.expectErrors && tt.errorContains != "" {
				found := false
				for _, err := range errors {
					if contains(err.Message, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing '%s', got: %v", tt.errorContains, errors)
				}
			}
		})
	}
}

func TestValidateGitRepositoryWithBuildType(t *testing.T) {
	tests := []struct {
		name          string
		config        *GitRepositoryConfig
		expectErrors  bool
		errorContains string
	}{
		{
			name: "valid Dockerfile build type",
			config: &GitRepositoryConfig{
				Provider:     GitProviderGitHub,
				Repository:   "org/repo",
				PublicAccess: true,
				BuildType:    BuildTypeDockerfile,
				DockerfileBuild: &DockerfileBuildConfig{
					DockerfilePath: "Dockerfile",
					BuildContext:   ".",
				},
			},
			expectErrors: false,
		},
		{
			name: "valid Railpack build type",
			config: &GitRepositoryConfig{
				Provider:     GitProviderGitHub,
				Repository:   "org/repo",
				PublicAccess: true,
				BuildType:    BuildTypeRailpack,
			},
			expectErrors: false,
		},
		{
			name: "invalid build type",
			config: &GitRepositoryConfig{
				Provider:     GitProviderGitHub,
				Repository:   "org/repo",
				PublicAccess: true,
				BuildType:    "InvalidType",
			},
			expectErrors:  true,
			errorContains: "must be one of: Railpack, Dockerfile",
		},
		{
			name: "Dockerfile build type without config",
			config: &GitRepositoryConfig{
				Provider:     GitProviderGitHub,
				Repository:   "org/repo",
				PublicAccess: true,
				BuildType:    BuildTypeDockerfile,
			},
			expectErrors:  true,
			errorContains: "DockerfileBuild configuration is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateGitRepository(tt.config)

			if tt.expectErrors && len(errors) == 0 {
				t.Errorf("expected errors but got none")
			}

			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("expected no errors but got: %v", errors)
			}

			if tt.expectErrors && tt.errorContains != "" {
				found := false
				for _, err := range errors {
					if contains(err.Message, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing '%s', got: %v", tt.errorContains, errors)
				}
			}
		})
	}
}

func TestIsValidBuildType(t *testing.T) {
	tests := []struct {
		name      string
		buildType BuildType
		expected  bool
	}{
		{"valid Railpack", BuildTypeRailpack, true},
		{"valid Dockerfile", BuildTypeDockerfile, true},
		{"invalid empty", "", false},
		{"invalid type", "InvalidType", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBuildType(tt.buildType)
			if result != tt.expected {
				t.Errorf("isValidBuildType(%s) = %v, expected %v", tt.buildType, result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
