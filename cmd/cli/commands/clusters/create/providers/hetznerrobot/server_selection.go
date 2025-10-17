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

	// Step 3: Interactive vSwitch selection (list all vswitches or create new)
	vswitchResult, err := SelectVSwitchInteractive(ctx, client, clusterName)
	if err != nil {
		return nil, fmt.Errorf("vswitch selection failed: %w", err)
	}

	// Display selection summary
	DisplayVSwitchSelectionSummary(vswitchResult)

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
