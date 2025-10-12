package config

import (
	"fmt"
	"os"
	"testing"
)

// Example demonstrating how the version system works
func ExampleGetComponentsConfig() {
	// Test development mode
	_ = os.Setenv("KIBASHIP_CLI_VERSION", "dev")
	config := GetComponentsConfig()
	fmt.Printf("Development mode base URL: %s\n", config.BaseURL)

	// Test production mode
	_ = os.Setenv("KIBASHIP_CLI_VERSION", "v1.2.0")
	config = GetComponentsConfig()
	fmt.Printf("Production mode base URL: %s\n", config.BaseURL)

	// Clean up
	_ = os.Unsetenv("KIBASHIP_CLI_VERSION")

	// Output:
	// Development mode base URL: https://raw.githubusercontent.com/kibamail/kibaship/refs/heads/main/
	//   cmd/cli/commands/clusters/create/components
	// Production mode base URL: https://raw.githubusercontent.com/kibamail/kibaship/refs/tags/v1.2.0/
	//   cmd/cli/commands/clusters/create/components
}

func TestVersionBehavior(t *testing.T) {
	// Save original env
	originalEnv := os.Getenv("KIBASHIP_CLI_VERSION")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("KIBASHIP_CLI_VERSION", originalEnv)
		} else {
			_ = os.Unsetenv("KIBASHIP_CLI_VERSION")
		}
	}()

	// Test development mode
	_ = os.Setenv("KIBASHIP_CLI_VERSION", "dev")
	baseURL := GetBaseURL()
	expectedDev := "https://raw.githubusercontent.com/kibamail/kibaship/refs/heads/main/" +
		"cmd/cli/commands/clusters/create/components"
	if baseURL != expectedDev {
		t.Errorf("Expected dev URL %s, got %s", expectedDev, baseURL)
	}

	// Test production mode
	_ = os.Setenv("KIBASHIP_CLI_VERSION", "v1.5.0")
	baseURL = GetBaseURL()
	expectedProd := "https://raw.githubusercontent.com/kibamail/kibaship/refs/tags/v1.5.0/" +
		"cmd/cli/commands/clusters/create/components"
	if baseURL != expectedProd {
		t.Errorf("Expected prod URL %s, got %s", expectedProd, baseURL)
	}
}
