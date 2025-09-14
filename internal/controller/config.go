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
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
)

// OperatorConfig holds the global configuration for the operator
type OperatorConfig struct {
	// Domain is the base domain for all application subdomains
	Domain string
	// DefaultPort is the default port for applications
	DefaultPort int32
}

var (
	operatorConfig *OperatorConfig
	configOnce     sync.Once
	configError    error
)

// GetOperatorConfig returns the singleton operator configuration
// The configuration is loaded once from environment variables
func GetOperatorConfig() (*OperatorConfig, error) {
	configOnce.Do(func() {
		operatorConfig, configError = loadOperatorConfig()
	})

	if configError != nil {
		return nil, configError
	}

	return operatorConfig, nil
}

// loadOperatorConfig loads configuration from environment variables
func loadOperatorConfig() (*OperatorConfig, error) {
	domain := os.Getenv("KIBASHIP_OPERATOR_DOMAIN")
	if domain == "" {
		return nil, fmt.Errorf("KIBASHIP_OPERATOR_DOMAIN environment variable is required")
	}

	// Validate domain format - must be a valid DNS name
	domainRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
	if !domainRegex.MatchString(domain) {
		return nil, fmt.Errorf("invalid domain format: %s - domain must be a valid DNS name (lowercase, alphanumeric, hyphens, dots)", domain)
	}

	// Load default port with fallback
	defaultPortStr := os.Getenv("KIBASHIP_DEFAULT_PORT")
	if defaultPortStr == "" {
		defaultPortStr = "3000"
	}

	defaultPort, err := strconv.ParseInt(defaultPortStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid default port: %s - must be a valid integer", defaultPortStr)
	}

	if defaultPort < 1 || defaultPort > 65535 {
		return nil, fmt.Errorf("invalid default port: %d - must be between 1 and 65535", defaultPort)
	}

	return &OperatorConfig{
		Domain:      domain,
		DefaultPort: int32(defaultPort),
	}, nil
}

// ValidateOperatorConfig validates that the operator configuration is properly set
// This should be called during operator startup
func ValidateOperatorConfig() error {
	_, err := GetOperatorConfig()
	return err
}
