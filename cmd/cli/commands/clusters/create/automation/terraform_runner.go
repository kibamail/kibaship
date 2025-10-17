package automation

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/config"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// runTerraformInit runs terraform init in the provision directory with backend configuration
// State path: clusters/<cluster>/provision.terraform.tfstate
// Future bootstrap state path will be: clusters/<cluster>/bootstrap.terraform.tfstate
func RunTerraformInit(config *config.CreateConfig) error {
	provisionDir := filepath.Join(".kibaship", config.Name, "provision")

	// Prepare backend configuration arguments
	backendArgs := []string{
		"init",
		fmt.Sprintf("-backend-config=bucket=%s", config.TerraformState.S3Bucket),
		fmt.Sprintf("-backend-config=key=clusters/%s/provision.terraform.tfstate", config.Name),
		fmt.Sprintf("-backend-config=region=%s", config.TerraformState.S3Region),
		fmt.Sprintf("-backend-config=access_key=%s", config.TerraformState.S3AccessKey),
		fmt.Sprintf("-backend-config=secret_key=%s", config.TerraformState.S3AccessSecret),
		"-backend-config=encrypt=true",
	}

	// Create terraform command
	cmd := exec.Command("terraform", backendArgs...)
	cmd.Dir = provisionDir

	// Set up environment variables
	cmd.Env = os.Environ()

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start terraform init: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	return nil
}

// streamOutput streams command output to console with proper formatting using styles
func streamOutput(reader io.Reader, _ string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Add color formatting for different types of output using styles
		if strings.Contains(line, "Initializing") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🔄"),
				styles.DescriptionStyle.Render(line))
		} else if strings.Contains(line, "Successfully") || strings.Contains(line, "Success!") {
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render(line))
		} else if strings.Contains(line, "Creating...") || strings.Contains(line, "Creation complete") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🔨"),
				styles.DescriptionStyle.Render(line))
		} else if strings.Contains(line, "Refreshing") || strings.Contains(line, "Reading") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🔄"),
				styles.DescriptionStyle.Render(line))
		} else if strings.Contains(line, "Plan:") || strings.Contains(line, "Apply complete!") {
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("📊"),
				styles.TitleStyle.Render(line))
		} else if strings.Contains(line, "Error") || strings.Contains(line, "error") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("❌"),
				styles.CommandStyle.Render(line))
		} else if strings.Contains(line, "Warning") || strings.Contains(line, "warning") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render(line))
		} else if strings.Contains(line, "Downloading") || strings.Contains(line, "Installing") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("📥"),
				styles.DescriptionStyle.Render(line))
		} else if strings.HasPrefix(line, "Terraform") {
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("🏗️"),
				styles.TitleStyle.Render(line))
		} else if line != "" {
			fmt.Printf("   %s\n", styles.DescriptionStyle.Render(line))
		}
	}
}

// runTerraformValidate runs terraform validate in the provision directory
func RunTerraformValidate(config *config.CreateConfig) error {
	provisionDir := filepath.Join(".kibaship", config.Name, "provision")

	// Create terraform validate command
	cmd := exec.Command("terraform", "validate")
	cmd.Dir = provisionDir

	// Set up environment variables
	cmd.Env = os.Environ()

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start terraform validate: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("terraform validate failed: %w", err)
	}

	return nil
}

// runTerraformApply runs terraform apply in the provision directory
func RunTerraformApply(config *config.CreateConfig) error {
	provisionDir := filepath.Join(".kibaship", config.Name, "provision")

	// Set up TF_VAR environment variables for Terraform
	env := os.Environ()
	env = append(env, fmt.Sprintf("TF_VAR_cluster_name=%s", config.Name))
	env = append(env, fmt.Sprintf("TF_VAR_cluster_email=%s", config.Email))
	env = append(env, fmt.Sprintf("TF_VAR_paas_features=%s", config.PaaSFeatures))

	// Add Terraform state configuration variables
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_bucket=%s", config.TerraformState.S3Bucket))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_region=%s", config.TerraformState.S3Region))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_access_key=%s", config.TerraformState.S3AccessKey))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_secret_key=%s", config.TerraformState.S3AccessSecret))

	// Add provider-specific environment variables
	switch config.Provider {
	case "digital-ocean":
		if config.DigitalOcean != nil {
			env = append(env, fmt.Sprintf("TF_VAR_do_token=%s", config.DigitalOcean.Token))
			env = append(env, fmt.Sprintf("TF_VAR_do_region=%s", config.DigitalOcean.Region))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_count=%s", config.DigitalOcean.Nodes))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_size=%s", config.DigitalOcean.NodesSize))
		}
	case "hetzner-robot":
		if config.HetznerRobot != nil {
			// Add Hetzner Cloud token
			env = append(env, fmt.Sprintf("TF_VAR_hcloud_token=%s", config.HetznerRobot.CloudToken))

			// Add vSwitch ID if available
			if config.HetznerRobot.VSwitchID != "" {
				env = append(env, fmt.Sprintf("TF_VAR_vswitch_id=%s", config.HetznerRobot.VSwitchID))
			}

			// Add network configuration if available
			if config.HetznerRobot.NetworkConfig != nil {
				env = append(env, fmt.Sprintf("TF_VAR_location=%s", config.HetznerRobot.NetworkConfig.Location))
				env = append(env, fmt.Sprintf("TF_VAR_network_zone=%s", config.HetznerRobot.NetworkConfig.NetworkZone))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_network_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterNetworkIPRange))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_vswitch_subnet_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_subnet_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterSubnetIPRange))
			}

			// Add rescue passwords as environment variables
			for serverID, password := range config.HetznerRobot.RescuePasswords {
				env = append(env, fmt.Sprintf("TF_VAR_server_%s_password=%s", serverID, password))
			}

            // Add Talos configuration
            if config.HetznerRobot.TalosConfig != nil {
                env = append(env, fmt.Sprintf("TF_VAR_cluster_endpoint=%s", config.HetznerRobot.TalosConfig.ClusterEndpoint))
                env = append(env, fmt.Sprintf("TF_VAR_cluster_dns_name=%s", fmt.Sprintf("kube.%s", config.Domain)))
                env = append(env, fmt.Sprintf("TF_VAR_vlan_id=%d", config.HetznerRobot.TalosConfig.VLANID))
                env = append(env, fmt.Sprintf("TF_VAR_vswitch_subnet_ip_range=%s", config.HetznerRobot.TalosConfig.VSwitchSubnetIPRange))
            }

			// Add server private IPs and network configuration
			for _, server := range config.HetznerRobot.SelectedServers {
				if server.PrivateIP != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_ip=%s", server.ID, server.PrivateIP))
				}

				// Add Talos network configuration for each server
				if server.PublicNetworkInterface != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_network_interface=%s", server.ID, server.PublicNetworkInterface))
				}
				if server.PublicAddressSubnet != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_address_subnet=%s", server.ID, server.PublicAddressSubnet))
				}
				if server.PublicIPv4Gateway != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_ipv4_gateway=%s", server.ID, server.PublicIPv4Gateway))
				}
				if server.PrivateAddressSubnet != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_address_subnet=%s", server.ID, server.PrivateAddressSubnet))
				}
				if server.PrivateIPv4Gateway != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_ipv4_gateway=%s", server.ID, server.PrivateIPv4Gateway))
				}
				if server.InstallationDisk != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_installation_disk=%s", server.ID, server.InstallationDisk))
				}
			}
		}
	}

	// Log all Terraform variables being passed
	fmt.Printf("\n%s %s\n",
		"\033[35m🔧\033[0m",
		"\033[1;35mTerraform Variables for Provision Apply:\033[0m")

	// Extract and display all TF_VAR_ environment variables
	tfVars := make(map[string]string)
	for _, envVar := range env {
		if strings.HasPrefix(envVar, "TF_VAR_") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimPrefix(parts[0], "TF_VAR_")
				varValue := parts[1]

				// Mask sensitive variables
				if isSensitiveVar(varName) {
					varValue = maskSensitiveValue(varValue)
				}

				tfVars[varName] = varValue
			}
		}
	}

	// Display variables in sorted order for readability
	varNames := make([]string, 0, len(tfVars))
	for name := range tfVars {
		varNames = append(varNames, name)
	}
	sort.Strings(varNames)

	for _, name := range varNames {
		fmt.Printf("  %s: %s\n", fmt.Sprintf("\033[90m%s\033[0m", name), tfVars[name])
	}

	// Create terraform apply command with auto-approve
	cmd := exec.Command("terraform", "apply", "-auto-approve")
	cmd.Dir = provisionDir
	cmd.Env = env

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start terraform apply: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	return nil
}

// runBootstrapTerraformInit runs terraform init in the bootstrap directory
func RunBootstrapTerraformInit(config *config.CreateConfig) error {
	bootstrapDir := filepath.Join(".kibaship", config.Name, "bootstrap")

	// Prepare backend configuration arguments
	backendArgs := []string{
		"init",
		"-reconfigure",
		fmt.Sprintf("-backend-config=bucket=%s", config.TerraformState.S3Bucket),
		fmt.Sprintf("-backend-config=key=clusters/%s/bootstrap.terraform.tfstate", config.Name),
		fmt.Sprintf("-backend-config=region=%s", config.TerraformState.S3Region),
		fmt.Sprintf("-backend-config=access_key=%s", config.TerraformState.S3AccessKey),
		fmt.Sprintf("-backend-config=secret_key=%s", config.TerraformState.S3AccessSecret),
		"-backend-config=encrypt=true",
	}

	// Create terraform command
	cmd := exec.Command("terraform", backendArgs...)
	cmd.Dir = bootstrapDir

	// Set up environment variables
	cmd.Env = os.Environ()

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start bootstrap terraform init: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("bootstrap terraform init failed: %w", err)
	}

	return nil
}

// runBootstrapTerraformValidate runs terraform validate in the bootstrap directory
func RunBootstrapTerraformValidate(config *config.CreateConfig) error {
	bootstrapDir := filepath.Join(".kibaship", config.Name, "bootstrap")

	// Create terraform validate command
	cmd := exec.Command("terraform", "validate")
	cmd.Dir = bootstrapDir

	// Set up environment variables including AWS credentials for remote state access
	env := os.Environ()
	env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", config.TerraformState.S3AccessKey))
	env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", config.TerraformState.S3AccessSecret))
	env = append(env, fmt.Sprintf("AWS_REGION=%s", config.TerraformState.S3Region))
	cmd.Env = env

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start bootstrap terraform validate: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("bootstrap terraform validate failed: %w", err)
	}

	return nil
}

// runBootstrapTerraformApply runs terraform apply in the bootstrap directory
func RunBootstrapTerraformApply(config *config.CreateConfig) error {
	bootstrapDir := filepath.Join(".kibaship", config.Name, "bootstrap")

	// Set up TF_VAR environment variables for Terraform
	env := os.Environ()

	// Add AWS credentials for remote state access
	env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", config.TerraformState.S3AccessKey))
	env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", config.TerraformState.S3AccessSecret))
	env = append(env, fmt.Sprintf("AWS_REGION=%s", config.TerraformState.S3Region))

	env = append(env, fmt.Sprintf("TF_VAR_cluster_name=%s", config.Name))
	env = append(env, fmt.Sprintf("TF_VAR_cluster_email=%s", config.Email))
	env = append(env, fmt.Sprintf("TF_VAR_paas_features=%s", config.PaaSFeatures))

	// Add provider-specific environment variables
	switch config.Provider {
	case "digital-ocean":
		if config.DigitalOcean != nil {
			env = append(env, fmt.Sprintf("TF_VAR_do_token=%s", config.DigitalOcean.Token))
			env = append(env, fmt.Sprintf("TF_VAR_do_region=%s", config.DigitalOcean.Region))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_count=%s", config.DigitalOcean.Nodes))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_size=%s", config.DigitalOcean.NodesSize))
		}
	case "hetzner-robot":
		if config.HetznerRobot != nil {
			// Add Hetzner Cloud token
			env = append(env, fmt.Sprintf("TF_VAR_hcloud_token=%s", config.HetznerRobot.CloudToken))

			// Add vSwitch ID if available
			if config.HetznerRobot.VSwitchID != "" {
				env = append(env, fmt.Sprintf("TF_VAR_vswitch_id=%s", config.HetznerRobot.VSwitchID))
			}

			// Add network configuration if available
			if config.HetznerRobot.NetworkConfig != nil {
				env = append(env, fmt.Sprintf("TF_VAR_location=%s", config.HetznerRobot.NetworkConfig.Location))
				env = append(env, fmt.Sprintf("TF_VAR_network_zone=%s", config.HetznerRobot.NetworkConfig.NetworkZone))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_network_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterNetworkIPRange))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_vswitch_subnet_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_subnet_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterSubnetIPRange))
			}

			// Add rescue passwords as environment variables
			for serverID, password := range config.HetznerRobot.RescuePasswords {
				env = append(env, fmt.Sprintf("TF_VAR_server_%s_password=%s", serverID, password))
			}

			// Add Talos configuration if available
            if config.HetznerRobot.TalosConfig != nil {
                env = append(env, fmt.Sprintf("TF_VAR_cluster_endpoint=%s", config.HetznerRobot.TalosConfig.ClusterEndpoint))
                env = append(env, fmt.Sprintf("TF_VAR_cluster_dns_name=%s", fmt.Sprintf("kube.%s", config.Domain)))
                env = append(env, fmt.Sprintf("TF_VAR_vlan_id=%d", config.HetznerRobot.TalosConfig.VLANID))
                env = append(env, fmt.Sprintf("TF_VAR_vswitch_subnet_ip_range=%s", config.HetznerRobot.TalosConfig.VSwitchSubnetIPRange))
            }

			// Add server private IPs and network configuration
			for _, server := range config.HetznerRobot.SelectedServers {
				if server.PrivateIP != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_ip=%s", server.ID, server.PrivateIP))
				}

				// Add Talos network configuration for each server
				if server.PublicNetworkInterface != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_network_interface=%s", server.ID, server.PublicNetworkInterface))
				}
				if server.PublicAddressSubnet != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_address_subnet=%s", server.ID, server.PublicAddressSubnet))
				}
				if server.PublicIPv4Gateway != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_ipv4_gateway=%s", server.ID, server.PublicIPv4Gateway))
				}
				if server.PrivateAddressSubnet != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_address_subnet=%s", server.ID, server.PrivateAddressSubnet))
				}
				if server.PrivateIPv4Gateway != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_ipv4_gateway=%s", server.ID, server.PrivateIPv4Gateway))
				}
				if server.InstallationDisk != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_installation_disk=%s", server.ID, server.InstallationDisk))
				}
			}
		}
	}

	// Create terraform apply command with auto-approve
	cmd := exec.Command("terraform", "apply", "-auto-approve")
	cmd.Dir = bootstrapDir
	cmd.Env = env

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start bootstrap terraform apply: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("bootstrap terraform apply failed: %w", err)
	}

	return nil
}

// CheckTerraformInstalled checks if terraform is available in PATH
func CheckTerraformInstalled() error {
	cmd := exec.Command("terraform", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform is not installed or not available in PATH. " +
			"Please install Terraform: https://terraform.io/downloads")
	}
	return nil
}

// ReadProvisionTerraformOutputs reads the Terraform outputs from the provision phase
func ReadProvisionTerraformOutputs(config *config.CreateConfig) (map[string]interface{}, error) {
	provisionDir := filepath.Join(".kibaship", config.Name, "provision")

	// Create terraform output command to get JSON output
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Dir = provisionDir
	cmd.Env = os.Environ()

	// Capture output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform outputs: %w", err)
	}

	// Parse JSON output
	var outputs map[string]interface{}
	if err := json.Unmarshal(output, &outputs); err != nil {
		return nil, fmt.Errorf("failed to parse terraform outputs: %w", err)
	}

	return outputs, nil
}

// Cloud phase removed for hetzner-robot; no cloud outputs to read



// RunTalosTerraformInit runs terraform init in the talos directory (for Hetzner Robot)
func RunTalosTerraformInit(config *config.CreateConfig) error {
	talosDir := filepath.Join(".kibaship", config.Name, "talos")

	// Prepare backend configuration arguments
	backendArgs := []string{
		"init",
		fmt.Sprintf("-backend-config=bucket=%s", config.TerraformState.S3Bucket),
		fmt.Sprintf("-backend-config=key=clusters/%s/bare-metal-talos-bootstrap/terraform.tfstate", config.Name),
		fmt.Sprintf("-backend-config=region=%s", config.TerraformState.S3Region),
		fmt.Sprintf("-backend-config=access_key=%s", config.TerraformState.S3AccessKey),
		fmt.Sprintf("-backend-config=secret_key=%s", config.TerraformState.S3AccessSecret),
		"-backend-config=encrypt=true",
	}

	// Create terraform command
	cmd := exec.Command("terraform", backendArgs...)
	cmd.Dir = talosDir

	// Set up environment variables
	cmd.Env = os.Environ()

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start talos terraform init: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("talos terraform init failed: %w", err)
	}

	return nil
}

// RunTalosTerraformValidate runs terraform validate in the talos directory (for Hetzner Robot)
func RunTalosTerraformValidate(config *config.CreateConfig) error {
	talosDir := filepath.Join(".kibaship", config.Name, "talos")

	// Create terraform validate command
	cmd := exec.Command("terraform", "validate")
	cmd.Dir = talosDir

	// Set up environment variables
	cmd.Env = os.Environ()

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start talos terraform validate: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("talos terraform validate failed: %w", err)
	}

	return nil
}

// RunTalosTerraformApply runs terraform apply in the talos directory (for Hetzner Robot)
func RunTalosTerraformApply(config *config.CreateConfig) error {
	talosDir := filepath.Join(".kibaship", config.Name, "talos")

	// Set up TF_VAR environment variables for Terraform
	env := os.Environ()
	env = append(env, fmt.Sprintf("TF_VAR_cluster_name=%s", config.Name))

	// Add Talos configuration
	if config.HetznerRobot.TalosConfig != nil {
		env = append(env, fmt.Sprintf("TF_VAR_cluster_endpoint=%s", config.HetznerRobot.TalosConfig.ClusterEndpoint))
		env = append(env, fmt.Sprintf("TF_VAR_cluster_dns_name=%s", fmt.Sprintf("kube.%s", config.Domain)))
		env = append(env, fmt.Sprintf("TF_VAR_vlan_id=%d", config.HetznerRobot.TalosConfig.VLANID))
		env = append(env, fmt.Sprintf("TF_VAR_vswitch_subnet_ip_range=%s", config.HetznerRobot.TalosConfig.VSwitchSubnetIPRange))
	}

	// Add network configuration
	if config.HetznerRobot.NetworkConfig != nil {
		env = append(env, fmt.Sprintf("TF_VAR_cluster_network_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterNetworkIPRange))
	}

	// Add server-specific network configuration (discovered in Phase 3)
	for _, server := range config.HetznerRobot.SelectedServers {
		if server.PublicNetworkInterface != "" {
			env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_network_interface=%s", server.ID, server.PublicNetworkInterface))
		}
		if server.PublicAddressSubnet != "" {
			env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_address_subnet=%s", server.ID, server.PublicAddressSubnet))
		}
		if server.PublicIPv4Gateway != "" {
			env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_ipv4_gateway=%s", server.ID, server.PublicIPv4Gateway))
		}
		if server.PrivateAddressSubnet != "" {
			env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_address_subnet=%s", server.ID, server.PrivateAddressSubnet))
		}
		if server.PrivateIPv4Gateway != "" {
			env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_ipv4_gateway=%s", server.ID, server.PrivateIPv4Gateway))
		}
		if server.InstallationDisk != "" {
			env = append(env, fmt.Sprintf("TF_VAR_server_%s_installation_disk=%s", server.ID, server.InstallationDisk))
		}

		// Add storage disks as JSON-encoded string
		if len(server.StorageDisks) > 0 {
			storageDisksJSON, err := json.Marshal(server.StorageDisks)
			if err == nil {
				env = append(env, fmt.Sprintf("TF_VAR_server_%s_storage_disks=%s", server.ID, string(storageDisksJSON)))
			}
		}
	}

	// Log all Terraform variables being passed
	fmt.Printf("\n%s %s\n",
		"\033[35m🔧\033[0m",
		"\033[1;35mTerraform Variables for Talos Bootstrap Apply:\033[0m")

	// Extract and display all TF_VAR_ environment variables
	tfVars := make(map[string]string)
	for _, envVar := range env {
		if strings.HasPrefix(envVar, "TF_VAR_") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimPrefix(parts[0], "TF_VAR_")
				varValue := parts[1]

				// Mask sensitive variables
				if isSensitiveVar(varName) {
					varValue = maskSensitiveValue(varValue)
				}

				tfVars[varName] = varValue
			}
		}
	}

	// Display variables in sorted order for readability
	varNames := make([]string, 0, len(tfVars))
	for name := range tfVars {
		varNames = append(varNames, name)
	}
	sort.Strings(varNames)

	for _, name := range varNames {
		fmt.Printf("  %s: %s\n", fmt.Sprintf("\033[90m%s\033[0m", name), tfVars[name])
	}

	// Create terraform apply command with auto-approve
	cmd := exec.Command("terraform", "apply", "-auto-approve")
	cmd.Dir = talosDir
	cmd.Env = env

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start talos terraform apply: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("talos terraform apply failed: %w", err)
	}

	return nil
}

// ReadTalosTerraformOutputs reads the Terraform outputs from the talos bootstrap phase
func ReadTalosTerraformOutputs(config *config.CreateConfig) (map[string]interface{}, error) {
	talosDir := filepath.Join(".kibaship", config.Name, "talos")

	// Create terraform output command to get JSON output
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Dir = talosDir
	cmd.Env = os.Environ()

	// Capture output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform outputs: %w", err)
	}

	// Parse JSON output
	var outputs map[string]interface{}
	if err := json.Unmarshal(output, &outputs); err != nil {
		return nil, fmt.Errorf("failed to parse terraform outputs: %w", err)
	}

	return outputs, nil
}

// runTerraformDestroy runs terraform destroy in the provision directory
func RunTerraformDestroy(config *config.CreateConfig) error {
	provisionDir := filepath.Join(".kibaship", config.Name, "provision")

	// Set up TF_VAR environment variables for Terraform
	env := os.Environ()
	env = append(env, fmt.Sprintf("TF_VAR_cluster_name=%s", config.Name))
	env = append(env, fmt.Sprintf("TF_VAR_cluster_email=%s", config.Email))
	env = append(env, fmt.Sprintf("TF_VAR_paas_features=%s", config.PaaSFeatures))

	// Add Terraform state configuration variables
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_bucket=%s", config.TerraformState.S3Bucket))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_region=%s", config.TerraformState.S3Region))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_access_key=%s", config.TerraformState.S3AccessKey))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_secret_key=%s", config.TerraformState.S3AccessSecret))

	// Add provider-specific environment variables
	switch config.Provider {
	case "digital-ocean":
		if config.DigitalOcean != nil {
			env = append(env, fmt.Sprintf("TF_VAR_do_token=%s", config.DigitalOcean.Token))
			env = append(env, fmt.Sprintf("TF_VAR_do_region=%s", config.DigitalOcean.Region))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_count=%s", config.DigitalOcean.Nodes))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_size=%s", config.DigitalOcean.NodesSize))
		}
	case "hetzner-robot":
		if config.HetznerRobot != nil {
			// Add Hetzner Cloud token
			env = append(env, fmt.Sprintf("TF_VAR_hcloud_token=%s", config.HetznerRobot.CloudToken))

			// Add vSwitch ID if available
			if config.HetznerRobot.VSwitchID != "" {
				env = append(env, fmt.Sprintf("TF_VAR_vswitch_id=%s", config.HetznerRobot.VSwitchID))
			}

			// Add network configuration if available
			if config.HetznerRobot.NetworkConfig != nil {
				env = append(env, fmt.Sprintf("TF_VAR_location=%s", config.HetznerRobot.NetworkConfig.Location))
				env = append(env, fmt.Sprintf("TF_VAR_network_zone=%s", config.HetznerRobot.NetworkConfig.NetworkZone))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_network_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterNetworkIPRange))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_vswitch_subnet_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange))
				env = append(env, fmt.Sprintf("TF_VAR_cluster_subnet_ip_range=%s", config.HetznerRobot.NetworkConfig.ClusterSubnetIPRange))
			}

			// Add rescue passwords as environment variables
			for serverID, password := range config.HetznerRobot.RescuePasswords {
				env = append(env, fmt.Sprintf("TF_VAR_server_%s_password=%s", serverID, password))
			}

			// Add Talos configuration if available
			if config.HetznerRobot.TalosConfig != nil {
				env = append(env, fmt.Sprintf("TF_VAR_cluster_endpoint=%s", config.HetznerRobot.TalosConfig.ClusterEndpoint))
				env = append(env, fmt.Sprintf("TF_VAR_vlan_id=%d", config.HetznerRobot.TalosConfig.VLANID))
				env = append(env, fmt.Sprintf("TF_VAR_vswitch_subnet_ip_range=%s", config.HetznerRobot.TalosConfig.VSwitchSubnetIPRange))
			}

			// Add server private IPs and network configuration
			for _, server := range config.HetznerRobot.SelectedServers {
				if server.PrivateIP != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_ip=%s", server.ID, server.PrivateIP))
				}

				// Add Talos network configuration for each server
				if server.PublicNetworkInterface != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_network_interface=%s", server.ID, server.PublicNetworkInterface))
				}
				if server.PublicAddressSubnet != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_address_subnet=%s", server.ID, server.PublicAddressSubnet))
				}
				if server.PublicIPv4Gateway != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_public_ipv4_gateway=%s", server.ID, server.PublicIPv4Gateway))
				}
				if server.PrivateAddressSubnet != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_address_subnet=%s", server.ID, server.PrivateAddressSubnet))
				}
				if server.PrivateIPv4Gateway != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_private_ipv4_gateway=%s", server.ID, server.PrivateIPv4Gateway))
				}
				if server.InstallationDisk != "" {
					env = append(env, fmt.Sprintf("TF_VAR_server_%s_installation_disk=%s", server.ID, server.InstallationDisk))
				}
			}
		}
	}

	// Create terraform destroy command with auto-approve
	cmd := exec.Command("terraform", "destroy", "-auto-approve")
	cmd.Dir = provisionDir
	cmd.Env = env

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start terraform destroy: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	return nil
}

// isSensitiveVar checks if a variable name contains sensitive keywords
func isSensitiveVar(varName string) bool {
	lowerName := strings.ToLower(varName)
	sensitiveKeywords := []string{"password", "secret", "token", "key", "credential"}
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowerName, keyword) {
			return true
		}
	}
	return false
}

// maskSensitiveValue masks sensitive values, showing only first 4 and last 4 characters
func maskSensitiveValue(value string) string {
	if value == "" {
		return "(empty)"
	}
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

// RunCommand executes a shell command with a timeout
func RunCommand(command string, dir string, timeout time.Duration) error {
	// Create command with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute command using shell
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if dir != "" {
		cmd.Dir = dir
	}

	// Run the command
	err := cmd.Run()

	// Check if context deadline exceeded
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("command timed out after %v", timeout)
	}

	return err
}
