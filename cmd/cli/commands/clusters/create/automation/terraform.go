package automation

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/config"
)

//go:embed terraform/providers/*
var terraformTemplates embed.FS

// ListEmbeddedTemplates lists all embedded template files (for debugging)
func ListEmbeddedTemplates() error {
	fmt.Println("Embedded template files:")
	return fs.WalkDir(terraformTemplates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fmt.Printf("  %s\n", path)
		}
		return nil
	})
}

// buildTerraformFiles creates the directory structure and compiles Terraform templates
func buildTerraformFiles(config *config.CreateConfig) error {
	return BuildTerraformFilesForConfig(config)
}

// BuildTerraformFilesForConfig creates the directory structure and compiles Terraform templates
// (exported for delete command)
func BuildTerraformFilesForConfig(config *config.CreateConfig) error {
	// Create .kibaship directory structure
	kibashipDir := ".kibaship"
	clusterDir := filepath.Join(kibashipDir, config.Name)
	provisionDir := filepath.Join(clusterDir, "provision")
	bootstrapDir := filepath.Join(clusterDir, "bootstrap")

	// Create all directories
	dirs := []string{kibashipDir, clusterDir, provisionDir, bootstrapDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Get provider-specific template path
	providerPath := fmt.Sprintf("terraform/providers/%s", config.Provider)

	// Compile provision templates
	if err := compileTemplate(providerPath, "provision.tf.tpl",
		filepath.Join(provisionDir, "main.tf"), config); err != nil {
		return fmt.Errorf("failed to compile provision template: %w", err)
	}

	if err := compileTemplate(providerPath, "vars.tf.tpl", filepath.Join(provisionDir, "vars.tf"), config); err != nil {
		return fmt.Errorf("failed to compile provision vars template: %w", err)
	}

	// Compile bootstrap templates
	if err := compileTemplate(providerPath, "bootstrap.tf.tpl",
		filepath.Join(bootstrapDir, "main.tf"), config); err != nil {
		return fmt.Errorf("failed to compile bootstrap template: %w", err)
	}

	if err := compileTemplate(providerPath, "vars.tf.tpl", filepath.Join(bootstrapDir, "vars.tf"), config); err != nil {
		return fmt.Errorf("failed to compile bootstrap vars template: %w", err)
	}

	return nil
}

// BuildHetznerRobotProvisionFiles creates the provision directory and compiles provision.tf.tpl
// Note: provision.tf.tpl declares variables inline, so no separate vars file is needed
func BuildHetznerRobotProvisionFiles(config *config.CreateConfig) error {
	// Create .kibaship directory structure
	kibashipDir := ".kibaship"
	clusterDir := filepath.Join(kibashipDir, config.Name)
	provisionDir := filepath.Join(clusterDir, "provision")

	// Create directories
	dirs := []string{kibashipDir, clusterDir, provisionDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Log template variables being passed
	fmt.Printf("\n%s %s\n",
		"\033[36m📋\033[0m",
		"\033[1;36mTemplate Variables for Provision Build:\033[0m")
	fmt.Printf("  %s: %s\n", "\033[90mName\033[0m", config.Name)

	if config.HetznerRobot != nil {
		fmt.Printf("  %s: %d server(s)\n", "\033[90mHetznerRobot.SelectedServers\033[0m", len(config.HetznerRobot.SelectedServers))
		for i, server := range config.HetznerRobot.SelectedServers {
			fmt.Printf("    %s [%d]:\n", "\033[90mServer\033[0m", i)
			fmt.Printf("      %s: %s\n", "\033[90mID\033[0m", server.ID)
			fmt.Printf("      %s: %s\n", "\033[90mName\033[0m", server.Name)
			fmt.Printf("      %s: %s\n", "\033[90mIP\033[0m", server.IP)
			fmt.Printf("      %s: %s\n", "\033[90mRole\033[0m", server.Role)
			fmt.Printf("      %s: %s\n", "\033[90mRescuePassword\033[0m", maskSensitive(server.RescuePassword))
		}
	}

	// Get provider-specific template path
	providerPath := "terraform/providers/hetzner-robot"

	// Compile provision template (includes variable declarations inline)
	if err := compileTemplate(providerPath, "provision.tf.tpl",
		filepath.Join(provisionDir, "main.tf"), config); err != nil {
		return fmt.Errorf("failed to compile provision template: %w", err)
	}

	return nil
}

// Cloud build removed for hetzner-robot (no longer used)

// BuildHetznerRobotBootstrapFiles creates the bootstrap directory and compiles bootstrap.tf.tpl and vars.bootstrap.tf.tpl
func BuildHetznerRobotBootstrapFiles(config *config.CreateConfig) error {
	// Create .kibaship directory structure
	kibashipDir := ".kibaship"
	clusterDir := filepath.Join(kibashipDir, config.Name)
	bootstrapDir := filepath.Join(clusterDir, "bootstrap")

	// Create directories
	dirs := []string{kibashipDir, clusterDir, bootstrapDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Log template variables being passed
	fmt.Printf("\n%s %s\n",
		"\033[36m📋\033[0m",
		"\033[1;36mTemplate Variables for Bootstrap Build:\033[0m")
	fmt.Printf("  %s: %s\n", "\033[90mName\033[0m", config.Name)

	if config.HetznerRobot != nil {
		fmt.Printf("  %s: %d server(s)\n", "\033[90mHetznerRobot.SelectedServers\033[0m", len(config.HetznerRobot.SelectedServers))
		for i, server := range config.HetznerRobot.SelectedServers {
			fmt.Printf("    %s [%d]:\n", "\033[90mServer\033[0m", i)
			fmt.Printf("      %s: %s\n", "\033[90mID\033[0m", server.ID)
			fmt.Printf("      %s: %s\n", "\033[90mName\033[0m", server.Name)
			fmt.Printf("      %s: %d disk(s)\n", "\033[90mStorageDisks\033[0m", len(server.StorageDisks))
			for j, disk := range server.StorageDisks {
				fmt.Printf("        [%d] %s: %s\n", j, disk.Name, disk.Path)
			}
		}
	}

	// Get provider-specific template path
	providerPath := "terraform/providers/hetzner-robot"

	// Compile bootstrap template (reads kubeconfig from Talos remote state)
	if err := compileTemplate(providerPath, "bootstrap.tf.tpl",
		filepath.Join(bootstrapDir, "main.tf"), config); err != nil {
		return fmt.Errorf("failed to compile bootstrap template: %w", err)
	}

	// Compile bootstrap-specific vars template
	if err := compileTemplate(providerPath, "vars.bootstrap.tf.tpl",
		filepath.Join(bootstrapDir, "vars.tf"), config); err != nil {
		return fmt.Errorf("failed to compile bootstrap vars template: %w", err)
	}

	return nil
}

// BuildHetznerRobotUbuntuFiles creates the ubuntu directory and compiles ubuntu.tf.tpl and vars.ubuntu.tf.tpl
func BuildHetznerRobotUbuntuFiles(config *config.CreateConfig) error {
    // Create .kibaship directory structure
    kibashipDir := ".kibaship"
    clusterDir := filepath.Join(kibashipDir, config.Name)
    ubuntuDir := filepath.Join(clusterDir, "ubuntu")

    // Create directories
    dirs := []string{kibashipDir, clusterDir, ubuntuDir}
    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir, err)
        }
    }

    // Log template variables being passed
    fmt.Printf("\n%s %s\n",
        "\033[36m📋\033[0m",
        "\033[1;36mTemplate Variables for Ubuntu Setup Build:\033[0m")
    fmt.Printf("  %s: %s\n", "\033[90mName\033[0m", config.Name)

    if config.HetznerRobot != nil {
        fmt.Printf("  %s: %d server(s)\n", "\033[90mHetznerRobot.SelectedServers\033[0m", len(config.HetznerRobot.SelectedServers))
        for i, server := range config.HetznerRobot.SelectedServers {
            fmt.Printf("    %s [%d]:\n", "\033[90mServer\033[0m", i)
            fmt.Printf("      %s: %s\n", "\033[90mID\033[0m", server.ID)
            fmt.Printf("      %s: %s\n", "\033[90mName\033[0m", server.Name)
            fmt.Printf("      %s: %s\n", "\033[90mIP\033[0m", server.IP)
        }
    }

    // Get provider-specific template path
    providerPath := "terraform/providers/hetzner-robot"

    // Compile ubuntu template
    if err := compileTemplate(providerPath, "ubuntu.tf.tpl",
        filepath.Join(ubuntuDir, "main.tf"), config); err != nil {
        return fmt.Errorf("failed to compile ubuntu template: %w", err)
    }

    // Compile ubuntu-specific vars template
    if err := compileTemplate(providerPath, "vars.ubuntu.tf.tpl",
        filepath.Join(ubuntuDir, "vars.tf"), config); err != nil {
        return fmt.Errorf("failed to compile ubuntu vars template: %w", err)
    }

    return nil
}

// BuildHetznerRobotTalosFiles creates the talos directory and compiles talos.tf.tpl and vars.talos.tf.tpl
func BuildHetznerRobotTalosFiles(config *config.CreateConfig) error {
	// Create .kibaship directory structure
	kibashipDir := ".kibaship"
	clusterDir := filepath.Join(kibashipDir, config.Name)
	talosDir := filepath.Join(clusterDir, "talos")

	// Create directories
	dirs := []string{kibashipDir, clusterDir, talosDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Log template variables being passed
	fmt.Printf("\n%s %s\n",
		"\033[36m📋\033[0m",
		"\033[1;36mTemplate Variables for Talos Bootstrap Build:\033[0m")
	fmt.Printf("  %s: %s\n", "\033[90mName\033[0m", config.Name)

	if config.HetznerRobot != nil {
		if config.HetznerRobot.TalosConfig != nil {
			fmt.Printf("  %s:\n", "\033[90mHetznerRobot.TalosConfig\033[0m")
			fmt.Printf("    %s: %s\n", "\033[90mClusterEndpoint\033[0m", config.HetznerRobot.TalosConfig.ClusterEndpoint)
			fmt.Printf("    %s: %d\n", "\033[90mVLANID\033[0m", config.HetznerRobot.TalosConfig.VLANID)
			fmt.Printf("    %s: %s\n", "\033[90mVSwitchSubnetIPRange\033[0m", config.HetznerRobot.TalosConfig.VSwitchSubnetIPRange)
		}

		if config.HetznerRobot.NetworkConfig != nil {
			fmt.Printf("  %s:\n", "\033[90mHetznerRobot.NetworkConfig\033[0m")
			fmt.Printf("    %s: %s\n", "\033[90mClusterNetworkIPRange\033[0m", config.HetznerRobot.NetworkConfig.ClusterNetworkIPRange)
		}

		fmt.Printf("  %s: %d server(s)\n", "\033[90mHetznerRobot.SelectedServers\033[0m", len(config.HetznerRobot.SelectedServers))
		for i, server := range config.HetznerRobot.SelectedServers {
			fmt.Printf("    %s [%d]:\n", "\033[90mServer\033[0m", i)
			fmt.Printf("      %s: %s\n", "\033[90mID\033[0m", server.ID)
			fmt.Printf("      %s: %s\n", "\033[90mName\033[0m", server.Name)
			fmt.Printf("      %s: %s\n", "\033[90mIP\033[0m", server.IP)
			fmt.Printf("      %s: %s\n", "\033[90mRole\033[0m", server.Role)
			fmt.Printf("      %s: %s\n", "\033[90mPublicNetworkInterface\033[0m", server.PublicNetworkInterface)
			fmt.Printf("      %s: %s\n", "\033[90mPublicAddressSubnet\033[0m", server.PublicAddressSubnet)
			fmt.Printf("      %s: %s\n", "\033[90mPublicIPv4Gateway\033[0m", server.PublicIPv4Gateway)
			fmt.Printf("      %s: %s\n", "\033[90mPrivateAddressSubnet\033[0m", server.PrivateAddressSubnet)
			fmt.Printf("      %s: %s\n", "\033[90mPrivateIPv4Gateway\033[0m", server.PrivateIPv4Gateway)
			fmt.Printf("      %s: %s\n", "\033[90mInstallationDisk\033[0m", server.InstallationDisk)
			fmt.Printf("      %s: %d disk(s)\n", "\033[90mStorageDisks\033[0m", len(server.StorageDisks))
			for j, disk := range server.StorageDisks {
				fmt.Printf("        [%d] %s: %s\n", j, disk.Name, disk.Path)
			}
		}
	}

	// Get provider-specific template path
	providerPath := "terraform/providers/hetzner-robot"

	// Compile talos template
	if err := compileTemplate(providerPath, "talos.tf.tpl",
		filepath.Join(talosDir, "main.tf"), config); err != nil {
		return fmt.Errorf("failed to compile talos template: %w", err)
	}

	// Compile talos-specific vars template
	if err := compileTemplate(providerPath, "vars.talos.tf.tpl",
		filepath.Join(talosDir, "vars.tf"), config); err != nil {
		return fmt.Errorf("failed to compile talos vars template: %w", err)
	}

	return nil
}

// compileTemplate loads a template from embedded filesystem and compiles it to a file
func compileTemplate(providerPath, templateName, outputPath string, config *config.CreateConfig) error {
	// Read template from embedded filesystem
	templatePath := filepath.Join(providerPath, templateName)
	templateContent, err := terraformTemplates.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Create new template with partials support and custom functions
	tmpl := template.New(templateName).Funcs(template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
	})

	// Load all partials from _partials directory
	partialsPath := "terraform/providers/_partials"
	partialEntries, err := fs.ReadDir(terraformTemplates, partialsPath)
	if err == nil {
		// Partials directory exists, load all .tpl files
		for _, entry := range partialEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tpl") {
				continue
			}

			partialPath := filepath.Join(partialsPath, entry.Name())
			partialContent, err := terraformTemplates.ReadFile(partialPath)
			if err != nil {
				return fmt.Errorf("failed to read partial %s: %w", partialPath, err)
			}

			// Parse the partial into the template
			if _, err := tmpl.Parse(string(partialContent)); err != nil {
				return fmt.Errorf("failed to parse partial %s: %w", entry.Name(), err)
			}
		}
	}
	// If partials directory doesn't exist, continue without them (not an error)

	// Parse main template
	if _, err := tmpl.Parse(string(templateContent)); err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer func() { _ = outputFile.Close() }()

	// Execute template with configuration data
	if err := tmpl.Execute(outputFile, config); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return nil
}

// maskSensitive masks sensitive values, showing only first 4 and last 4 characters
func maskSensitive(value string) string {
	if value == "" {
		return "(empty)"
	}
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}
