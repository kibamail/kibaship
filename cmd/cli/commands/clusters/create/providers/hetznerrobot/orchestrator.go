package hetznerrobot

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"path/filepath"
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

	// Store VSwitch ID and VLAN ID if available
	if selection.VSwitchSelection != nil && selection.VSwitchSelection.SelectedVSwitch != nil {
		cfg.HetznerRobot.VSwitchID = selection.VSwitchSelection.SelectedVSwitch.ID
		cfg.HetznerRobot.VLANID = selection.VSwitchSelection.SelectedVSwitch.VLAN
		fmt.Printf("%s %s %d\n",
			styles.CommandStyle.Render("🔗"),
			styles.DescriptionStyle.Render("Stored VLAN ID:"),
			cfg.HetznerRobot.VLANID)
	}

	// Store network configuration
	if selection.NetworkRanges != nil {
		cfg.HetznerRobot.NetworkConfig = &config.HetznerRobotNetworkConfig{
			Location:                    "nbg1",       // Default location
			NetworkZone:                 "eu-central", // Default network zone
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

// Cloud phase removed: Talos endpoint uses DNS-based HA; no cloud outputs needed.

// storeProvisionOutputs stores the provision phase outputs (like discovered disks) in the config
func storeProvisionOutputs(cfg *config.CreateConfig, outputs map[string]interface{}) error {
	if cfg.HetznerRobot == nil {
		return fmt.Errorf("HetznerRobot config is nil")
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("💾"),
		styles.DescriptionStyle.Render("Storing provision outputs in config..."))

	// FIX: Extract disk discovery outputs per server
	// Terraform outputs individual server disk discovery: server_<ID>_disk_discovery
	// Structure: server_2664303_disk_discovery = {
	//   "value" = {
	//     "all_devices" = [...]
	//     "os_installation" = {
	//       "device" = "nvme0n1"
	//       "disk_by_id" = "nvme-SAMSUNG_..."
	//       "disk_by_id_path" = "/dev/disk/by-id/nvme-SAMSUNG_..."
	//       "full_path" = "/dev/nvme0n1"
	//     }
	//   }
	// }
	for i := range cfg.HetznerRobot.SelectedServers {
		server := &cfg.HetznerRobot.SelectedServers[i]

		// Look for server-specific disk discovery output
		diskOutputKey := fmt.Sprintf("server_%s_disk_discovery", server.ID)

		if diskDiscoveryRaw, ok := outputs[diskOutputKey]; ok {
			// Extract the value from Terraform output structure
			if diskDiscoveryMap, ok := diskDiscoveryRaw.(map[string]interface{}); ok {
				if valueMap, ok := diskDiscoveryMap["value"].(map[string]interface{}); ok {
					// Extract os_installation nested object
					var installationDiskByID string
					if talosInstallation, ok := valueMap["os_installation"].(map[string]interface{}); ok {
						// Prefer disk_by_id_path, fallback to full_path
						var diskPath string
						if diskByIDPath, ok := talosInstallation["disk_by_id_path"].(string); ok && diskByIDPath != "" {
							diskPath = diskByIDPath
						} else if fullPath, ok := talosInstallation["full_path"].(string); ok && fullPath != "" {
							diskPath = fullPath
						}

						if diskPath != "" {
							server.InstallationDisk = diskPath
							fmt.Printf("%s %s %s: %s\n",
								styles.CommandStyle.Render("✅"),
								styles.DescriptionStyle.Render("Stored disk for"),
								styles.CommandStyle.Render(server.Name),
								styles.CommandStyle.Render(diskPath))

							// Extract disk_by_id for filtering storage disks
							if diskByID, ok := talosInstallation["disk_by_id"].(string); ok {
								installationDiskByID = diskByID
							}
						} else {
							fmt.Printf("%s %s %s (no disk path in os_installation)\n",
								styles.CommandStyle.Render("⚠️"),
								styles.DescriptionStyle.Render("Warning: Could not extract disk path for"),
								styles.CommandStyle.Render(server.Name))
						}
					} else {
						fmt.Printf("%s %s %s (no os_installation object)\n",
							styles.CommandStyle.Render("⚠️"),
							styles.DescriptionStyle.Render("Warning: Could not extract os_installation for"),
							styles.CommandStyle.Render(server.Name))
					}

					// Extract storage disks (all devices except the installation disk)
					if allDevicesRaw, ok := valueMap["all_devices"]; ok {
						if allDevices, ok := allDevicesRaw.([]interface{}); ok {
							var storageDisks []config.StorageDisk
							storageIndex := 0

							for _, deviceRaw := range allDevices {
								if device, ok := deviceRaw.(map[string]interface{}); ok {
									// Extract disk_by_id
									diskByID, _ := device["disk_by_id"].(string)

									// Skip if this is the installation disk
									if diskByID == installationDiskByID {
										continue
									}

									// Create storage disk entry with disk-by-id path
									if diskByID != "" {
										storageDisk := config.StorageDisk{
											Name: fmt.Sprintf("storage-disk-%d", storageIndex),
											Path: fmt.Sprintf("/dev/disk/by-id/%s", diskByID),
										}
										storageDisks = append(storageDisks, storageDisk)
										storageIndex++
									}
								}
							}

							server.StorageDisks = storageDisks
							if len(storageDisks) > 0 {
								fmt.Printf("%s %s %s: %d storage disk(s)\n",
									styles.CommandStyle.Render("✅"),
									styles.DescriptionStyle.Render("Stored storage disks for"),
									styles.CommandStyle.Render(server.Name),
									len(storageDisks))
								for _, disk := range storageDisks {
									fmt.Printf("  - %s: %s\n",
										styles.CommandStyle.Render(disk.Name),
										styles.DescriptionStyle.Render(disk.Path))
								}
							} else {
								fmt.Printf("%s %s %s (no additional disks found)\n",
									styles.CommandStyle.Render("ℹ️"),
									styles.DescriptionStyle.Render("No storage disks for"),
									styles.CommandStyle.Render(server.Name))
							}
						}
					}
				}
			}
		} else {
			fmt.Printf("%s %s %s (key: %s)\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render("Warning: No disk discovery output found for"),
				styles.CommandStyle.Render(server.Name),
				diskOutputKey)
		}
	}

	return nil
}

// storeSSHKeypair writes the SSH keypair to .kibaship/<cluster>/credentials/.ssh/
func storeSSHKeypair(cfg *config.CreateConfig, outputs map[string]interface{}) error {
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🔑"),
		styles.DescriptionStyle.Render("Storing SSH keypair to filesystem..."))

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create .kibaship/<cluster>/credentials/.ssh directory relative to pwd
	sshDir := filepath.Join(cwd, ".kibaship", cfg.Name, "credentials", ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Extract SSH private key from outputs
	var privateKey string
	if sshPrivateKeyRaw, ok := outputs["ssh_private_key"]; ok {
		if sshPrivateKeyMap, ok := sshPrivateKeyRaw.(map[string]interface{}); ok {
			if value, ok := sshPrivateKeyMap["value"].(string); ok {
				privateKey = value
			}
		}
	}

	if privateKey == "" {
		return fmt.Errorf("ssh_private_key not found in outputs")
	}

	// Extract SSH public key from outputs
	var publicKey string
	if sshPublicKeyRaw, ok := outputs["ssh_public_key"]; ok {
		if sshPublicKeyMap, ok := sshPublicKeyRaw.(map[string]interface{}); ok {
			if value, ok := sshPublicKeyMap["value"].(string); ok {
				publicKey = value
			}
		}
	}

	if publicKey == "" {
		return fmt.Errorf("ssh_public_key not found in outputs")
	}

	// Write private key to id_rsa
	privateKeyPath := filepath.Join(sshDir, "id_rsa")
	if err := os.WriteFile(privateKeyPath, []byte(privateKey), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	fmt.Printf("%s %s: %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("Wrote SSH private key to"),
		styles.CommandStyle.Render(privateKeyPath))

	// Write public key to id_rsa.pub
	publicKeyPath := filepath.Join(sshDir, "id_rsa.pub")
	if err := os.WriteFile(publicKeyPath, []byte(publicKey), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	fmt.Printf("%s %s: %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("Wrote SSH public key to"),
		styles.CommandStyle.Render(publicKeyPath))

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

	// DEBUG: Log discovery result structure
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.CommandStyle.Render("DEBUG: Discovery Result:"))
	fmt.Printf("  %s %d\n",
		styles.DescriptionStyle.Render("Total servers in discovery:"),
		len(discovery.ServersInfo))
	fmt.Printf("  %s %v\n",
		styles.DescriptionStyle.Render("Discovery success:"),
		discovery.Success)

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

		fmt.Printf("\n%s %s %s (ID: %s)\n",
			styles.CommandStyle.Render("🔍"),
			styles.DescriptionStyle.Render("Processing server:"),
			styles.CommandStyle.Render(server.Name),
			styles.CommandStyle.Render(server.ID))

		// Get discovery info for this server
		serverInfo, exists := discovery.ServersInfo[server.ID]
		if !exists {
			fmt.Printf("%s %s %s - %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render("Warning: No discovery info found for server"),
				styles.CommandStyle.Render(server.Name),
				styles.DescriptionStyle.Render("key not in ServersInfo map"))
			continue
		}

		if !serverInfo.IsOnline {
			fmt.Printf("%s %s %s - %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render("Warning: Server"),
				styles.CommandStyle.Render(server.Name),
				styles.DescriptionStyle.Render("is marked as offline"))
			continue
		}

		// DEBUG: Log what we received from discovery
		fmt.Printf("  %s\n", styles.DescriptionStyle.Render("Discovered values:"))
		fmt.Printf("    PublicInterface: '%s'\n", serverInfo.PublicInterface)
		fmt.Printf("    PublicGW: '%s'\n", serverInfo.PublicGW)
		fmt.Printf("    PublicCIDR: '%s'\n", serverInfo.PublicCIDR)
		fmt.Printf("    PrivateInterface: '%s'\n", serverInfo.PrivateInterface)
		fmt.Printf("    PrivateGW: '%s'\n", serverInfo.PrivateGW)
		fmt.Printf("    PrivateCIDR: '%s'\n", serverInfo.PrivateCIDR)

		// Store public gateway
		if serverInfo.PublicGW != "" {
			server.PublicIPv4Gateway = serverInfo.PublicGW
			fmt.Printf("  %s PublicIPv4Gateway = %s\n",
				styles.CommandStyle.Render("✅"),
				serverInfo.PublicGW)
		} else {
			fmt.Printf("  %s PublicIPv4Gateway (empty, not stored)\n",
				styles.CommandStyle.Render("⚠️"))
		}

		// Store private gateway (vSwitch gateway - same for all servers)
		if vswitchGateway != "" {
			server.PrivateIPv4Gateway = vswitchGateway
			fmt.Printf("  %s PrivateIPv4Gateway = %s\n",
				styles.CommandStyle.Render("✅"),
				vswitchGateway)
		} else {
			fmt.Printf("  %s PrivateIPv4Gateway (vSwitch gateway not calculated)\n",
				styles.CommandStyle.Render("⚠️"))
		}

		// Store public and private addresses with CIDR notation
		if serverInfo.PublicCIDR != "" {
			server.PublicAddressSubnet = serverInfo.PublicCIDR
			fmt.Printf("  %s PublicAddressSubnet = %s\n",
				styles.CommandStyle.Render("✅"),
				serverInfo.PublicCIDR)
		} else {
			fmt.Printf("  %s PublicAddressSubnet (empty, not stored)\n",
				styles.CommandStyle.Render("⚠️"))
		}

		// Calculate PrivateAddressSubnet manually from server.PrivateIP + vSwitch CIDR
		// Since the VLAN interface doesn't exist yet, we can't discover it via Talos
		// Format: {server.PrivateIP}/{CIDR-from-vSwitch-subnet}
		// Example: if PrivateIP="172.21.224.10" and vSwitch subnet="172.21.224.0/20", result="172.21.224.10/20"
		if server.PrivateIP != "" && cfg.HetznerRobot.NetworkConfig != nil && cfg.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange != "" {
			// Extract CIDR mask from vSwitch subnet (e.g., "/20" from "172.21.224.0/20")
			_, ipNet, err := net.ParseCIDR(cfg.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange)
			if err == nil {
				maskSize, _ := ipNet.Mask.Size()
				privateAddressSubnet := fmt.Sprintf("%s/%d", server.PrivateIP, maskSize)
				server.PrivateAddressSubnet = privateAddressSubnet
				fmt.Printf("  %s PrivateAddressSubnet = %s (calculated from PrivateIP + vSwitch CIDR)\n",
					styles.CommandStyle.Render("✅"),
					privateAddressSubnet)
			} else {
				fmt.Printf("  %s PrivateAddressSubnet (failed to calculate: %v)\n",
					styles.CommandStyle.Render("⚠️"), err)
			}
		} else if serverInfo.PrivateCIDR != "" {
			// Fallback: use discovered private CIDR if available (for future when VLAN is already configured)
			server.PrivateAddressSubnet = serverInfo.PrivateCIDR
			fmt.Printf("  %s PrivateAddressSubnet = %s (discovered)\n",
				styles.CommandStyle.Render("✅"),
				serverInfo.PrivateCIDR)
		} else {
			fmt.Printf("  %s PrivateAddressSubnet (not calculated: PrivateIP=%q, vSwitch subnet=%q)\n",
				styles.CommandStyle.Render("⚠️"),
				server.PrivateIP,
				func() string {
					if cfg.HetznerRobot.NetworkConfig != nil {
						return cfg.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange
					}
					return ""
				}())
		}

		// Store public network interface name
		if serverInfo.PublicInterface != "" {
			server.PublicNetworkInterface = serverInfo.PublicInterface
			fmt.Printf("  %s PublicNetworkInterface = %s\n",
				styles.CommandStyle.Render("✅"),
				serverInfo.PublicInterface)
		} else {
			fmt.Printf("  %s PublicNetworkInterface (empty, not stored)\n",
				styles.CommandStyle.Render("⚠️"))
		}

		// Note: PrivateNetworkInterface field may need to be added to HetznerRobotServer config struct
		// For now, we only store the public interface which is used in Talos config generation

		fmt.Printf("%s %s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.DescriptionStyle.Render("Completed storing network info for"),
			styles.CommandStyle.Render(server.Name))
	}

	return nil
}

// storeTalosCredentials writes Talos credentials to files
func storeTalosCredentials(cfg *config.CreateConfig, outputs map[string]interface{}) error {
	talosDir := filepath.Join(".kibaship", cfg.Name, "talos")

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("💾"),
		styles.DescriptionStyle.Render("Writing cluster credentials to files..."))

	// Extract and write kubeconfig
	if kubeconfigRaw, ok := outputs["kubeconfig"]; ok {
		if kubeconfigMap, ok := kubeconfigRaw.(map[string]interface{}); ok {
			if kubeconfigValue, ok := kubeconfigMap["value"].(string); ok {
				kubeconfigPath := filepath.Join(talosDir, "kubeconfig")
				if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigValue), 0600); err != nil {
					return fmt.Errorf("failed to write kubeconfig: %w", err)
				}
				fmt.Printf("%s %s %s\n",
					styles.CommandStyle.Render("✅"),
					styles.DescriptionStyle.Render("Kubeconfig written to:"),
					styles.CommandStyle.Render(kubeconfigPath))
			}
		}
	}

	// Extract and write talosconfig
	if talosconfigRaw, ok := outputs["talos_config"]; ok {
		if talosconfigMap, ok := talosconfigRaw.(map[string]interface{}); ok {
			if talosconfigValue, ok := talosconfigMap["value"].(string); ok {
				talosconfigPath := filepath.Join(talosDir, "talosconfig")
				if err := os.WriteFile(talosconfigPath, []byte(talosconfigValue), 0600); err != nil {
					return fmt.Errorf("failed to write talosconfig: %w", err)
				}
				fmt.Printf("%s %s %s\n",
					styles.CommandStyle.Render("✅"),
					styles.DescriptionStyle.Render("Talosconfig written to:"),
					styles.CommandStyle.Render(talosconfigPath))
			}
		}
	}

	// Create credentials subdirectory for machine configs
	clusterDir := filepath.Join(".kibaship", cfg.Name)
	credentialsDir := filepath.Join(clusterDir, "credentials")
	if err := os.MkdirAll(credentialsDir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	// Extract and write control plane machine configurations
	if cpConfigsRaw, ok := outputs["control_plane_machine_configurations"]; ok {
		if cpConfigsMap, ok := cpConfigsRaw.(map[string]interface{}); ok {
			if valueMap, ok := cpConfigsMap["value"].(map[string]interface{}); ok {
				for serverID, configRaw := range valueMap {
					if configValue, ok := configRaw.(string); ok {
						configPath := filepath.Join(credentialsDir, fmt.Sprintf("controlplane-%s.yaml", serverID))
						if err := os.WriteFile(configPath, []byte(configValue), 0600); err != nil {
							return fmt.Errorf("failed to write control plane config for server %s: %w", serverID, err)
						}
						fmt.Printf("%s %s %s\n",
							styles.CommandStyle.Render("✅"),
							styles.DescriptionStyle.Render(fmt.Sprintf("Control plane config (server %s):", serverID)),
							styles.CommandStyle.Render(configPath))
					}
				}
			}
		}
	}

	// Extract and write worker machine configurations
	if workerConfigsRaw, ok := outputs["worker_machine_configurations"]; ok {
		if workerConfigsMap, ok := workerConfigsRaw.(map[string]interface{}); ok {
			if valueMap, ok := workerConfigsMap["value"].(map[string]interface{}); ok {
				for serverID, configRaw := range valueMap {
					if configValue, ok := configRaw.(string); ok {
						configPath := filepath.Join(credentialsDir, fmt.Sprintf("worker-node-%s.yaml", serverID))
						if err := os.WriteFile(configPath, []byte(configValue), 0600); err != nil {
							return fmt.Errorf("failed to write worker config for server %s: %w", serverID, err)
						}
						fmt.Printf("%s %s %s\n",
							styles.CommandStyle.Render("✅"),
							styles.DescriptionStyle.Render(fmt.Sprintf("Worker config (server %s):", serverID)),
							styles.CommandStyle.Render(configPath))
					}
				}
			}
		}
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
		"║   • This script manages destructive operations on bare metal servers          ║",
		"║   • Running 'terraform destroy' before 'terraform init' RESETS ALL STATE      ║",
		"║   • Running this script again will COMPLETELY DESTROY the existing cluster    ║",
		"║   • All data, configurations, and workloads will be PERMANENTLY LOST          ║",
		"║                                                                               ║",
		"║   WHAT WILL HAPPEN:                                                           ║",
		"║   1. Bare metal servers will be wiped and reinstalled with Talos Linux        ║",
		"║   2. All Kubernetes state will be lost                                        ║",
		"║   3. All applications and data will be deleted                                ║",
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

// ConfigureControlPlaneDNS prompts the user to configure DNS and waits for propagation
func ConfigureControlPlaneDNS(cfg *config.CreateConfig) error {
	// Calculate DNS name
	dnsName := fmt.Sprintf("kube.%s", cfg.Domain)

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🌐"),
		styles.TitleStyle.Render("DNS Configuration Required"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("For high availability, the cluster endpoint will use multiple A records"))

	// Collect control plane public IPs
	var controlPlaneIPs []string
	for _, server := range cfg.HetznerRobot.SelectedServers {
		if server.Role == "control-plane" {
			controlPlaneIPs = append(controlPlaneIPs, server.IP)
		}
	}

	if len(controlPlaneIPs) == 0 {
		return fmt.Errorf("no control plane servers found")
	}

	// Display DNS records table
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📋"),
		styles.TitleStyle.Render("Please create the following DNS A records:"))
	fmt.Printf("\n")
	fmt.Printf("  ┌─────────────────────────────────────────────────────────┐\n")
	fmt.Printf("  │ %-25s │ %-10s │ %-15s │\n", "NAME", "TYPE", "VALUE")
	fmt.Printf("  ├─────────────────────────────────────────────────────────┤\n")
	for _, ip := range controlPlaneIPs {
		fmt.Printf("  │ %-25s │ %-10s │ %-15s │\n", dnsName, "A", ip)
	}
	fmt.Printf("  └─────────────────────────────────────────────────────────┘\n")

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("ℹ️"),
		styles.DescriptionStyle.Render("These DNS records will provide high availability for your Kubernetes API endpoint"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("ℹ️"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Cluster endpoint will be: https://%s:6443", dnsName)))

	// Ask user to confirm DNS configuration
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("⏳"),
		styles.TitleStyle.Render("Press ENTER once you have created these DNS records..."))
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	// Start DNS propagation check
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.TitleStyle.Render("Checking DNS propagation..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⏱️"),
		styles.DescriptionStyle.Render("This will check every minute for up to 60 minutes"))

	timeout := time.After(60 * time.Minute)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Check immediately first
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.DescriptionStyle.Render("Initial DNS check..."))
	statuses, allResolved := checkDNSRecordsWithStatus(dnsName, controlPlaneIPs)
	displayDNSStatusTable(dnsName, statuses)

	if allResolved {
		fmt.Printf("\n%s %s\n",
			styles.TitleStyle.Render("✅"),
			styles.TitleStyle.Render("DNS records propagated successfully!"))
		return nil
	}

	startTime := time.Now()
	for {
		select {
		case <-timeout:
			return fmt.Errorf("DNS propagation timeout after 60 minutes")
		case <-ticker.C:
			elapsed := time.Since(startTime).Round(time.Minute)
			fmt.Printf("\n%s %s (elapsed: %v)\n",
				styles.CommandStyle.Render("🔄"),
				styles.DescriptionStyle.Render("Checking DNS records..."),
				elapsed)

			statuses, allResolved := checkDNSRecordsWithStatus(dnsName, controlPlaneIPs)
			displayDNSStatusTable(dnsName, statuses)

			if allResolved {
				fmt.Printf("\n%s %s\n",
					styles.TitleStyle.Render("✅"),
					styles.TitleStyle.Render("DNS records propagated successfully!"))
				return nil
			}
		}
	}
}

// DNSRecordStatus represents the status of a single DNS record
type DNSRecordStatus struct {
	IP       string
	Resolved bool
	Status   string
}

// checkDNSRecordsWithStatus verifies that all expected IPs are returned and returns detailed status
func checkDNSRecordsWithStatus(dnsName string, expectedIPs []string) ([]DNSRecordStatus, bool) {
	statuses := make([]DNSRecordStatus, 0, len(expectedIPs))

	// Lookup DNS
	ips, err := net.LookupIP(dnsName)

	// If DNS lookup failed completely
	if err != nil {
		for _, expectedIP := range expectedIPs {
			statuses = append(statuses, DNSRecordStatus{
				IP:       expectedIP,
				Resolved: false,
				Status:   "⏳ Not Found",
			})
		}
		return statuses, false
	}

	// Convert resolved IPs to strings
	resolvedIPMap := make(map[string]bool)
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			resolvedIPMap[ipv4.String()] = true
		}
	}

	// Check each expected IP
	allResolved := true
	for _, expectedIP := range expectedIPs {
		if resolvedIPMap[expectedIP] {
			statuses = append(statuses, DNSRecordStatus{
				IP:       expectedIP,
				Resolved: true,
				Status:   "✅ Resolved",
			})
		} else {
			statuses = append(statuses, DNSRecordStatus{
				IP:       expectedIP,
				Resolved: false,
				Status:   "⏳ Waiting",
			})
			allResolved = false
		}
	}

	return statuses, allResolved
}

// displayDNSStatusTable displays a table showing the status of each DNS record
func displayDNSStatusTable(dnsName string, statuses []DNSRecordStatus) {
	fmt.Printf("\n  ┌─────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("  │ %-30s │ %-10s │ %-15s │\n", "DNS NAME", "IP ADDRESS", "STATUS")
	fmt.Printf("  ├─────────────────────────────────────────────────────────────────┤\n")
	for _, status := range statuses {
		fmt.Printf("  │ %-30s │ %-10s │ %-15s │\n", dnsName, status.IP, status.Status)
	}
	fmt.Printf("  └─────────────────────────────────────────────────────────────────┘\n")
}

// checkDNSRecords verifies that all expected IPs are returned for the DNS name (backwards compatibility)
func checkDNSRecords(dnsName string, expectedIPs []string) bool {
	_, allResolved := checkDNSRecordsWithStatus(dnsName, expectedIPs)
	return allResolved
}

// ServerPingStatus represents the ping status of a server
type ServerPingStatus struct {
	Name   string
	IP     string
	Status string
	Online bool
}

// waitForServersPing pings all servers every 15 seconds until they're all up or timeout
func waitForServersPing(cfg *config.CreateConfig, timeout time.Duration) error {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🏓"),
		styles.TitleStyle.Render("Waiting for servers to respond to ping..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⏱️"),
		styles.DescriptionStyle.Render("Checking every 15 seconds for up to 5 minutes"))

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	timeoutChan := time.After(timeout)

	// Check immediately first
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.DescriptionStyle.Render("Initial ping check..."))
	statuses, allOnline := checkServersPing(cfg)
	displayServerPingTable(statuses)

	if allOnline {
		fmt.Printf("\n%s %s\n",
			styles.TitleStyle.Render("✅"),
			styles.TitleStyle.Render("All servers are online!"))
		return nil
	}

	startTime := time.Now()
	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("timeout: not all servers responded to ping after %v", timeout)
		case <-ticker.C:
			elapsed := time.Since(startTime).Round(time.Second)
			fmt.Printf("\n%s %s (elapsed: %v)\n",
				styles.CommandStyle.Render("🔄"),
				styles.DescriptionStyle.Render("Checking server connectivity..."),
				elapsed)

			statuses, allOnline := checkServersPing(cfg)
			displayServerPingTable(statuses)

			if allOnline {
				fmt.Printf("\n%s %s\n",
					styles.TitleStyle.Render("✅"),
					styles.TitleStyle.Render("All servers are online!"))
				return nil
			}
		}
	}
}

// checkServersPing pings all servers and returns their status
func checkServersPing(cfg *config.CreateConfig) ([]ServerPingStatus, bool) {
	statuses := make([]ServerPingStatus, 0, len(cfg.HetznerRobot.SelectedServers))
	allOnline := true

	for _, server := range cfg.HetznerRobot.SelectedServers {
		// Ping server using system ping command (1 packet, 2 second timeout)
		cmd := fmt.Sprintf("ping -c 1 -W 2 %s > /dev/null 2>&1", server.IP)
		err := automation.RunCommand(cmd, "", 5*time.Second)

		status := ServerPingStatus{
			Name: server.Name,
			IP:   server.IP,
		}

		if err == nil {
			status.Status = "✅ Online"
			status.Online = true
		} else {
			status.Status = "⏳ Waiting"
			status.Online = false
			allOnline = false
		}

		statuses = append(statuses, status)
	}

	return statuses, allOnline
}

// displayServerPingTable displays a table showing ping status of each server
func displayServerPingTable(statuses []ServerPingStatus) {
	fmt.Printf("\n  ┌──────────────────────────────────────────────────────────┐\n")
	fmt.Printf("  │ %-20s │ %-15s │ %-15s │\n", "SERVER NAME", "IP ADDRESS", "STATUS")
	fmt.Printf("  ├──────────────────────────────────────────────────────────┤\n")
	for _, status := range statuses {
		fmt.Printf("  │ %-20s │ %-15s │ %-15s │\n", status.Name, status.IP, status.Status)
	}
	fmt.Printf("  └──────────────────────────────────────────────────────────┘\n")
}

// waitForKubernetesAPI checks if Kubernetes API is accessible every 15 seconds
func waitForKubernetesAPI(kubeconfigPath string, timeout time.Duration) error {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔌"),
		styles.TitleStyle.Render("Waiting for Kubernetes API to be accessible..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⏱️"),
		styles.DescriptionStyle.Render("Checking every 15 seconds for up to 5 minutes"))

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	timeoutChan := time.After(timeout)

	// Check immediately first
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.DescriptionStyle.Render("Initial API check..."))

	isAccessible, message := checkKubernetesAPI(kubeconfigPath)
	displayKubernetesAPIStatus(isAccessible, message)

	if isAccessible {
		fmt.Printf("\n%s %s\n",
			styles.TitleStyle.Render("✅"),
			styles.TitleStyle.Render("Kubernetes API is accessible!"))
		return nil
	}

	startTime := time.Now()
	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("timeout: Kubernetes API not accessible after %v", timeout)
		case <-ticker.C:
			elapsed := time.Since(startTime).Round(time.Second)
			fmt.Printf("\n%s %s (elapsed: %v)\n",
				styles.CommandStyle.Render("🔄"),
				styles.DescriptionStyle.Render("Checking Kubernetes API..."),
				elapsed)

			isAccessible, message := checkKubernetesAPI(kubeconfigPath)
			displayKubernetesAPIStatus(isAccessible, message)

			if isAccessible {
				fmt.Printf("\n%s %s\n",
					styles.TitleStyle.Render("✅"),
					styles.TitleStyle.Render("Kubernetes API is accessible!"))
				return nil
			}
		}
	}
}

// checkKubernetesAPI attempts to connect to the Kubernetes API
func checkKubernetesAPI(kubeconfigPath string) (bool, string) {
	// Try kubectl get nodes to check API connectivity
	cmd := fmt.Sprintf("kubectl --kubeconfig %s get nodes -o json 2>&1", kubeconfigPath)
	err := automation.RunCommand(cmd, "", 10*time.Second)

	if err == nil {
		return true, "API Server responding"
	}
	return false, "API Server not ready"
}

// displayKubernetesAPIStatus displays the Kubernetes API status
func displayKubernetesAPIStatus(isAccessible bool, message string) {
	statusIcon := "⏳"
	if isAccessible {
		statusIcon = "✅"
	}

	fmt.Printf("\n  ┌────────────────────────────────────────────────────────┐\n")
	fmt.Printf("  │ %-30s │ %-20s │\n", "COMPONENT", "STATUS")
	fmt.Printf("  ├────────────────────────────────────────────────────────┤\n")
	fmt.Printf("  │ %-30s │ %s %-17s │\n", "Kubernetes API Server", statusIcon, message)
	fmt.Printf("  └────────────────────────────────────────────────────────┘\n")
}

// RunClusterCreationFlow executes the Hetzner Robot specific cluster creation flow
func RunClusterCreationFlow(cfg *config.CreateConfig) {
	// =====================================
	// PHASE 1: PROVISION (Bare Metal Setup)
	// =====================================
	var provisionOutputs map[string]interface{}

	if cfg.Resume == "ubuntu" || cfg.Resume == "microk8s" {
		// Resume mode: Skip provision phase and load outputs from storage
		fmt.Printf("\n%s %s\n",
			styles.TitleStyle.Render("⏭️"),
			styles.TitleStyle.Render("PHASE 1: Bare Metal Provisioning (SKIPPED - Resume Mode)"))
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("📊"),
			styles.HelpStyle.Render("Loading provision outputs from previous run..."))

		var err error
		provisionOutputs, err = automation.ReadProvisionTerraformOutputs(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n",
				styles.CommandStyle.Render("❌"),
				styles.CommandStyle.Render(fmt.Sprintf("Failed to read provision outputs: %v", err)))
			fmt.Fprintf(os.Stderr, "%s %s\n",
				styles.CommandStyle.Render("💡"),
				styles.DescriptionStyle.Render("Make sure PHASE 1 was completed successfully before resuming"))
			os.Exit(1)
		}

		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Loaded %d output(s) from provision phase", len(provisionOutputs))))

		// Store outputs in config
		if err := storeProvisionOutputs(cfg, provisionOutputs); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render(fmt.Sprintf("Warning: Failed to store provision outputs: %v", err)))
		}
	} else {
		// Normal mode: Run provision phase
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
		fmt.Printf("%s %s\n\n",
			styles.CommandStyle.Render("📝"),
			styles.DescriptionStyle.Render("Running: terraform init with local backend configuration"))

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
		var err error
		provisionOutputs, err = automation.ReadProvisionTerraformOutputs(cfg)
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

			// Store SSH keypair to filesystem
			if err := storeSSHKeypair(cfg, provisionOutputs); err != nil {
				fmt.Fprintf(os.Stderr, "%s %s\n",
					styles.CommandStyle.Render("⚠️"),
					styles.DescriptionStyle.Render(fmt.Sprintf("Warning: Failed to store SSH keypair: %v", err)))
			}
		}
	}

	// Wait for servers to come back online after OS installation (skip if resuming)
	if cfg.Resume == "" {
		if err := WaitForServersAvailability(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s %s\n",
				styles.CommandStyle.Render("❌"),
				styles.CommandStyle.Render(fmt.Sprintf("Servers did not come online: %v", err)))
			os.Exit(1)
		}
	} else {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("⏭️"),
			styles.DescriptionStyle.Render("Skipping server availability check (Resume Mode)"))
	}

	os.Exit(0)

	// =====================================
	// PHASE 2: UBUNTU SETUP (MicroK8s prereqs)
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🛠️"),
		styles.TitleStyle.Render("PHASE 2: Ubuntu Preflight for MicroK8s"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Preparing Ubuntu on each server (swap off, kernel mods, sysctl, packages)"))

	// Build ubuntu terraform files
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔨"),
		styles.HelpStyle.Render("Building Ubuntu Terraform files..."))
	if err := automation.BuildHetznerRobotUbuntuFiles(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Error building Ubuntu Terraform files: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Ubuntu Terraform files built successfully!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Files created in: .kibaship/%s/ubuntu/", cfg.Name)))

	// Init/Validate/Apply Ubuntu terraform
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Initializing Ubuntu Terraform..."))
	if err := automation.RunUbuntuTerraformInit(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Ubuntu Terraform init failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Ubuntu Terraform initialization completed!"))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Validating Ubuntu Terraform configuration..."))
	if err := automation.RunUbuntuTerraformValidate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Ubuntu Terraform validate failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Ubuntu Terraform configuration is valid!"))

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Applying Ubuntu setup on servers..."))
	if err := automation.RunUbuntuTerraformApply(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Ubuntu Terraform apply failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Ubuntu preflight completed on all servers!"))

	os.Exit(0)

	// Cloud phase removed for hetzner-robot: load balancers and cloud networking are not managed here

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
	// PHASE 3.5: DNS CONFIGURATION
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🌐"),
		styles.TitleStyle.Render("PHASE 3.5: Control Plane DNS Configuration"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Configuring DNS for high availability control plane access"))

	// Configure DNS and wait for propagation
	if err := ConfigureControlPlaneDNS(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("DNS configuration failed: %v", err)))
		os.Exit(1)
	}

	// =====================================
	// PHASE 4: TALOS BOOTSTRAP (Kubernetes Cluster)
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🎯"),
		styles.TitleStyle.Render("PHASE 4: Talos Kubernetes Bootstrap"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("This phase will bootstrap the Kubernetes cluster using Talos"))

	// Ensure TalosConfig is initialized
	if cfg.HetznerRobot.TalosConfig == nil {
		cfg.HetznerRobot.TalosConfig = &config.HetznerRobotTalosConfig{}
	}

	// Set ClusterEndpoint to DNS name (kube.{domain})
	dnsName := fmt.Sprintf("kube.%s", cfg.Domain)
	cfg.HetznerRobot.TalosConfig.ClusterEndpoint = fmt.Sprintf("https://%s:6443", dnsName)

	fmt.Printf("%s %s %s\n",
		styles.CommandStyle.Render("🔗"),
		styles.DescriptionStyle.Render("Cluster endpoint:"),
		styles.CommandStyle.Render(cfg.HetznerRobot.TalosConfig.ClusterEndpoint))

	// Populate VLAN ID and vSwitch subnet if available
	if cfg.HetznerRobot.VLANID != 0 {
		cfg.HetznerRobot.TalosConfig.VLANID = cfg.HetznerRobot.VLANID
	}
	if cfg.HetznerRobot.NetworkConfig != nil {
		cfg.HetznerRobot.TalosConfig.VSwitchSubnetIPRange = cfg.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange
	}

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
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform init with local backend configuration"))

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

		// Write credentials to files
		if err := storeTalosCredentials(cfg, talosOutputs); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render(fmt.Sprintf("Warning: Failed to write credentials: %v", err)))
		}

		// Display cluster access information
		if clusterInfo, ok := talosOutputs["cluster_info"]; ok {
			fmt.Printf("\n%s %s\n",
				styles.CommandStyle.Render("🔐"),
				styles.DescriptionStyle.Render("Cluster Access Information:"))
			fmt.Printf("   %s\n", styles.DescriptionStyle.Render(fmt.Sprintf("Cluster Endpoint: %v", clusterInfo)))
		}
	}

	// =====================================
	// PHASE 5: CLUSTER HEALTH CHECK
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🏥"),
		styles.TitleStyle.Render("PHASE 5: Cluster Health Check"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Waiting for servers to be fully operational and Kubernetes API to be accessible"))

	// Wait for servers to respond to ping
	if err := waitForServersPing(cfg, 5*time.Minute); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Server ping check failed: %v", err)))
		os.Exit(1)
	}

	// Wait for Kubernetes API to be accessible
	kubeconfigPath := filepath.Join(".kibaship", cfg.Name, "talos", "kubeconfig")
	if err := waitForKubernetesAPI(kubeconfigPath, 5*time.Minute); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Kubernetes API check failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cluster is healthy and ready!"))

	// =====================================
	// PHASE 6: BOOTSTRAP INFRASTRUCTURE
	// =====================================
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🏗️"),
		styles.TitleStyle.Render("PHASE 6: Bootstrap Infrastructure"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Installing Cilium, Longhorn, and operators"))

	// Build bootstrap terraform files
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔨"),
		styles.HelpStyle.Render("Building bootstrap Terraform files..."))
	if err := automation.BuildHetznerRobotBootstrapFiles(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Error building bootstrap Terraform files: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Bootstrap Terraform files built successfully!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Files created in: .kibaship/%s/bootstrap/", cfg.Name)))

	// Run Terraform init for bootstrap
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Initializing bootstrap Terraform..."))
	if err := automation.RunBootstrapTerraformInit(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Bootstrap Terraform init failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Bootstrap Terraform initialization completed!"))

	// Run Terraform validate for bootstrap
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Validating bootstrap Terraform configuration..."))
	if err := automation.RunBootstrapTerraformValidate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Bootstrap Terraform validate failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Bootstrap Terraform configuration is valid!"))

	// Run Terraform apply for bootstrap
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Applying bootstrap infrastructure..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Installing: Cilium, Longhorn, MySQL Operator"))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🕰️"),
		styles.DescriptionStyle.Render("This may take several minutes..."))

	if err := automation.RunBootstrapTerraformApply(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Bootstrap Terraform apply failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Bootstrap infrastructure installed successfully!"))

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
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("✅ Cilium CNI installed and configured"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("✅ Longhorn storage system deployed"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("✅ MySQL Operator installed"))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Access your cluster:"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render(fmt.Sprintf("Kubeconfig: .kibaship/%s/talos/kubeconfig", cfg.Name)))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render(fmt.Sprintf("Talosconfig: .kibaship/%s/talos/talosconfig", cfg.Name)))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render(fmt.Sprintf("Machine configs: .kibaship/%s/credentials/", cfg.Name)))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🚀"),
		styles.DescriptionStyle.Render("Your Kubernetes cluster is now ready for workload deployment!"))
}

// WaitForServersAvailability checks if all servers are reachable via SSH after reboot
// It pings all servers every 15 seconds for up to 5 minutes
func WaitForServersAvailability(cfg *config.CreateConfig) error {
	if cfg.HetznerRobot == nil || len(cfg.HetznerRobot.SelectedServers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("⏳"),
		styles.TitleStyle.Render("Waiting for servers to come online after OS installation..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📡"),
		styles.DescriptionStyle.Render("Checking SSH connectivity every 15 seconds (timeout: 5 minutes)"))

	timeout := 5 * time.Minute
	checkInterval := 15 * time.Second
	deadline := time.Now().Add(timeout)

	servers := cfg.HetznerRobot.SelectedServers
	serverStatus := make(map[string]bool) // Track which servers are online
	totalServers := len(servers)

	for time.Now().Before(deadline) {
		onlineCount := 0
		allOnline := true

		for _, server := range servers {
			if serverStatus[server.ID] {
				// Server already confirmed online
				onlineCount++
				continue
			}

			// Try to connect to SSH port
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", server.IP), 5*time.Second)
			if err == nil {
				conn.Close()
				serverStatus[server.ID] = true
				onlineCount++
				fmt.Printf("%s %s\n",
					styles.TitleStyle.Render("✅"),
					styles.DescriptionStyle.Render(fmt.Sprintf("Server %s (%s) is now online", server.Name, server.IP)))
			} else {
				allOnline = false
			}
		}

		if allOnline {
			fmt.Printf("\n%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render(fmt.Sprintf("All %d servers are online and ready!", totalServers)))
			return nil
		}

		// Show progress
		remaining := time.Until(deadline)
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("⏳"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Status: %d/%d servers online | Time remaining: %s",
				onlineCount, totalServers, remaining.Round(time.Second))))

		// Wait before next check
		time.Sleep(checkInterval)
	}

	// Timeout reached
	onlineCount := 0
	var offlineServers []string
	for _, server := range servers {
		if serverStatus[server.ID] {
			onlineCount++
		} else {
			offlineServers = append(offlineServers, fmt.Sprintf("%s (%s)", server.Name, server.IP))
		}
	}

	if onlineCount == totalServers {
		return nil // All came online just before timeout
	}

	return fmt.Errorf("timeout waiting for servers: %d/%d online, offline servers: %s",
		onlineCount, totalServers, strings.Join(offlineServers, ", "))
}
