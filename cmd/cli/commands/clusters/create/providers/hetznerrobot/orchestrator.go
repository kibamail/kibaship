package hetznerrobot

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/automation"
	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/config"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// PerformServerSelection handles the interactive server selection for Hetzner Robot
func PerformServerSelection(cfg *config.CreateConfig) error {
	// Create Hetzner Robot client
	client, err := NewClientWithToken(fmt.Sprintf("%s:%s",
		cfg.HetznerRobot.Username, cfg.HetznerRobot.Password))
	if err != nil {
		return fmt.Errorf("failed to create Hetzner Robot client: %w", err)
	}

	// Validate credentials first
	ctx := context.Background()
	if err := client.ValidateCredentials(ctx); err != nil {
		return fmt.Errorf("failed to validate Hetzner Robot credentials: %w", err)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Hetzner Robot credentials validated successfully!"))

	// Display servers summary
	if err := DisplayServersSummary(ctx, client); err != nil {
		return fmt.Errorf("failed to display servers summary: %w", err)
	}

	// Complete server and vswitch selection process
	selection, err := SelectServersAndVSwitchInteractive(ctx, client, cfg.Name)
	if err != nil {
		return fmt.Errorf("server and vswitch selection failed: %w", err)
	}

	// Store the selection and network ranges in the config for later use by Terraform
	if err := storeSelection(cfg, selection); err != nil {
		return fmt.Errorf("failed to store server selection: %w", err)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Server and VSwitch selection completed successfully!"))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📋"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Selected %d servers for %s cluster",
			len(selection.ServerSelection.SelectedServers), selection.ServerSelection.ClusterType)))

	if selection.VSwitchSelection.SelectedVSwitch != nil {
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("🔗"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Using vswitch: %s (VLAN %d)",
				selection.VSwitchSelection.SelectedVSwitch.Name,
				selection.VSwitchSelection.SelectedVSwitch.VLAN)))
	}

	return nil
}

// storeSelection stores the server selection results in the config for Terraform templating
func storeSelection(cfg *config.CreateConfig, selection *CompleteSelectionResult) error {
	if cfg.HetznerRobot == nil {
		return fmt.Errorf("HetznerRobot config is nil")
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📋"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Storing selection of %d servers",
			len(selection.ServerSelection.SelectedServers))))

	// Convert Server objects to HetznerRobotServer objects
	selectedServers := make([]config.HetznerRobotServer, 0, len(selection.ServerSelection.SelectedServers))
	for i, server := range selection.ServerSelection.SelectedServers {
		// Determine role based on cluster type and index
		role := determineServerRole(i, selection.ServerSelection.ClusterType, len(selection.ServerSelection.SelectedServers))

		// Generate private IP based on index (starting from .10 in the vSwitch subnet)
		// Example: if vSwitch subnet is 172.20.1.0/24, first server gets 172.20.1.10
		privateIP := ""
		if selection.NetworkRanges != nil {
			privateIP = generatePrivateIP(selection.NetworkRanges.ClusterVSwitchSubnetIPRange, i+10)
		}

		robotServer := config.HetznerRobotServer{
			ID:        server.ID,
			Name:      server.Name,
			IP:        server.IP,
			PrivateIP: privateIP,
			Status:    server.Status,
			Product:   server.Product,
			DC:        server.DC,
			Role:      role,
		}

		// Add rescue password if available
		if selection.RescueResult != nil && selection.RescueResult.RescuePasswords != nil {
			if password, exists := selection.RescueResult.RescuePasswords[server.ID]; exists {
				robotServer.RescuePassword = password
			}
		}

		selectedServers = append(selectedServers, robotServer)
	}

	// Store the converted servers
	cfg.HetznerRobot.SelectedServers = selectedServers

	// Store rescue passwords if available
	if selection.RescueResult != nil && selection.RescueResult.RescuePasswords != nil {
		cfg.HetznerRobot.RescuePasswords = selection.RescueResult.RescuePasswords
	} else {
		// Initialize rescue passwords map if not already done
		if cfg.HetznerRobot.RescuePasswords == nil {
			cfg.HetznerRobot.RescuePasswords = make(map[string]string)
		}
	}

	// Store VSwitch ID if available
	if selection.VSwitchSelection != nil && selection.VSwitchSelection.SelectedVSwitch != nil {
		cfg.HetznerRobot.VSwitchID = selection.VSwitchSelection.SelectedVSwitch.ID
	}

	// Store network configuration
	if selection.NetworkRanges != nil {
		cfg.HetznerRobot.NetworkConfig = &config.HetznerRobotNetworkConfig{
			Location:                    "nbg1",                                                       // Default location
			NetworkZone:                 "eu-central",                                                 // Default network zone
			ClusterNetworkIPRange:       selection.NetworkRanges.ClusterNetworkIPRange,
			ClusterVSwitchSubnetIPRange: selection.NetworkRanges.ClusterVSwitchSubnetIPRange,
			ClusterSubnetIPRange:        selection.NetworkRanges.ClusterSubnetIPRange,
		}
	}

	return nil
}

// determineServerRole determines the role of a server based on its position and cluster type
func determineServerRole(index int, clusterType string, totalServers int) string {
	switch clusterType {
	case "single-node":
		return "control-plane-worker"
	case "multi-node":
		if index == 0 {
			return "control-plane"
		}
		return "worker"
	case "ha-cluster":
		if index < 3 {
			return "control-plane"
		}
		return "worker"
	default:
		return "worker"
	}
}

// storeProvisionOutputs stores the provision phase outputs (like discovered disks) in the config
func storeProvisionOutputs(cfg *config.CreateConfig, outputs map[string]interface{}) error {
	if cfg.HetznerRobot == nil {
		return fmt.Errorf("HetznerRobot config is nil")
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("💾"),
		styles.DescriptionStyle.Render("Storing provision outputs in config..."))

	// Extract discovered_disks output
	// The structure from Terraform is: discovered_disks = { "server_id" = { "value" = "/dev/sda", "sensitive" = false } }
	if discoveredDisksRaw, ok := outputs["discovered_disks"]; ok {
		if discoveredDisksMap, ok := discoveredDisksRaw.(map[string]interface{}); ok {
			// Extract the "value" from the Terraform output structure
			if valueMap, ok := discoveredDisksMap["value"].(map[string]interface{}); ok {
				// Update each server with its discovered disk
				for i := range cfg.HetznerRobot.SelectedServers {
					server := &cfg.HetznerRobot.SelectedServers[i]

					// Get disk for this server
					if diskPath, ok := valueMap[server.ID].(string); ok && diskPath != "" {
						server.InstallationDisk = diskPath
						fmt.Printf("%s %s %s: %s\n",
							styles.CommandStyle.Render("✅"),
							styles.DescriptionStyle.Render("Stored disk for"),
							styles.CommandStyle.Render(server.Name),
							styles.CommandStyle.Render(diskPath))
					}
				}
			}
		}
	} else {
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.DescriptionStyle.Render("Warning: No discovered_disks output found"))
	}

	return nil
}

// storeNetworkDiscovery stores the discovered network information in the config
func storeNetworkDiscovery(cfg *config.CreateConfig, discovery *TalosDiscoveryResult) error {
	if cfg.HetznerRobot == nil {
		return fmt.Errorf("HetznerRobot config is nil")
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("💾"),
		styles.DescriptionStyle.Render("Storing discovered network information in config..."))

	// Calculate the vSwitch gateway (first IP in the vSwitch subnet)
	// This is the same for all servers since they all connect to the same vSwitch
	var vswitchGateway string
	if cfg.HetznerRobot.NetworkConfig != nil && cfg.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange != "" {
		gw, err := CalculateGatewayFromCIDR(cfg.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange)
		if err == nil {
			vswitchGateway = gw
			fmt.Printf("%s %s %s\n",
				styles.CommandStyle.Render("🔗"),
				styles.DescriptionStyle.Render("VSwitch Gateway:"),
				styles.CommandStyle.Render(vswitchGateway))
		} else {
			fmt.Printf("%s %s: %v\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render("Warning: Failed to calculate vSwitch gateway"),
				err)
		}
	}

	// Update each server with discovered network info
	for i := range cfg.HetznerRobot.SelectedServers {
		server := &cfg.HetznerRobot.SelectedServers[i]

		// Get discovery info for this server
		serverInfo, exists := discovery.ServersInfo[server.ID]
		if !exists || !serverInfo.IsOnline {
			fmt.Printf("%s %s %s - %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render("Warning: No network info for server"),
				styles.CommandStyle.Render(server.Name),
				styles.DescriptionStyle.Render("skipping"))
			continue
		}

		// Store public gateway
		if serverInfo.PublicGW != "" {
			server.PublicIPv4Gateway = serverInfo.PublicGW
		}

		// Store private gateway (vSwitch gateway - same for all servers)
		if vswitchGateway != "" {
			server.PrivateIPv4Gateway = vswitchGateway
		}

		// Store public and private addresses with CIDR notation
		if serverInfo.PublicCIDR != "" {
			server.PublicAddressSubnet = serverInfo.PublicCIDR
		}

		if serverInfo.PrivateCIDR != "" {
			server.PrivateAddressSubnet = serverInfo.PrivateCIDR
		}

		// Store public network interface name
		if serverInfo.PublicInterface != "" {
			server.PublicNetworkInterface = serverInfo.PublicInterface
		}

		// Note: PrivateNetworkInterface field may need to be added to HetznerRobotServer config struct
		// For now, we only store the public interface which is used in Talos config generation

		fmt.Printf("%s %s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.DescriptionStyle.Render("Stored network info for"),
			styles.CommandStyle.Render(server.Name))
	}

	return nil
}

// generatePrivateIP generates a private IP address from a subnet CIDR and host offset
// Example: generatePrivateIP("172.20.1.0/24", 10) returns "172.20.1.10"
func generatePrivateIP(subnetCIDR string, hostOffset int) string {
	ip, ipNet, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return ""
	}

	// Get the base IP
	baseIP := ip.Mask(ipNet.Mask)

	// Add the offset to the last octet
	newIP := make(net.IP, len(baseIP))
	copy(newIP, baseIP)

	// Handle IPv4
	if len(newIP) == 4 || (len(newIP) == 16 && newIP.To4() != nil) {
		ipv4 := newIP.To4()
		// Add offset to the last octet (simple case, doesn't handle overflow)
		ipv4[3] += byte(hostOffset)
		return ipv4.String()
	}

	return ""
}

// generateRandomConfirmationCode generates a random 6-character lowercase string
func generateRandomConfirmationCode() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	const length = 6

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i := range bytes {
		bytes[i] = charset[bytes[i]%byte(len(charset))]
	}

	return string(bytes), nil
}

// showCriticalWarningAndConfirm displays a critical warning about the destructive nature of the operation
// and requires user confirmation before proceeding
func showCriticalWarningAndConfirm(clusterName string) bool {
	// Generate random confirmation code
	confirmCode, err := generateRandomConfirmationCode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Error generating confirmation code: %v", err)))
		return false
	}

	// Create a prominent warning box
	warningBox := []string{
		"",
		"╔═══════════════════════════════════════════════════════════════════════════════╗",
		"║                                                                               ║",
		"║                         ⚠️  CRITICAL WARNING ⚠️                                ║",
		"║                                                                               ║",
		"║   This Hetzner Robot cluster creation process is DESTRUCTIVE and can         ║",
		"║   only be run ONCE per cluster configuration.                                ║",
		"║                                                                               ║",
		"║   WHY THIS MATTERS:                                                           ║",
		"║   • This script manages a mix of bare metal servers AND cloud resources       ║",
		"║   • Running 'terraform destroy' before 'terraform init' RESETS ALL STATE      ║",
		"║   • Running this script again will COMPLETELY DESTROY the existing cluster    ║",
		"║   • All data, configurations, and workloads will be PERMANENTLY LOST          ║",
		"║                                                                               ║",
		"║   WHAT WILL HAPPEN:                                                           ║",
		"║   1. Bare metal servers will be wiped and reinstalled with Talos Linux        ║",
		"║   2. Cloud networks and load balancers will be destroyed and recreated        ║",
		"║   3. All Kubernetes state will be lost                                        ║",
		"║   4. All applications and data will be deleted                                ║",
		"║                                                                               ║",
		"║   ⚠️  DO NOT PROCEED if this cluster is already in use!                        ║",
		"║   ⚠️  BACKUP ALL DATA before running this script again!                        ║",
		"║                                                                               ║",
		"╚═══════════════════════════════════════════════════════════════════════════════╝",
		"",
	}

	// Print the warning box in red
	for _, line := range warningBox {
		fmt.Printf("\033[1;31m%s\033[0m\n", line)
	}

	// Show cluster name
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📋"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Cluster Name: %s", clusterName)))

	// Ask for confirmation with random code
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("❓"),
		styles.DescriptionStyle.Render(fmt.Sprintf("To proceed, type the following code exactly: %s",
			"\033[1;33m"+confirmCode+"\033[0m")))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render(""),
		styles.DescriptionStyle.Render("(Type anything else to cancel)"))
	fmt.Print("\n> ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Error reading input: %v", err)))
		return false
	}

	// Trim whitespace and compare (exact match required)
	response = strings.TrimSpace(response)
	return response == confirmCode
}

// RunClusterCreationFlow executes the Hetzner Robot specific cluster creation flow
func RunClusterCreationFlow(cfg *config.CreateConfig) {
	// =====================================
	// PHASE 1: PROVISION (Bare Metal Setup)
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.TitleStyle.Render("PHASE 1: Bare Metal Provisioning"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("This phase will install Talos Linux on your bare metal servers"))

	// Build provision terraform files
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔨"),
		styles.HelpStyle.Render("Building provision Terraform files..."))
	if err := automation.BuildHetznerRobotProvisionFiles(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Error building provision Terraform files: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Provision Terraform files built successfully!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Files created in: .kibaship/%s/provision/", cfg.Name)))

	// Check if Terraform is installed
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Checking Terraform installation..."))
	if err := automation.CheckTerraformInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(err.Error()))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Terraform is installed and available"))

	// Run Terraform init for provision
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Initializing provision Terraform..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform init with S3 backend configuration"))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📄"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Backend: s3://%s/clusters/%s/provision.terraform.tfstate",
			cfg.TerraformState.S3Bucket, cfg.Name)))

	if err := automation.RunTerraformInit(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Provision Terraform init failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Provision Terraform initialization completed!"))

	// Run Terraform validate for provision
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Validating provision Terraform configuration..."))
	if err := automation.RunTerraformValidate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Provision Terraform validate failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Provision Terraform configuration is valid!"))

	// Run Terraform apply for provision
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Provisioning bare metal servers..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform apply -auto-approve"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⚠️"),
		styles.DescriptionStyle.Render("This will install Talos Linux on your servers and may take several minutes..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🕰️"),
		styles.DescriptionStyle.Render("Please wait while the servers are being provisioned..."))

	if err := automation.RunTerraformApply(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Provision Terraform apply failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Bare metal servers provisioned successfully!"))

	// Read Terraform outputs from provision phase
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📊"),
		styles.HelpStyle.Render("Reading provision Terraform outputs..."))
	provisionOutputs, err := automation.ReadProvisionTerraformOutputs(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Warning: Failed to read provision outputs: %v", err)))
		// Don't exit here, as outputs might not be critical for cloud phase
	} else {
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Read %d output(s) from provision phase", len(provisionOutputs))))

		// Store disk discovery outputs in config
		if err := storeProvisionOutputs(cfg, provisionOutputs); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render(fmt.Sprintf("Warning: Failed to store provision outputs: %v", err)))
		}
	}

	// =====================================
	// PHASE 2: CLOUD (Networking & Load Balancers)
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("☁️"),
		styles.TitleStyle.Render("PHASE 2: Cloud Infrastructure Setup"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("This phase will create Hetzner Cloud network and load balancers"))

	// Build cloud terraform files
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔨"),
		styles.HelpStyle.Render("Building cloud Terraform files..."))
	if err := automation.BuildHetznerRobotCloudFiles(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Error building cloud Terraform files: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cloud Terraform files built successfully!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Files created in: .kibaship/%s/cloud/", cfg.Name)))

	// Run Terraform init for cloud
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Initializing cloud Terraform..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform init with S3 backend configuration"))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📄"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Backend: s3://%s/clusters/%s/cloud.terraform.tfstate",
			cfg.TerraformState.S3Bucket, cfg.Name)))

	if err := automation.RunCloudTerraformInit(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Cloud Terraform init failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cloud Terraform initialization completed!"))

	// Run Terraform validate for cloud
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Validating cloud Terraform configuration..."))
	if err := automation.RunCloudTerraformValidate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Cloud Terraform validate failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cloud Terraform configuration is valid!"))

	// Run Terraform apply for cloud
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Creating cloud infrastructure..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform apply -auto-approve"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⚠️"),
		styles.DescriptionStyle.Render("This will create Hetzner Cloud network and load balancers..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🕰️"),
		styles.DescriptionStyle.Render("Please wait while the cloud infrastructure is being created..."))

	if err := automation.RunCloudTerraformApply(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Cloud Terraform apply failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cloud infrastructure created successfully!"))

	// =====================================
	// PHASE 3: SERVER DISCOVERY (Wait for servers and discover network info)
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🔍"),
		styles.TitleStyle.Render("PHASE 3: Server Network Discovery"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("This phase will wait for servers to come online and discover their network configuration"))

	// Wait for all servers to come back online (with 10 minute timeout)
	ctx := context.Background()
	if err := WaitForServersOnline(ctx, cfg, 10*time.Minute); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Server readiness check failed: %v", err)))
		os.Exit(1)
	}

	// Discover network information from each server via Talos
	discoveryResult, err := DiscoverServerNetworks(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Network discovery failed: %v", err)))
		os.Exit(1)
	}

	if !discoveryResult.Success {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.CommandStyle.Render("Warning: Network discovery completed with some failures"))
		// Don't exit, as we may be able to proceed
	}

	// Store discovered network information in config for subsequent steps
	if err := storeNetworkDiscovery(cfg, discoveryResult); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.CommandStyle.Render(fmt.Sprintf("Warning: Failed to store network discovery: %v", err)))
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Server network discovery completed!"))

	// =====================================
	// PHASE 4: TALOS BOOTSTRAP (Kubernetes Cluster)
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🎯"),
		styles.TitleStyle.Render("PHASE 4: Talos Kubernetes Bootstrap"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("This phase will bootstrap the Kubernetes cluster using Talos"))

	// Build Talos terraform files
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔨"),
		styles.HelpStyle.Render("Building Talos bootstrap Terraform files..."))
	if err := automation.BuildHetznerRobotTalosFiles(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Error building Talos Terraform files: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Talos Terraform files built successfully!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Files created in: .kibaship/%s/talos/", cfg.Name)))

	// Run Terraform init for Talos
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Initializing Talos bootstrap Terraform..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform init with S3 backend configuration"))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📄"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Backend: s3://%s/clusters/%s/bare-metal-talos-bootstrap/terraform.tfstate",
			cfg.TerraformState.S3Bucket, cfg.Name)))

	if err := automation.RunTalosTerraformInit(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Talos Terraform init failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Talos Terraform initialization completed!"))

	// Run Terraform validate for Talos
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Validating Talos bootstrap Terraform configuration..."))
	if err := automation.RunTalosTerraformValidate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Talos Terraform validate failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Talos Terraform configuration is valid!"))

	// Run Terraform apply for Talos bootstrap
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Bootstrapping Kubernetes cluster..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform apply -auto-approve"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⚠️"),
		styles.DescriptionStyle.Render("This will configure all nodes and bootstrap the Kubernetes control plane..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🕰️"),
		styles.DescriptionStyle.Render("Please wait while the Kubernetes cluster is being bootstrapped..."))

	if err := automation.RunTalosTerraformApply(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Talos Terraform apply failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Kubernetes cluster bootstrapped successfully!"))

	// Read Talos Terraform outputs
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📊"),
		styles.HelpStyle.Render("Reading Talos Terraform outputs..."))
	talosOutputs, err := automation.ReadTalosTerraformOutputs(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Warning: Failed to read Talos outputs: %v", err)))
	} else {
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Read %d output(s) from Talos bootstrap phase", len(talosOutputs))))

		// Display cluster access information
		if clusterInfo, ok := talosOutputs["cluster_info"]; ok {
			fmt.Printf("\n%s %s\n",
				styles.CommandStyle.Render("🔐"),
				styles.DescriptionStyle.Render("Cluster Access Information:"))
			fmt.Printf("   %s\n", styles.DescriptionStyle.Render(fmt.Sprintf("Cluster Endpoint: %v", clusterInfo)))
		}
	}

	// =====================================
	// COMPLETION
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🎉"),
		styles.TitleStyle.Render("Hetzner Robot Kubernetes Cluster Complete!"))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📋"),
		styles.DescriptionStyle.Render("Setup Summary:"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("✅ Bare metal servers provisioned with Talos Linux"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("✅ Hetzner Cloud network and load balancers configured"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("✅ Server network configuration discovered"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("✅ Kubernetes cluster bootstrapped and ready"))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Access your cluster:"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render(fmt.Sprintf("Kubeconfig: .kibaship/%s/talos/kubeconfig", cfg.Name)))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render(fmt.Sprintf("Talosconfig: .kibaship/%s/talos/talosconfig", cfg.Name)))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🚀"),
		styles.DescriptionStyle.Render("Your Kubernetes cluster is now ready for workload deployment!"))
}
