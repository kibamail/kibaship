package version

import "os"

const (
	// LatestVersion is the fallback version when component version is not found
	LatestVersion = "latest"
)

// Version represents the CLI version
const DefaultVersion = "v1.0.0" // This should match the release tag

// ComponentVersions defines the default versions for all components
var ComponentVersions = map[string]string{
	"cilium":            "1.18.2",
	"cert-manager":      "1.18.2",
	"longhorn-operator": "1.10.0",
	"mysql-operator":    "9.4.0-2.2.5",
	"valkey-operator":   "1.0.0", // TODO: Set actual default version
	"acme-dns":          "1.0.0", // TODO: Set actual default version
}

// HetznerRobotComponentVersions defines the default versions for hetznerrobot provider components
var HetznerRobotComponentVersions = map[string]string{
	"cilium":       "v1.18.2",
	"cert-manager": "v1.18.2",
}

// GetVersion returns the CLI version, checking environment variable first
func GetVersion() string {
	if envVersion := os.Getenv("KIBASHIP_CLI_VERSION"); envVersion != "" {
		return envVersion
	}
	return DefaultVersion
}

// IsDevelopment returns true if the version is set to "dev"
func IsDevelopment() bool {
	return GetVersion() == "dev"
}

// GetComponentVersion returns the default version for a given component
func GetComponentVersion(component string) string {
	if version, exists := ComponentVersions[component]; exists {
		return version
	}
	return LatestVersion // fallback
}

// GetHetznerRobotComponentVersion returns the default version for a hetznerrobot provider component
func GetHetznerRobotComponentVersion(component string) string {
	if version, exists := HetznerRobotComponentVersions[component]; exists {
		return version
	}
	// Fallback to regular component version with "v" prefix if not found
	if version := GetComponentVersion(component); version != LatestVersion {
		return "v" + version
	}
	return LatestVersion
}
