package create

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the nested YAML configuration structure
type YAMLConfig struct {
	State struct {
		S3 struct {
			Bucket       string `yaml:"bucket"`
			Region       string `yaml:"region"`
			AccessKey    string `yaml:"access-key"`
			AccessSecret string `yaml:"access-secret"`
		} `yaml:"s3"`
	} `yaml:"state"`

	Cluster struct {
		Domain       string `yaml:"domain"`
		Email        string `yaml:"email"`
		PaaSFeatures string `yaml:"paas-features"`

		Provider struct {
			AWS struct {
				AccessKeyID     string `yaml:"access-key-id"`
				SecretAccessKey string `yaml:"secret-access-key"`
				Region          string `yaml:"region"`
			} `yaml:"aws"`

			DigitalOcean struct {
				Token     string `yaml:"token"`
				Nodes     string `yaml:"nodes"`
				NodesSize string `yaml:"nodes-size"`
				Region    string `yaml:"region"`
			} `yaml:"digital-ocean"`

			Hetzner struct {
				Token string `yaml:"token"`
			} `yaml:"hetzner"`

			HetznerRobot struct {
				Username   string `yaml:"username"`
				Password   string `yaml:"password"`
				CloudToken string `yaml:"cloud-token"`
			} `yaml:"hetzner-robot"`

			Linode struct {
				Token string `yaml:"token"`
			} `yaml:"linode"`

			GCloud struct {
				ServiceAccountKey string `yaml:"service-account-key"`
				ProjectID         string `yaml:"project-id"`
				Region            string `yaml:"region"`
			} `yaml:"gcloud"`

			Kind struct {
				Nodes string `yaml:"nodes"`
			} `yaml:"kind"`
		} `yaml:"provider"`
	} `yaml:"cluster"`
}

// LoadConfigFromYAML loads and parses a YAML configuration file
func LoadConfigFromYAML(filePath string) (*CreateConfig, error) {
	// Read the YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse the YAML
	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	// Convert to CreateConfig
	config, err := convertYAMLToCreateConfig(&yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML configuration: %w", err)
	}

	// Set the configuration file path
	config.Configuration = filePath

	return config, nil
}

// convertYAMLToCreateConfig converts YAMLConfig to CreateConfig
func convertYAMLToCreateConfig(yamlConfig *YAMLConfig) (*CreateConfig, error) {
	config := &CreateConfig{
		Domain:       yamlConfig.Cluster.Domain,
		Email:        yamlConfig.Cluster.Email,
		PaaSFeatures: yamlConfig.Cluster.PaaSFeatures,
	}

	// Derive cluster name from domain
	// For Kind clusters, use domain as-is; for others, replace dots with dashes
	if config.Domain != "" {
		// Determine provider first to decide naming strategy
		provider, _, err := determineProviderFromYAML(yamlConfig)
		if err != nil {
			return nil, err
		}

		if provider == "kind" {
			// For Kind clusters, use domain as-is
			config.Name = config.Domain
		} else {
			// For cloud providers, replace dots with dashes
			config.Name = strings.ReplaceAll(config.Domain, ".", "-")
		}
	}

	// Set default PaaS features if not specified
	if config.PaaSFeatures == "" {
		config.PaaSFeatures = "mysql,valkey,postgres"
	}

	// Determine the provider based on which provider section has data
	provider, providerConfig, err := determineProviderFromYAML(yamlConfig)
	if err != nil {
		return nil, err
	}

	config.Provider = provider

	// Set provider-specific configuration
	switch provider {
	case "aws":
		config.AWS = providerConfig.(*AWSConfig)
	case "digital-ocean":
		config.DigitalOcean = providerConfig.(*DigitalOceanConfig)
	case "hetzner":
		config.Hetzner = providerConfig.(*HetznerConfig)
	case "hetzner-robot":
		config.HetznerRobot = providerConfig.(*HetznerRobotConfig)
	case "linode":
		config.Linode = providerConfig.(*LinodeConfig)
	case "gcloud":
		config.GCloud = providerConfig.(*GCloudConfig)
	case "kind":
		config.Kind = providerConfig.(*KindConfig)
	}

	// Set Terraform state configuration (only for cloud providers)
	if provider != "kind" {
		config.TerraformState = &TerraformStateConfig{
			S3Bucket:       yamlConfig.State.S3.Bucket,
			S3Region:       yamlConfig.State.S3.Region,
			S3AccessKey:    yamlConfig.State.S3.AccessKey,
			S3AccessSecret: yamlConfig.State.S3.AccessSecret,
		}
	} else {
		// Kind clusters use local state - no S3 configuration needed
		config.TerraformState = &TerraformStateConfig{}
	}

	return config, nil
}

// determineProviderFromYAML determines which provider is configured and returns its config
func determineProviderFromYAML(yamlConfig *YAMLConfig) (string, interface{}, error) {
	providers := []struct {
		name      string
		hasData   func() bool
		getConfig func() interface{}
	}{
		{
			name: "aws",
			hasData: func() bool {
				return yamlConfig.Cluster.Provider.AWS.AccessKeyID != "" ||
					yamlConfig.Cluster.Provider.AWS.SecretAccessKey != "" ||
					yamlConfig.Cluster.Provider.AWS.Region != ""
			},
			getConfig: func() interface{} {
				return &AWSConfig{
					AccessKeyID:     yamlConfig.Cluster.Provider.AWS.AccessKeyID,
					SecretAccessKey: yamlConfig.Cluster.Provider.AWS.SecretAccessKey,
					Region:          yamlConfig.Cluster.Provider.AWS.Region,
				}
			},
		},
		{
			name: "digital-ocean",
			hasData: func() bool {
				return yamlConfig.Cluster.Provider.DigitalOcean.Token != "" ||
					yamlConfig.Cluster.Provider.DigitalOcean.Nodes != "" ||
					yamlConfig.Cluster.Provider.DigitalOcean.NodesSize != "" ||
					yamlConfig.Cluster.Provider.DigitalOcean.Region != ""
			},
			getConfig: func() interface{} {
				return &DigitalOceanConfig{
					Token:     yamlConfig.Cluster.Provider.DigitalOcean.Token,
					Nodes:     yamlConfig.Cluster.Provider.DigitalOcean.Nodes,
					NodesSize: yamlConfig.Cluster.Provider.DigitalOcean.NodesSize,
					Region:    yamlConfig.Cluster.Provider.DigitalOcean.Region,
				}
			},
		},
		{
			name: "hetzner",
			hasData: func() bool {
				return yamlConfig.Cluster.Provider.Hetzner.Token != ""
			},
			getConfig: func() interface{} {
				return &HetznerConfig{
					Token: yamlConfig.Cluster.Provider.Hetzner.Token,
				}
			},
		},
		{
			name: "hetzner-robot",
			hasData: func() bool {
				return yamlConfig.Cluster.Provider.HetznerRobot.Username != "" ||
					yamlConfig.Cluster.Provider.HetznerRobot.Password != "" ||
					yamlConfig.Cluster.Provider.HetznerRobot.CloudToken != ""
			},
			getConfig: func() interface{} {
				return &HetznerRobotConfig{
					Username:   yamlConfig.Cluster.Provider.HetznerRobot.Username,
					Password:   yamlConfig.Cluster.Provider.HetznerRobot.Password,
					CloudToken: yamlConfig.Cluster.Provider.HetznerRobot.CloudToken,
				}
			},
		},
		{
			name: "linode",
			hasData: func() bool {
				return yamlConfig.Cluster.Provider.Linode.Token != ""
			},
			getConfig: func() interface{} {
				return &LinodeConfig{
					Token: yamlConfig.Cluster.Provider.Linode.Token,
				}
			},
		},
		{
			name: "gcloud",
			hasData: func() bool {
				return yamlConfig.Cluster.Provider.GCloud.ServiceAccountKey != "" ||
					yamlConfig.Cluster.Provider.GCloud.ProjectID != "" ||
					yamlConfig.Cluster.Provider.GCloud.Region != ""
			},
			getConfig: func() interface{} {
				return &GCloudConfig{
					ServiceAccountKey: yamlConfig.Cluster.Provider.GCloud.ServiceAccountKey,
					ProjectID:         yamlConfig.Cluster.Provider.GCloud.ProjectID,
					Region:            yamlConfig.Cluster.Provider.GCloud.Region,
				}
			},
		},
		{
			name: "kind",
			hasData: func() bool {
				return yamlConfig.Cluster.Provider.Kind.Nodes != ""
			},
			getConfig: func() interface{} {
				return &KindConfig{
					Nodes: yamlConfig.Cluster.Provider.Kind.Nodes,
				}
			},
		},
	}

	// Find which provider has configuration data
	var configuredProviders []string
	var selectedProvider string
	var selectedConfig interface{}

	for _, provider := range providers {
		if provider.hasData() {
			configuredProviders = append(configuredProviders, provider.name)
			selectedProvider = provider.name
			selectedConfig = provider.getConfig()
		}
	}

	// Validate that exactly one provider is configured
	if len(configuredProviders) == 0 {
		return "", nil, fmt.Errorf("no provider configuration found in YAML file")
	}

	if len(configuredProviders) > 1 {
		return "", nil, fmt.Errorf("multiple providers configured in YAML file: %v. Please configure only one provider", configuredProviders)
	}

	return selectedProvider, selectedConfig, nil
}
