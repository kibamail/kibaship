package config

import (
	"fmt"

	"github.com/kibamail/kibaship/cmd/cli/internal/version"
)

// ComponentFile represents a manifest file for a component
type ComponentFile struct {
	Name string `json:"name"` // e.g., "manifest.yaml", "crds.yaml", "operator.yaml"
	URL  string `json:"url"`  // Full URL to the file
}

// ComponentVersion represents a specific version of a component for a provider
type ComponentVersion struct {
	Version string          `json:"version"`
	Files   []ComponentFile `json:"files"`
}

// ComponentProvider represents a provider configuration for a component
type ComponentProvider struct {
	Name     string                      `json:"name"`
	Versions map[string]ComponentVersion `json:"versions"`
}

// Component represents a Kubernetes component that can be installed
type Component struct {
	Name      string                       `json:"name"`
	Providers map[string]ComponentProvider `json:"providers"`
}

// ComponentsConfig holds the configuration for all available components
type ComponentsConfig struct {
	BaseURL    string               `json:"base_url"`
	Components map[string]Component `json:"components"`
}

// GetBaseURL returns the base URL for component manifests based on version
func GetBaseURL() string {
	if version.IsDevelopment() {
		return "https://raw.githubusercontent.com/kibamail/kibaship/refs/heads/main/" +
			"cmd/cli/commands/clusters/create/components"
	}
	v := version.GetVersion()
	return fmt.Sprintf("https://raw.githubusercontent.com/kibamail/kibaship/refs/tags/%s/"+
		"cmd/cli/commands/clusters/create/components", v)
}

// GetComponentsConfig returns the configuration for all components with dynamic versioning
func GetComponentsConfig() ComponentsConfig {
	baseURL := GetBaseURL()

	// Get dynamic versions
	certManagerVersion := version.GetComponentVersion("cert-manager")
	certManagerHetznerVersion := version.GetHetznerRobotComponentVersion("cert-manager")
	ciliumVersion := version.GetComponentVersion("cilium")
	ciliumHetznerVersion := version.GetHetznerRobotComponentVersion("cilium")
	longhornVersion := version.GetComponentVersion("longhorn-operator")
	mysqlVersion := version.GetComponentVersion("mysql-operator")
	valkeyVersion := version.GetComponentVersion("valkey-operator")
	acmeDnsVersion := version.GetComponentVersion("acme-dns")

	return ComponentsConfig{
		BaseURL: baseURL,
		Components: map[string]Component{
			"cert-manager": {
				Name: "cert-manager",
				Providers: map[string]ComponentProvider{
					"default": {
						Name: "default",
						Versions: map[string]ComponentVersion{
							certManagerVersion: {
								Version: certManagerVersion,
								Files: []ComponentFile{
									{
										Name: "manifest.yaml",
										URL:  fmt.Sprintf("%s/cert-manager/providers/default/versions/%s/manifest.yaml", baseURL, certManagerVersion),
									},
								},
							},
						},
					},
					"hetznerrobot": {
						Name: "hetznerrobot",
						Versions: map[string]ComponentVersion{
							certManagerHetznerVersion: {
								Version: certManagerHetznerVersion,
								Files: []ComponentFile{
									{
										Name: "manifest.yaml",
										URL: fmt.Sprintf("%s/cert-manager/providers/hetznerrobot/versions/%s/manifest.yaml",
											baseURL, certManagerHetznerVersion),
									},
								},
							},
						},
					},
				},
			},
			"cilium": {
				Name: "cilium",
				Providers: map[string]ComponentProvider{
					"default": {
						Name: "default",
						Versions: map[string]ComponentVersion{
							ciliumVersion: {
								Version: ciliumVersion,
								Files: []ComponentFile{
									{
										Name: "manifest.yaml",
										URL:  fmt.Sprintf("%s/cilium/providers/default/versions/%s/manifest.yaml", baseURL, ciliumVersion),
									},
								},
							},
						},
					},
					"hetznerrobot": {
						Name: "hetznerrobot",
						Versions: map[string]ComponentVersion{
							ciliumHetznerVersion: {
								Version: ciliumHetznerVersion,
								Files: []ComponentFile{
									{
										Name: "manifest.yaml",
										URL: fmt.Sprintf("%s/cilium/providers/hetznerrobot/versions/%s/manifest.yaml",
											baseURL, ciliumHetznerVersion),
									},
								},
							},
						},
					},
				},
			},
			"longhorn-operator": {
				Name: "longhorn-operator",
				Providers: map[string]ComponentProvider{
					"default": {
						Name: "default",
						Versions: map[string]ComponentVersion{
							longhornVersion: {
								Version: longhornVersion,
								Files:   []ComponentFile{
									// TODO: Add files when available
								},
							},
						},
					},
				},
			},
			"mysql-operator": {
				Name: "mysql-operator",
				Providers: map[string]ComponentProvider{
					"default": {
						Name: "default",
						Versions: map[string]ComponentVersion{
							mysqlVersion: {
								Version: mysqlVersion,
								Files:   []ComponentFile{
									// TODO: Add files when available
								},
							},
						},
					},
				},
			},
			"valkey-operator": {
				Name: "valkey-operator",
				Providers: map[string]ComponentProvider{
					"default": {
						Name: "default",
						Versions: map[string]ComponentVersion{
							valkeyVersion: {
								Version: valkeyVersion,
								Files:   []ComponentFile{
									// TODO: Add files when available
								},
							},
						},
					},
				},
			},
			"acme-dns": {
				Name: "acme-dns",
				Providers: map[string]ComponentProvider{
					"default": {
						Name: "default",
						Versions: map[string]ComponentVersion{
							acmeDnsVersion: {
								Version: acmeDnsVersion,
								Files:   []ComponentFile{
									// TODO: Add files when available
								},
							},
						},
					},
				},
			},
		},
	}
}

// GetComponentURL constructs the URL for a specific component file
func GetComponentURL(component, provider, componentVersion, filename string) string {
	baseURL := GetBaseURL()
	return fmt.Sprintf("%s/%s/providers/%s/versions/%s/%s", baseURL, component, provider, componentVersion, filename)
}
