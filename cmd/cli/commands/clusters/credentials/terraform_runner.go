package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/config"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// TerraformOutput represents the structure of terraform output
type TerraformOutput struct {
	Kubeconfig struct {
		Value string `json:"value"`
	} `json:"kubeconfig"`
	KindClusterInfo struct {
		StoragePerNode string `json:"storage_per_node"`
		StorageTotal   string `json:"storage_total"`
	} `json:"kind_cluster_info"`
}

// extractCredentials extracts credentials from terraform output based on provider
func extractCredentials(config *config.CreateConfig) error {
	provisionDir := filepath.Join(".kibaship", config.Name, "provision")

	// Run terraform output to get all outputs in JSON format
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Dir = provisionDir

	// Set up environment variables (same as apply)
	env := os.Environ()
	env = append(env, fmt.Sprintf("TF_VAR_cluster_name=%s", config.Name))
	env = append(env, fmt.Sprintf("TF_VAR_cluster_email=%s", config.Email))
	env = append(env, fmt.Sprintf("TF_VAR_paas_features=%s", config.PaaSFeatures))

	// Add Terraform state configuration variables (only for cloud providers)
	if config.Provider != "kind" {
		env = append(env, fmt.Sprintf("TF_VAR_terraform_state_bucket=%s", config.TerraformState.S3Bucket))
		env = append(env, fmt.Sprintf("TF_VAR_terraform_state_region=%s", config.TerraformState.S3Region))
		env = append(env, fmt.Sprintf("TF_VAR_terraform_state_access_key=%s", config.TerraformState.S3AccessKey))
		env = append(env, fmt.Sprintf("TF_VAR_terraform_state_secret_key=%s", config.TerraformState.S3AccessSecret))
	}

	// Add provider-specific environment variables
	switch config.Provider {
	case "digital-ocean":
		if config.DigitalOcean != nil {
			env = append(env, fmt.Sprintf("TF_VAR_do_token=%s", config.DigitalOcean.Token))
			env = append(env, fmt.Sprintf("TF_VAR_do_region=%s", config.DigitalOcean.Region))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_count=%s", config.DigitalOcean.Nodes))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_size=%s", config.DigitalOcean.NodesSize))
		}
	case "kind":
		if config.Kind != nil {
			env = append(env, fmt.Sprintf("TF_VAR_kind_node_count=%s", config.Kind.Nodes))
			env = append(env, fmt.Sprintf("TF_VAR_kind_storage_per_node=%s", config.Kind.Storage))
		}
	}

	cmd.Env = env

	// Execute terraform output command
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("terraform output failed: %w", err)
	}

	// Parse the JSON output
	var terraformOutput TerraformOutput
	if err := json.Unmarshal(output, &terraformOutput); err != nil {
		return fmt.Errorf("failed to parse terraform output JSON: %w", err)
	}

	// Process credentials based on provider
	switch config.Provider {
	case "digital-ocean":
		return extractDigitalOceanCredentials(config, &terraformOutput)
	case "kind":
		return extractKindCredentials(config, &terraformOutput)
	case "hetzner-robot":
		return extractHetznerRobotCredentials(config)
	default:
		return fmt.Errorf("credential extraction not implemented for provider: %s", config.Provider)
	}
}

// extractDigitalOceanCredentials extracts and saves DigitalOcean cluster credentials
func extractDigitalOceanCredentials(config *config.CreateConfig, output *TerraformOutput) error {
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🌊"),
		styles.DescriptionStyle.Render("Processing DigitalOcean cluster credentials..."))

	// Check if kubeconfig output exists
	if output.Kubeconfig.Value == "" {
		return fmt.Errorf("kubeconfig output not found in terraform state")
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Kubeconfig found in terraform output"))

	// Save kubeconfig to file
	credentialsDir := filepath.Join(".kibaship", config.Name, "credentials")
	kubeconfigPath := filepath.Join(credentialsDir, "kubeconfig.yaml")

	// The kubeconfig from DigitalOcean is already in YAML format
	kubeconfigContent := output.Kubeconfig.Value

	// Write kubeconfig to file
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig file: %w", err)
	}

	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Kubeconfig saved successfully!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("File: %s", kubeconfigPath)))

	// Validate the kubeconfig content
	if err := validateKubeconfig(kubeconfigContent); err != nil {
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Warning: kubeconfig validation failed: %v", err)))
	} else {
		fmt.Printf("%s %s\n",
			styles.TitleStyle.Render("✅"),
			styles.TitleStyle.Render("Kubeconfig is valid"))
	}

	return nil
}

// extractKindCredentials extracts and saves Kind cluster credentials
func extractKindCredentials(config *config.CreateConfig, output *TerraformOutput) error {
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🐳"),
		styles.DescriptionStyle.Render("Processing Kind cluster credentials..."))

	// Check if kubeconfig output exists
	if output.Kubeconfig.Value == "" {
		return fmt.Errorf("kubeconfig output not found in terraform state")
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Kubeconfig found in terraform output"))

	// Save kubeconfig to file
	credentialsDir := filepath.Join(".kibaship", config.Name, "credentials")
	kubeconfigPath := filepath.Join(credentialsDir, "kubeconfig.yaml")

	// The kubeconfig from Kind is already in YAML format
	kubeconfigContent := output.Kubeconfig.Value

	// Write kubeconfig to file
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig file: %w", err)
	}

	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Kubeconfig saved successfully!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("File: %s", kubeconfigPath)))

	// Validate the kubeconfig content
	if err := validateKubeconfig(kubeconfigContent); err != nil {
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Warning: kubeconfig validation failed: %v", err)))
	} else {
		fmt.Printf("%s %s\n",
			styles.TitleStyle.Render("✅"),
			styles.TitleStyle.Render("Kubeconfig is valid"))
	}

	// Show Kind-specific information
	fmt.Printf("\n%s %s\n",
		styles.HelpStyle.Render("🐳"),
		styles.HelpStyle.Render("Kind Cluster Information:"))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Cluster Name:"),
		styles.DescriptionStyle.Render(config.Name))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Context:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("kind-%s", config.Name)))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Docker Network:"),
		styles.DescriptionStyle.Render("kind"))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("CNI:"),
		styles.DescriptionStyle.Render("Disabled (ready for Cilium)"))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Kube-proxy:"),
		styles.DescriptionStyle.Render("Disabled"))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Longhorn Storage:"),
		styles.DescriptionStyle.Render("Dedicated volumes per node"))

	// Get storage information from Terraform outputs
	if output.KindClusterInfo.StoragePerNode != "" {
		fmt.Printf("   %s %s\n",
			styles.CommandStyle.Render("Per Node:"),
			styles.DescriptionStyle.Render(output.KindClusterInfo.StoragePerNode))
	}
	if output.KindClusterInfo.StorageTotal != "" {
		fmt.Printf("   %s %s\n",
			styles.CommandStyle.Render("Total:"),
			styles.DescriptionStyle.Render(output.KindClusterInfo.StorageTotal))
	}

	return nil
}

// extractHetznerRobotCredentials extracts and saves Hetzner Robot cluster credentials from Talos terraform
func extractHetznerRobotCredentials(config *config.CreateConfig) error {
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🤖"),
		styles.DescriptionStyle.Render("Processing Hetzner Robot (Talos) cluster credentials..."))

	talosDir := filepath.Join(".kibaship", config.Name, "talos")

	// Run terraform output to get all outputs in JSON format from talos directory
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Dir = talosDir
	cmd.Env = os.Environ()

	// Execute terraform output command
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("terraform output failed in talos directory: %w", err)
	}

	// Parse the JSON output
	var outputs map[string]interface{}
	if err := json.Unmarshal(output, &outputs); err != nil {
		return fmt.Errorf("failed to parse terraform output JSON: %w", err)
	}

	credentialsDir := filepath.Join(".kibaship", config.Name, "credentials")

	// Extract talos_config (handles both string format from new template and object format from old template)
	if talosConfigOutput, ok := outputs["talos_config"].(map[string]interface{}); ok {
		talosConfigPath := filepath.Join(credentialsDir, "talosconfig.yaml")

		// Try new format: rendered YAML string
		if talosConfigValue, ok := talosConfigOutput["value"].(string); ok && talosConfigValue != "" {
			if err := os.WriteFile(talosConfigPath, []byte(talosConfigValue), 0600); err != nil {
				return fmt.Errorf("failed to write talosconfig file: %w", err)
			}
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render("Talos config saved successfully!"))
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("📁"),
				styles.DescriptionStyle.Render(fmt.Sprintf("File: %s", talosConfigPath)))
		} else if talosConfigObj, ok := talosConfigOutput["value"].(map[string]interface{}); ok {
			// Old format: object with ca_certificate, client_certificate, client_key
			// Construct talosconfig YAML manually
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render("Note: Using legacy talos_config format. Consider regenerating Terraform files."))

			caCert, _ := talosConfigObj["ca_certificate"].(string)
			clientCert, _ := talosConfigObj["client_certificate"].(string)
			clientKey, _ := talosConfigObj["client_key"].(string)

			if caCert != "" && clientCert != "" && clientKey != "" {
				// Get cluster info for endpoints
				endpoints := []string{}
				if clusterInfoOutput, ok := outputs["cluster_info"].(map[string]interface{}); ok {
					if clusterInfoValue, ok := clusterInfoOutput["value"].(map[string]interface{}); ok {
						if cpNodes, ok := clusterInfoValue["control_plane_nodes"].([]interface{}); ok {
							for _, node := range cpNodes {
								if nodeMap, ok := node.(map[string]interface{}); ok {
									if ip, ok := nodeMap["ip"].(string); ok {
										endpoints = append(endpoints, ip)
									}
								}
							}
						}
					}
				}

				// Construct talosconfig YAML
				talosConfigYAML := fmt.Sprintf(`context: %s
contexts:
    %s:
        endpoints:
%s
        ca: %s
        crt: %s
        key: %s
`, config.Name, config.Name, formatEndpointsYAML(endpoints), caCert, clientCert, clientKey)

				if err := os.WriteFile(talosConfigPath, []byte(talosConfigYAML), 0600); err != nil {
					return fmt.Errorf("failed to write talosconfig file: %w", err)
				}
				fmt.Printf("%s %s\n",
					styles.TitleStyle.Render("✅"),
					styles.TitleStyle.Render("Talos config saved successfully!"))
				fmt.Printf("%s %s\n",
					styles.CommandStyle.Render("📁"),
					styles.DescriptionStyle.Render(fmt.Sprintf("File: %s", talosConfigPath)))
			}
		}
	}

	// Extract kubeconfig
	if kubeconfigOutput, ok := outputs["kubeconfig"].(map[string]interface{}); ok {
		if kubeconfigValue, ok := kubeconfigOutput["value"].(string); ok && kubeconfigValue != "" {
			kubeconfigPath := filepath.Join(credentialsDir, "kubeconfig.yaml")
			if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigValue), 0600); err != nil {
				return fmt.Errorf("failed to write kubeconfig file: %w", err)
			}
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render("Kubeconfig saved successfully!"))
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("📁"),
				styles.DescriptionStyle.Render(fmt.Sprintf("File: %s", kubeconfigPath)))
		}
	}

	// Extract control plane machine configurations
	if cpConfigsOutput, ok := outputs["control_plane_machine_configurations"].(map[string]interface{}); ok {
		if cpConfigsValue, ok := cpConfigsOutput["value"].(map[string]interface{}); ok {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("🖥️"),
				styles.HelpStyle.Render("Extracting control plane machine configurations..."))

			for serverID, machineConfig := range cpConfigsValue {
				if machineConfigStr, ok := machineConfig.(string); ok && machineConfigStr != "" {
					machineConfigPath := filepath.Join(credentialsDir, fmt.Sprintf("cp-%s-machineconfig.yaml", serverID))
					if err := os.WriteFile(machineConfigPath, []byte(machineConfigStr), 0600); err != nil {
						return fmt.Errorf("failed to write control plane machine config for server %s: %w", serverID, err)
					}
					fmt.Printf("%s %s\n",
						styles.TitleStyle.Render("✅"),
						styles.DescriptionStyle.Render(fmt.Sprintf("Control plane machine config saved: %s", machineConfigPath)))
				}
			}
		}
	}

	// Extract worker machine configurations
	if workerConfigsOutput, ok := outputs["worker_machine_configurations"].(map[string]interface{}); ok {
		if workerConfigsValue, ok := workerConfigsOutput["value"].(map[string]interface{}); ok && len(workerConfigsValue) > 0 {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("💼"),
				styles.HelpStyle.Render("Extracting worker machine configurations..."))

			for serverID, machineConfig := range workerConfigsValue {
				if machineConfigStr, ok := machineConfig.(string); ok && machineConfigStr != "" {
					machineConfigPath := filepath.Join(credentialsDir, fmt.Sprintf("worker-%s-machineconfig.yaml", serverID))
					if err := os.WriteFile(machineConfigPath, []byte(machineConfigStr), 0600); err != nil {
						return fmt.Errorf("failed to write worker machine config for server %s: %w", serverID, err)
					}
					fmt.Printf("%s %s\n",
						styles.TitleStyle.Render("✅"),
						styles.DescriptionStyle.Render(fmt.Sprintf("Worker machine config saved: %s", machineConfigPath)))
				}
			}
		}
	}

	return nil
}

// validateKubeconfig performs basic validation on the kubeconfig content
func validateKubeconfig(content string) error {
	// Basic validation - check for required fields
	requiredFields := []string{"apiVersion", "clusters", "contexts", "users"}

	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}

// showUsageInstructions displays usage instructions for the extracted credentials
func showUsageInstructions(config *config.CreateConfig) {
	fmt.Printf("\n%s %s\n",
		styles.HelpStyle.Render("📖"),
		styles.HelpStyle.Render("Usage Instructions:"))

	kubeconfigPath := fmt.Sprintf(".kibaship/%s/credentials/kubeconfig.yaml", config.Name)

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("1."),
		styles.DescriptionStyle.Render("Set KUBECONFIG environment variable:"))
	fmt.Printf("   %s\n",
		styles.TitleStyle.Render(fmt.Sprintf("export KUBECONFIG=%s", kubeconfigPath)))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("2."),
		styles.DescriptionStyle.Render("Test cluster connection:"))
	fmt.Printf("   %s\n",
		styles.TitleStyle.Render("kubectl get nodes"))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("3."),
		styles.DescriptionStyle.Render("View cluster information:"))
	fmt.Printf("   %s\n",
		styles.TitleStyle.Render("kubectl cluster-info"))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("4."),
		styles.DescriptionStyle.Render("List all pods:"))
	fmt.Printf("   %s\n",
		styles.TitleStyle.Render("kubectl get pods --all-namespaces"))

	fmt.Printf("\n%s %s\n",
		styles.HelpStyle.Render("💡"),
		styles.HelpStyle.Render("Alternative Usage:"))
	fmt.Printf("   %s\n",
		styles.DescriptionStyle.Render("You can also use the kubeconfig directly with kubectl:"))
	fmt.Printf("   %s\n",
		styles.TitleStyle.Render(fmt.Sprintf("kubectl --kubeconfig=%s get nodes", kubeconfigPath)))

	fmt.Printf("\n%s %s\n",
		styles.HelpStyle.Render("🔒"),
		styles.HelpStyle.Render("Security Note:"))
	fmt.Printf("   %s\n",
		styles.DescriptionStyle.Render("The kubeconfig file contains sensitive credentials."))
	fmt.Printf("   %s\n",
		styles.DescriptionStyle.Render("Keep it secure and do not commit it to version control."))
	fmt.Printf("   %s\n",
		styles.DescriptionStyle.Render("File permissions: 600 (owner read/write only)"))

	// Show Kind-specific port information if it's a Kind cluster
	if config.Provider == "kind" {
		fmt.Printf("\n%s %s\n",
			styles.HelpStyle.Render("🐳"),
			styles.HelpStyle.Render("Kind Cluster Port Mappings:"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("• HTTP: localhost:14080 (instead of 80)"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("• HTTPS: localhost:14443 (instead of 443)"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("• DNS: localhost:14053 (instead of 53)"))
		if strings.Contains(config.PaaSFeatures, "mysql") {
			fmt.Printf("   %s\n",
				styles.DescriptionStyle.Render("• MySQL: localhost:14306 (instead of 3306)"))
		}
		if strings.Contains(config.PaaSFeatures, "postgres") {
			fmt.Printf("   %s\n",
				styles.DescriptionStyle.Render("• PostgreSQL: localhost:14432 (instead of 5432)"))
		}
		if strings.Contains(config.PaaSFeatures, "valkey") {
			fmt.Printf("   %s\n",
				styles.DescriptionStyle.Render("• Valkey/Redis: localhost:14379 (instead of 6379)"))
		}
		fmt.Printf("\n%s %s\n",
			styles.HelpStyle.Render("💡"),
			styles.HelpStyle.Render("Next Steps:"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("1. Install Cilium CNI:"))
		fmt.Printf("      %s\n",
			styles.TitleStyle.Render("cilium install"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("2. Wait for Cilium to be ready:"))
		fmt.Printf("      %s\n",
			styles.TitleStyle.Render("cilium status --wait"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("3. Test cluster connectivity:"))
		fmt.Printf("      %s\n",
			styles.TitleStyle.Render("kubectl get nodes"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("4. Verify storage setup:"))
		fmt.Printf("      %s\n",
			styles.TitleStyle.Render("kubectl get storageclass"))
		fmt.Printf("\n%s %s\n",
			styles.HelpStyle.Render("💾"),
			styles.HelpStyle.Render("Longhorn Storage Information:"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("Each node has dedicated Longhorn storage mounted at:"))
		fmt.Printf("   %s\n",
			styles.TitleStyle.Render("/tmp/kibaship-longhorn/"+config.Name+"-*"))
		fmt.Printf("   %s\n",
			styles.DescriptionStyle.Render("This enables Longhorn distributed storage for persistent volumes."))
		fmt.Printf("\n%s %s\n",
			styles.HelpStyle.Render("💡"),
			styles.HelpStyle.Render("Example Service Access:"))
		fmt.Printf("   %s\n",
			styles.TitleStyle.Render("curl http://localhost:14080"))
		fmt.Printf("   %s\n",
			styles.TitleStyle.Render("curl https://localhost:14443"))
	}
}

// formatEndpointsYAML formats a list of endpoints for talosconfig YAML
func formatEndpointsYAML(endpoints []string) string {
	if len(endpoints) == 0 {
		return "            - localhost"
	}

	var formatted strings.Builder
	for _, endpoint := range endpoints {
		formatted.WriteString(fmt.Sprintf("            - %s\n", endpoint))
	}
	return strings.TrimSuffix(formatted.String(), "\n")
}

// checkTerraformInstalled checks if terraform is available in PATH
func checkTerraformInstalled() error {
	cmd := exec.Command("terraform", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform is not installed or not available in PATH. " +
			"Please install Terraform: https://terraform.io/downloads")
	}
	return nil
}
