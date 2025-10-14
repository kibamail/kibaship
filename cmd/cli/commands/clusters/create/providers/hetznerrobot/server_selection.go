package hetznerrobot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// ServerSelectionResult contains the user's server selection choices
type ServerSelectionResult struct {
	SelectedServers []Server
	ClusterType     string
	ConfirmSetup    bool
}

// CompleteSelectionResult contains both server and vswitch selections
type CompleteSelectionResult struct {
	ServerSelection  *ServerSelectionResult
	VSwitchSelection *VSwitchSelectionResult
	RescueResult     *RescueManagementResult
	NetworkRanges    *NetworkRanges
}

// ClusterTypeOption represents different cluster setup options
type ClusterTypeOption struct {
	Value       string
	Label       string
	Description string
}

// GetClusterTypeOptions returns available cluster setup options
func GetClusterTypeOptions() []ClusterTypeOption {
	return []ClusterTypeOption{
		{
			Value:       "single-node",
			Label:       "Single Node Cluster",
			Description: "All-in-one cluster (control plane + worker on one server)",
		},
		{
			Value:       "multi-node",
			Label:       "Multi-Node Cluster",
			Description: "Separate control plane and worker nodes (requires 2+ servers)",
		},
		{
			Value:       "ha-cluster",
			Label:       "High Availability Cluster",
			Description: "HA control plane with multiple workers (requires 3+ servers)",
		},
	}
}

// SelectServersInteractive presents an interactive form for server selection
func SelectServersInteractive(ctx context.Context, client *Client) (*ServerSelectionResult, error) {
	// First, fetch and display servers
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🤖"),
		styles.TitleStyle.Render("Hetzner Robot Server Selection"))

	servers, err := client.ListServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	// Filter out cancelled servers
	availableServers := make([]Server, 0)
	for _, server := range servers {
		if !server.Cancelled && strings.ToLower(server.Status) == "ready" {
			availableServers = append(availableServers, server)
		}
	}

	if len(availableServers) == 0 {
		return nil, fmt.Errorf("no ready servers available for cluster setup")
	}

	fmt.Printf("\n%s %s\n",
		styles.HelpStyle.Render("📋"),
		styles.HelpStyle.Render(fmt.Sprintf("Found %d ready servers available for cluster setup:", len(availableServers))))

	// Display servers in a compact format
	displayCompactServerList(availableServers)

	// Create server selection options
	serverOptions := make([]huh.Option[string], len(availableServers))
	for i, server := range availableServers {
		label := fmt.Sprintf("%s (%s) - %s [%s]",
			server.Name,
			server.ID,
			server.IP,
			server.Product)

		serverOptions[i] = huh.NewOption(label, server.ID)
	}

	// Create cluster type options
	clusterTypes := GetClusterTypeOptions()
	clusterTypeOptions := make([]huh.Option[string], len(clusterTypes))
	for i, ct := range clusterTypes {
		clusterTypeOptions[i] = huh.NewOption(ct.Label+" - "+ct.Description, ct.Value)
	}

	// Form variables
	var selectedServerIDs []string
	var clusterType string

	// Create the interactive form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("🚀 Kubernetes Cluster Setup").
				Description("Select servers and cluster configuration for your Kubernetes cluster.\n"+
					"Make sure selected servers meet the requirements for your chosen cluster type."),

			huh.NewSelect[string]().
				Title("Choose cluster type").
				Description("Select the type of Kubernetes cluster you want to create").
				Options(clusterTypeOptions...).
				Value(&clusterType),
		),

		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select servers for your cluster").
				Description("Choose one or more servers based on your cluster type selection").
				Options(serverOptions...).
				Value(&selectedServerIDs).
				Validate(func(selectedIDs []string) error {
					return validateServerSelection(selectedIDs, clusterType)
				}),
		),
	).WithTheme(createFormTheme())

	// Run the form
	err = form.Run()
	if err != nil {
		return nil, fmt.Errorf("form interaction failed: %w", err)
	}

	// Convert selected server IDs to Server objects
	selectedServers := make([]Server, 0, len(selectedServerIDs))
	for _, serverID := range selectedServerIDs {
		for _, server := range availableServers {
			if server.ID == serverID {
				selectedServers = append(selectedServers, server)
				break
			}
		}
	}

	return &ServerSelectionResult{
		SelectedServers: selectedServers,
		ClusterType:     clusterType,
		ConfirmSetup:    true, // Always true since we removed the confirmation step
	}, nil
}

// validateServerSelection validates the server selection based on cluster type
func validateServerSelection(selectedIDs []string, clusterType string) error {
	switch clusterType {
	case "single-node":
		if len(selectedIDs) != 1 {
			return fmt.Errorf("single-node cluster requires exactly 1 server")
		}
	case "multi-node":
		if len(selectedIDs) < 2 {
			return fmt.Errorf("multi-node cluster requires at least 2 servers")
		}
	case "ha-cluster":
		if len(selectedIDs) < 3 {
			return fmt.Errorf("high availability cluster requires at least 3 servers")
		}
	default:
		return fmt.Errorf("unknown cluster type: %s", clusterType)
	}
	return nil
}

// displayCompactServerList shows servers in a compact, readable format
func displayCompactServerList(servers []Server) {
	for i, server := range servers {
		statusColor := getStatusColor(server.Status)

		serverInfo := lipgloss.NewStyle().
			Foreground(styles.TextColor).
			Render(fmt.Sprintf("  %d. %s (%s) - %s | %s | %s",
				i+1,
				server.Name,
				server.ID,
				server.IP,
				server.Product,
				server.DC))

		statusBadge := lipgloss.NewStyle().
			Foreground(statusColor).
			Bold(true).
			Render(fmt.Sprintf("[%s]", server.Status))

		fmt.Printf("%s %s\n", serverInfo, statusBadge)
	}
	fmt.Println()
}

// createFormTheme creates a custom theme for the huh form
func createFormTheme() *huh.Theme {
	theme := huh.ThemeCharm()

	// Customize colors to match our existing style
	theme.Focused.Title = theme.Focused.Title.Foreground(styles.PrimaryColor)
	theme.Focused.NoteTitle = theme.Focused.NoteTitle.Foreground(styles.PrimaryColor)
	theme.Focused.Directory = theme.Focused.Directory.Foreground(styles.AccentColor)
	theme.Focused.Description = theme.Focused.Description.Foreground(styles.MutedColor)
	theme.Focused.ErrorIndicator = theme.Focused.ErrorIndicator.Foreground(lipgloss.Color("#EF4444"))
	theme.Focused.ErrorMessage = theme.Focused.ErrorMessage.Foreground(lipgloss.Color("#EF4444"))

	return theme
}

// DisplaySelectionSummary shows a summary of the user's selections
func DisplaySelectionSummary(result *ServerSelectionResult) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Server Selection Summary"))

	clusterTypes := GetClusterTypeOptions()
	var clusterTypeLabel string
	for _, ct := range clusterTypes {
		if ct.Value == result.ClusterType {
			clusterTypeLabel = ct.Label
			break
		}
	}

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("Cluster Type:"),
		styles.DescriptionStyle.Render(clusterTypeLabel))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("Selected Servers:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d servers", len(result.SelectedServers))))

	for i, server := range result.SelectedServers {
		role := getServerRole(i, result.ClusterType, len(result.SelectedServers))
		fmt.Printf("   %d. %s (%s) - %s [%s]\n",
			i+1,
			server.Name,
			server.ID,
			server.IP,
			role)
	}
	fmt.Println()
}

// getServerRole determines the role of a server based on its position and cluster type
func getServerRole(index int, clusterType string, totalServers int) string {
	switch clusterType {
	case "single-node":
		return "Control Plane + Worker"
	case "multi-node":
		if index == 0 {
			return "Control Plane"
		}
		return "Worker"
	case "ha-cluster":
		if index < 3 {
			return "Control Plane"
		}
		return "Worker"
	default:
		return "Unknown"
	}
}

// SelectServersAndVSwitchInteractive orchestrates the complete server and vswitch selection process
func SelectServersAndVSwitchInteractive(ctx context.Context, client *Client, clusterName string) (*CompleteSelectionResult, error) {
	// Step 1: Server selection
	serverResult, err := SelectServersInteractive(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("server selection failed: %w", err)
	}

	// Display server selection summary
	DisplaySelectionSummary(serverResult)

	// Step 2: Show critical warning and get confirmation BEFORE any destructive operations
	if !showCriticalWarningAndConfirm(clusterName) {
		return nil, fmt.Errorf("cluster creation cancelled by user")
	}

	// Step 3: Check for existing vSwitch or create new one
	vswitchResult, err := createOrReuseVSwitchResult(ctx, client, clusterName)
	if err != nil {
		return nil, fmt.Errorf("vswitch configuration failed: %w", err)
	}

	// Step 4: Final confirmation with complete summary
	proceed, err := DisplayFinalConfirmation(serverResult, vswitchResult)
	if err != nil {
		return nil, fmt.Errorf("final confirmation failed: %w", err)
	}

	if !proceed {
		return nil, fmt.Errorf("cluster setup cancelled by user")
	}

	// Step 5: Process vswitch attachment (create if needed, attach servers, monitor status)
	attachmentResult, err := ProcessVSwitchAttachment(ctx, client, vswitchResult, serverResult.SelectedServers)
	if err != nil {
		return nil, fmt.Errorf("vswitch attachment failed: %w", err)
	}

	if !attachmentResult.Success {
		return nil, fmt.Errorf("vswitch attachment was not successful")
	}

	// Step 6: Generate network ranges for the cluster
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🌐"),
		styles.TitleStyle.Render("Generating Network Configuration"))

	networkRanges, err := GenerateClusterNetworkRanges()
	if err != nil {
		return nil, fmt.Errorf("network range generation failed: %w", err)
	}

	// Display generated network ranges
	displayNetworkRanges(networkRanges)

	// Step 7: Process server rescue mode (check status, enable if needed, reset, monitor readiness)
	rescueResult, err := ProcessServerRescueMode(ctx, client, serverResult.SelectedServers)
	if err != nil {
		return nil, fmt.Errorf("server rescue mode management failed: %w", err)
	}

	if !rescueResult.Success {
		return nil, fmt.Errorf("server rescue mode was not successful")
	}

	return &CompleteSelectionResult{
		ServerSelection:  serverResult,
		VSwitchSelection: vswitchResult,
		RescueResult:     rescueResult,
		NetworkRanges:    networkRanges,
	}, nil
}

// createOrReuseVSwitchResult checks for existing vSwitch with cluster name and asks user to reuse or create new
func createOrReuseVSwitchResult(ctx context.Context, client *Client, clusterName string) (*VSwitchSelectionResult, error) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🔗"),
		styles.TitleStyle.Render("VSwitch Configuration"))

	// Fetch existing vSwitches
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.DescriptionStyle.Render("Checking for existing vSwitches..."))

	vswitches, err := client.ListVSwitches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list vSwitches: %w", err)
	}

	// Check if there's a vSwitch with matching name
	var matchingVSwitch *VSwitch
	for _, vs := range vswitches {
		if vs.Name == clusterName && !vs.Cancelled {
			matchingVSwitch = &vs
			break
		}
	}

	// If matching vSwitch found, ask user if they want to reuse it
	if matchingVSwitch != nil {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.CommandStyle.Render("Found existing vSwitch with matching name!"))

		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  → Name:"),
			styles.CommandStyle.Render(matchingVSwitch.Name))

		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  → ID:"),
			styles.DescriptionStyle.Render(matchingVSwitch.ID))

		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  → VLAN:"),
			styles.DescriptionStyle.Render(fmt.Sprintf("%d", matchingVSwitch.VLAN)))

		// Ask user for confirmation
		var reuseVSwitch bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Do you want to reuse this vSwitch?").
					Description("Select 'Yes' to reuse the existing vSwitch, or 'No' to create a new one").
					Value(&reuseVSwitch),
			),
		).WithTheme(createFormTheme())

		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("failed to get user confirmation: %w", err)
		}

		if reuseVSwitch {
			fmt.Printf("\n%s %s\n",
				styles.TitleStyle.Render("♻️"),
				styles.TitleStyle.Render("Checking vSwitch for cloud network attachments..."))

			// Get full vSwitch details to check for cloud network attachments
			vswitchDetails, err := client.GetVSwitchDetails(ctx, matchingVSwitch.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get vSwitch details: %w", err)
			}

			// Check if vSwitch has cloud network attachments
			if vswitchDetails.VSwitch.HasCloudNetworkAttached {
				fmt.Printf("\n%s %s\n",
					styles.CommandStyle.Render("⚠️"),
					styles.CommandStyle.Render("VSwitch is attached to a Hetzner Cloud Network"))

				fmt.Printf("%s %s\n",
					styles.DescriptionStyle.Render("  →"),
					styles.DescriptionStyle.Render("This vSwitch cannot be reused while attached to a cloud network"))

				// Ask user if they want to delete and recreate
				var deleteAndRecreate bool
				deleteForm := huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title("Delete and recreate this vSwitch?").
							Description("This will delete the existing vSwitch, wait 30 seconds, recreate it with the same name, and proceed.").
							Value(&deleteAndRecreate),
					),
				).WithTheme(createFormTheme())

				if err := deleteForm.Run(); err != nil {
					return nil, fmt.Errorf("failed to get user confirmation: %w", err)
				}

				if !deleteAndRecreate {
					return nil, fmt.Errorf("vSwitch '%s' (ID: %s) is currently attached to a Hetzner Cloud Network. "+
						"You must manually detach the vSwitch from the cloud network before reusing it. "+
						"Go to Hetzner Cloud Console → Networks → Detach vSwitch, then try again",
						vswitchDetails.VSwitch.Name, vswitchDetails.VSwitch.ID)
				}

				// Delete the vSwitch
				fmt.Printf("\n%s %s\n",
					styles.CommandStyle.Render("🗑️"),
					styles.DescriptionStyle.Render(fmt.Sprintf("Deleting vSwitch '%s' (ID: %s)...", matchingVSwitch.Name, matchingVSwitch.ID)))

				if err := client.DeleteVSwitch(ctx, matchingVSwitch.ID); err != nil {
					return nil, fmt.Errorf("failed to delete vSwitch: %w", err)
				}

				fmt.Printf("%s %s\n",
					styles.TitleStyle.Render("✅"),
					styles.TitleStyle.Render("VSwitch deleted successfully"))

				// Wait 30 seconds after deletion
				fmt.Printf("\n%s %s\n",
					styles.CommandStyle.Render("⏳"),
					styles.DescriptionStyle.Render("Waiting 30 seconds after deletion..."))

				if err := waitWithCountdown(ctx, 30); err != nil {
					return nil, err
				}

				// Now we'll create a new vSwitch with the same name
				// Fall through to the creation logic below by unsetting matchingVSwitch
				matchingVSwitch = nil
			} else {
				fmt.Printf("\n%s %s\n",
					styles.TitleStyle.Render("✅"),
					styles.TitleStyle.Render("VSwitch is ready for reuse"))

				return &VSwitchSelectionResult{
					SelectedVSwitch: matchingVSwitch,
					CreateNew:       false,
				}, nil
			}
		}

		// User declined to reuse - exit with error (only if matchingVSwitch still exists)
		if matchingVSwitch != nil {
			return nil, fmt.Errorf("a vSwitch named '%s' already exists (ID: %s, VLAN: %d). "+
				"Please delete or rename the existing vSwitch before creating a new cluster with this name",
				matchingVSwitch.Name, matchingVSwitch.ID, matchingVSwitch.VLAN)
		}
	}

	// Create new vSwitch - find available VLAN ID
	usedVLANs := make(map[int]bool)
	for _, vs := range vswitches {
		if !vs.Cancelled {
			usedVLANs[vs.VLAN] = true
		}
	}

	// Find an available VLAN in the range 4000-4091
	var availableVLAN int
	for vlan := 4000; vlan <= 4091; vlan++ {
		if !usedVLANs[vlan] {
			availableVLAN = vlan
			break
		}
	}

	if availableVLAN == 0 {
		return nil, fmt.Errorf("no available VLAN IDs in range 4000-4091")
	}

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("A new vswitch will be created for this cluster"))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("Name:"),
		styles.DescriptionStyle.Render(clusterName))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("VLAN ID:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d", availableVLAN)))

	return &VSwitchSelectionResult{
		CreateNew:      true,
		NewVSwitchName: clusterName,
		NewVSwitchVLAN: availableVLAN,
	}, nil
}

// displayNetworkRanges shows the generated network configuration
func displayNetworkRanges(ranges *NetworkRanges) {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📊"),
		styles.CommandStyle.Render("Generated Network Configuration"))

	networkInfo := ranges.GetNetworkInfo()

	// Display cluster network
	clusterNet := networkInfo["cluster_network"].(map[string]interface{})
	fmt.Printf("%s %s: %s (%s)\n",
		styles.DescriptionStyle.Render("  →"),
		styles.CommandStyle.Render("Cluster Network"),
		styles.DescriptionStyle.Render(clusterNet["cidr"].(string)),
		styles.DescriptionStyle.Render(clusterNet["size"].(string)))

	// Display vSwitch subnet
	vswitchNet := networkInfo["vswitch_subnet"].(map[string]interface{})
	fmt.Printf("%s %s: %s (%s)\n",
		styles.DescriptionStyle.Render("  →"),
		styles.CommandStyle.Render("VSwitch Subnet"),
		styles.DescriptionStyle.Render(vswitchNet["cidr"].(string)),
		styles.DescriptionStyle.Render(vswitchNet["size"].(string)))

	// Display load balancer subnet
	lbNet := networkInfo["loadbalancer_subnet"].(map[string]interface{})
	fmt.Printf("%s %s: %s (%s)\n",
		styles.DescriptionStyle.Render("  →"),
		styles.CommandStyle.Render("Load Balancer Subnet"),
		styles.DescriptionStyle.Render(lbNet["cidr"].(string)),
		styles.DescriptionStyle.Render(lbNet["size"].(string)))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("Network ranges generated successfully"))
}

// waitWithCountdown waits for the specified number of seconds with a countdown display
func waitWithCountdown(ctx context.Context, seconds int) error {
	waitDuration := time.Duration(seconds) * time.Second
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(waitDuration)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}

		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  →"),
			styles.DescriptionStyle.Render(fmt.Sprintf("%.0f seconds remaining...", remaining.Seconds())))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(remaining):
			break
		case <-ticker.C:
			continue
		}

		if remaining <= 0 {
			break
		}
	}

	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Wait complete"))

	return nil
}
