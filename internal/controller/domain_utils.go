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
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

const (
	// SubdomainCharset defines the characters allowed in generated subdomains
	SubdomainCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
	// SubdomainRandomSuffixLength defines the length of the random suffix
	SubdomainRandomSuffixLength = 8
	// MaxSubdomainAttempts defines the maximum number of attempts to generate a unique subdomain
	MaxSubdomainAttempts = 10
)

// GenerateSubdomain creates a unique subdomain for an application based on its UUID
// It takes the app UUID directly and adds a random suffix
// Example: 123e4567-e89b-12d3-a456-426614174000 -> 123e4567e89b12d3-abc12345
func GenerateSubdomain(appUUID string) (string, error) {
	// Generate random suffix for uniqueness
	randomSuffix, err := generateRandomString(SubdomainRandomSuffixLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate random suffix: %v", err)
	}

	// Remove hyphens from UUID to make it shorter and more DNS-friendly
	// Take first 16 characters for brevity (still unique enough)
	cleanUUID := strings.ReplaceAll(appUUID, "-", "")
	if len(cleanUUID) > 16 {
		cleanUUID = cleanUUID[:16]
	}

	// Combine app UUID with random suffix
	subdomain := fmt.Sprintf("%s-%s", cleanUUID, randomSuffix)

	// Ensure subdomain meets DNS requirements
	subdomain = sanitizeSubdomain(subdomain)

	return subdomain, nil
}

// sanitizeSubdomain ensures the generated subdomain is valid for DNS
func sanitizeSubdomain(subdomain string) string {
	// Ensure it starts with alphanumeric
	if len(subdomain) > 0 && !isAlphanumeric(subdomain[0]) {
		subdomain = "a" + subdomain
	}

	// Ensure it ends with alphanumeric
	if len(subdomain) > 0 && !isAlphanumeric(subdomain[len(subdomain)-1]) {
		subdomain = subdomain + "a"
	}

	// Limit total length (DNS label limit is 63 characters)
	if len(subdomain) > 60 {
		subdomain = subdomain[:60]
		subdomain = strings.TrimSuffix(subdomain, "-")
	}

	return subdomain
}

// generateRandomString generates a cryptographically secure random string
// using the specified character set and length
func generateRandomString(length int) (string, error) {
	charset := SubdomainCharset
	charsetLen := big.NewInt(int64(len(charset)))

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %v", err)
		}
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result), nil
}

// GenerateFullDomain creates the full domain name by combining subdomain with operator domain
// For web applications, this produces domains in the format: <subdomain>.apps.<baseDomain>
func GenerateFullDomain(subdomain string) (string, error) {
	config, err := GetOperatorConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get operator configuration: %v", err)
	}

	fullDomain := fmt.Sprintf("%s.apps.%s", subdomain, config.Domain)

	// Validate total domain length (DNS limit is 253 characters)
	if len(fullDomain) > 253 {
		return "", fmt.Errorf("generated domain %s exceeds maximum length of 253 characters", fullDomain)
	}

	return fullDomain, nil
}

// GenerateFullDomainForApplicationType creates the full domain name based on application type
func GenerateFullDomainForApplicationType(subdomain string, appType platformv1alpha1.ApplicationType) (string, int32, error) {
	config, err := GetOperatorConfig()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get operator configuration: %v", err)
	}

	var fullDomain string
	var port int32

	switch appType {
	case platformv1alpha1.ApplicationTypeGitRepository, platformv1alpha1.ApplicationTypeDockerImage:
		fullDomain = fmt.Sprintf("%s.apps.%s", subdomain, config.Domain)
		port = 3000
	case platformv1alpha1.ApplicationTypeValkey, platformv1alpha1.ApplicationTypeValkeyCluster:
		fullDomain = fmt.Sprintf("%s.valkey.%s", subdomain, config.Domain)
		port = 6379
	case platformv1alpha1.ApplicationTypeMySQL, platformv1alpha1.ApplicationTypeMySQLCluster:
		fullDomain = fmt.Sprintf("%s.mysql.%s", subdomain, config.Domain)
		port = 3306
	case platformv1alpha1.ApplicationTypePostgres, platformv1alpha1.ApplicationTypePostgresCluster:
		fullDomain = fmt.Sprintf("%s.postgres.%s", subdomain, config.Domain)
		port = 5432
	default:
		return "", 0, fmt.Errorf("unsupported application type for domain generation: %s", appType)
	}

	// Validate total domain length (DNS limit is 253 characters)
	if len(fullDomain) > 253 {
		return "", 0, fmt.Errorf("generated domain %s exceeds maximum length of 253 characters", fullDomain)
	}

	return fullDomain, port, nil
}

// isAlphanumeric checks if a character is alphanumeric
func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// ValidateDomainFormat validates that a domain follows DNS naming rules
func ValidateDomainFormat(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if len(domain) > 253 {
		return fmt.Errorf("domain exceeds maximum length of 253 characters")
	}

	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 {
			return fmt.Errorf("domain cannot have empty labels")
		}
		if len(label) > 63 {
			return fmt.Errorf("domain label '%s' exceeds maximum length of 63 characters", label)
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("domain label '%s' cannot start or end with hyphen", label)
		}
		// Check for valid characters (apply De Morgan's law)
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return fmt.Errorf("domain label '%s' contains invalid character '%c'", label, r)
			}
		}
	}

	return nil
}

// GenerateApplicationDomainName generates a unique name for an ApplicationDomain resource
// Format: <app-name>-<type>-<suffix>
func GenerateApplicationDomainName(appName string, domainType string) string {
	// For default domains, use simple suffix
	if domainType == "default" {
		return fmt.Sprintf("%s-default", appName)
	}

	// For custom domains, include a short random suffix
	randomSuffix, err := generateRandomString(6)
	if err != nil {
		// Fallback to timestamp-based suffix if random generation fails
		randomSuffix = "custom"
	}

	return fmt.Sprintf("%s-custom-%s", appName, randomSuffix)
}
