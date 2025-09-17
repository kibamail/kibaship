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
	"strings"
	"testing"
)

func TestValidateDomainFormat(t *testing.T) {
	tests := []struct {
		name      string
		domain    string
		wantError bool
	}{
		{
			name:      "valid domain",
			domain:    "frontend-abc123.myapps.kibaship.com",
			wantError: false,
		},
		{
			name:      "valid subdomain",
			domain:    "app.example.com",
			wantError: false,
		},
		{
			name:      "empty domain",
			domain:    "",
			wantError: true,
		},
		{
			name:      "domain with uppercase",
			domain:    "Frontend-ABC123.myapps.kibaship.com",
			wantError: true,
		},
		{
			name:      "domain starting with hyphen",
			domain:    "-frontend.myapps.kibaship.com",
			wantError: true,
		},
		{
			name:      "domain ending with hyphen",
			domain:    "frontend-.myapps.kibaship.com",
			wantError: true,
		},
		{
			name:      "domain with double dots",
			domain:    "frontend..myapps.kibaship.com",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomainFormat(tt.domain)
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("ValidateDomainFormat(%s) error = %v; wantError = %v", tt.domain, err, tt.wantError)
			}
		})
	}
}

func TestGenerateSubdomain(t *testing.T) {
	tests := []struct {
		name    string
		appName string
	}{
		{
			name:    "frontend app",
			appName: "project-myproject-app-frontend-kibaship-com",
		},
		{
			name:    "api gateway app",
			appName: "project-myproject-app-api-gateway-kibaship-com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subdomain, err := GenerateSubdomain(tt.appName)
			if err != nil {
				t.Errorf("GenerateSubdomain(%s) returned error: %v", tt.appName, err)
				return
			}

			// Check that subdomain is not empty
			if subdomain == "" {
				t.Errorf("GenerateSubdomain(%s) returned empty subdomain", tt.appName)
				return
			}

			// Check that subdomain follows expected pattern (contains hyphen for random suffix)
			if !strings.Contains(subdomain, "-") {
				t.Errorf("GenerateSubdomain(%s) = %s; expected to contain hyphen for random suffix", tt.appName, subdomain)
				return
			}

			// Check that subdomain is valid DNS format
			if err := ValidateDomainFormat(subdomain + ".example.com"); err != nil {
				t.Errorf("GenerateSubdomain(%s) = %s; generated subdomain creates invalid domain: %v", tt.appName, subdomain, err)
			}

			// Check length constraints
			if len(subdomain) > 60 {
				t.Errorf("GenerateSubdomain(%s) = %s; subdomain too long (%d > 60)", tt.appName, subdomain, len(subdomain))
			}

			// Check that it only contains allowed characters (apply De Morgan's law)
			for _, r := range subdomain {
				if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
					t.Errorf("GenerateSubdomain(%s) = %s; contains invalid character '%c'", tt.appName, subdomain, r)
					break
				}
			}
		})
	}
}

func TestGenerateApplicationDomainName(t *testing.T) {
	tests := []struct {
		name       string
		appName    string
		domainType string
		expected   string
	}{
		{
			name:       "default domain",
			appName:    "project-myproject-app-frontend-kibaship-com",
			domainType: "default",
			expected:   "project-myproject-app-frontend-kibaship-com-default",
		},
		{
			name:       "custom domain",
			appName:    "project-myproject-app-frontend-kibaship-com",
			domainType: "custom",
			expected:   "", // Variable due to random suffix, we'll check the pattern
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateApplicationDomainName(tt.appName, tt.domainType)

			switch tt.domainType {
			case "default":
				if result != tt.expected {
					t.Errorf("GenerateApplicationDomainName(%s, %s) = %s; expected %s", tt.appName, tt.domainType, result, tt.expected)
				}
			case "custom":
				// Check that it follows the pattern for custom domains
				expectedPrefix := tt.appName + "-custom-"
				if !strings.HasPrefix(result, expectedPrefix) {
					t.Errorf("GenerateApplicationDomainName(%s, %s) = %s; expected to start with %s", tt.appName, tt.domainType, result, expectedPrefix)
				}
				// Check that it has a suffix after the prefix
				suffix := strings.TrimPrefix(result, expectedPrefix)
				if len(suffix) == 0 {
					t.Errorf("GenerateApplicationDomainName(%s, %s) = %s; expected to have random suffix", tt.appName, tt.domainType, result)
				}
			}
		})
	}
}
