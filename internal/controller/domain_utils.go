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
)

const (
	// SubdomainCharset defines the characters allowed in generated subdomains
	SubdomainCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
	// SubdomainRandomSuffixLength defines the length of the random suffix
	SubdomainRandomSuffixLength = 8
	// MaxSubdomainAttempts defines the maximum number of attempts to generate a unique subdomain
	MaxSubdomainAttempts = 10
)

// GenerateSubdomain creates a unique subdomain for an application based on its name
// It extracts the app slug from the application name and adds a random suffix
// Example: project-myproject-app-frontend-kibaship-com -> frontend-abc12345
func GenerateSubdomain(appName string) (string, error) {
	// Extract app slug from application name
	appSlug := extractAppSlugFromName(appName)

	// Generate random suffix for uniqueness
	randomSuffix, err := generateRandomString(SubdomainRandomSuffixLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate random suffix: %v", err)
	}

	// Combine app slug with random suffix
	subdomain := fmt.Sprintf("%s-%s", appSlug, randomSuffix)

	// Ensure subdomain meets DNS requirements
	subdomain = sanitizeSubdomain(subdomain)

	return subdomain, nil
}

// extractAppSlugFromName extracts the application slug from the full application name
// Expected format: project-<project-name>-app-<app-slug>-kibaship-com
func extractAppSlugFromName(appName string) string {
	parts := strings.Split(appName, "-")

	// Find the app slug (part after "app-")
	const appDelimiter = "app"
	for i, part := range parts {
		if part == appDelimiter && i+1 < len(parts) {
			// Take the next part as the app slug
			appSlug := parts[i+1]

			// If there are more parts before "kibaship", include them
			// This handles cases like "api-gateway" in "project-name-app-api-gateway-kibaship-com"
			j := i + 2
			for j < len(parts) && parts[j] != "kibaship" {
				appSlug += "-" + parts[j]
				j++
			}

			return sanitizeAppSlug(appSlug)
		}
	}

	// Fallback: if no "app-" pattern found, use "app"
	return appDelimiter
}

// sanitizeAppSlug ensures the app slug is valid for DNS
func sanitizeAppSlug(slug string) string {
	// Convert to lowercase
	slug = strings.ToLower(slug)

	// Remove invalid characters, keep only alphanumeric and hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	cleaned := result.String()

	// Ensure doesn't start or end with hyphen
	cleaned = strings.Trim(cleaned, "-")

	// Ensure not empty
	if cleaned == "" {
		cleaned = "app"
	}

	// Limit length for subdomain constraints
	if len(cleaned) > 20 {
		cleaned = cleaned[:20]
		cleaned = strings.TrimSuffix(cleaned, "-")
	}

	return cleaned
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
func GenerateFullDomain(subdomain string) (string, error) {
	config, err := GetOperatorConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get operator configuration: %v", err)
	}

	fullDomain := fmt.Sprintf("%s.%s", subdomain, config.Domain)

	// Validate total domain length (DNS limit is 253 characters)
	if len(fullDomain) > 253 {
		return "", fmt.Errorf("generated domain %s exceeds maximum length of 253 characters", fullDomain)
	}

	return fullDomain, nil
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
