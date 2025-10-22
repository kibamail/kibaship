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
	"regexp"
	"sync"
)

// OperatorConfig holds the global configuration for the operator
type OperatorConfig struct {
	// Domain is the base domain for all application subdomains
	Domain string
	// DefaultPort is the default port for applications (hardcoded to 3000)
	DefaultPort int32
	// GatewayClassName is the Gateway API gateway class to use for routing
	GatewayClassName string
}

var (
	operatorConfig *OperatorConfig
	configOnce     sync.Once
)

// SetOperatorConfig sets the global operator configuration
// This should be called once at startup after loading from ConfigMap
func SetOperatorConfig(domain, gatewayClassName string) error {
	// Validate domain format - must be a valid DNS name
	domainRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("invalid domain format: %s - domain must be a valid DNS name (lowercase, alphanumeric, hyphens, dots)", domain)
	}

	// Validate gateway class name - must be non-empty
	if gatewayClassName == "" {
		return fmt.Errorf("gateway class name cannot be empty")
	}

	configOnce.Do(func() {
		operatorConfig = &OperatorConfig{
			Domain:           domain,
			DefaultPort:      3000, // Hardcoded to 3000
			GatewayClassName: gatewayClassName,
		}
	})

	return nil
}

// GetOperatorConfig returns the singleton operator configuration
func GetOperatorConfig() (*OperatorConfig, error) {
	if operatorConfig == nil {
		return nil, fmt.Errorf("operator configuration not initialized - call SetOperatorConfig first")
	}

	return operatorConfig, nil
}
