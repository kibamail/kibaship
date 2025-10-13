package hetznerrobot

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/config"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
	"github.com/siderolabs/talos/pkg/machinery/client"
)

// ServerNetworkInfo contains discovered network information from a Talos server
type ServerNetworkInfo struct {
	ServerID         string
	ServerIP         string
	ServerName       string
	NetworkInfo      *DiscoveredNetworkInfo
	PublicInterface  string
	PrivateInterface string
	PublicGW         string
	PrivateGW        string
	PublicCIDR       string
	PrivateCIDR      string
	IsOnline         bool
	Error            error
}

// TalosDiscoveryResult contains results from Talos discovery process
type TalosDiscoveryResult struct {
	ServersInfo map[string]*ServerNetworkInfo
	AllOnline   bool
	Success     bool
}

// WaitForServersOnline waits for all servers to come back online after reboot
func WaitForServersOnline(ctx context.Context, cfg *config.CreateConfig, timeout time.Duration) error {
	if cfg.HetznerRobot == nil || len(cfg.HetznerRobot.SelectedServers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("⏳"),
		styles.TitleStyle.Render("Waiting for Servers to Come Online"))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Checking %d server(s) for connectivity every 15 seconds (timeout: 6 minutes)...", len(cfg.HetznerRobot.SelectedServers))))

	startTime := time.Now()
	deadline := startTime.Add(6 * time.Minute) // Fixed 6 minute timeout

	// Track which servers are online
	serverOnline := make(map[string]bool)
	for _, server := range cfg.HetznerRobot.SelectedServers {
		serverOnline[server.ID] = false
	}

	ticker := time.NewTicker(15 * time.Second) // Check every 15 seconds
	defer ticker.Stop()

	checkCount := 0

	// Do initial check immediately
	performServerChecks(cfg, serverOnline, &checkCount, startTime)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for servers")

		case <-time.After(time.Until(deadline)):
			// Timeout reached - show final status
			displayServerStatusTable(cfg, serverOnline, startTime)

			offlineServers := []string{}
			for serverID, online := range serverOnline {
				if !online {
					offlineServers = append(offlineServers, serverID)
				}
			}
			return fmt.Errorf("timeout waiting for servers to come online after 6 minutes: %v", offlineServers)

		case <-ticker.C:
			allOnline := performServerChecks(cfg, serverOnline, &checkCount, startTime)

			if allOnline {
				displayServerStatusTable(cfg, serverOnline, startTime)
				totalTime := time.Since(startTime).Round(time.Second)
				fmt.Printf("\n%s %s\n",
					styles.TitleStyle.Render("🎉"),
					styles.TitleStyle.Render(fmt.Sprintf("All servers online after %s!", totalTime)))
				return nil
			}
		}
	}
}

// performServerChecks checks all servers and updates their online status
func performServerChecks(cfg *config.CreateConfig, serverOnline map[string]bool, checkCount *int, startTime time.Time) bool {
	*checkCount++
	allOnline := true

	for _, server := range cfg.HetznerRobot.SelectedServers {
		if serverOnline[server.ID] {
			continue // Skip servers already online
		}

		// Try to check if Talos API is responding on port 50000
		online := checkTalosAPIReady(server.IP, 3*time.Second)
		serverOnline[server.ID] = online

		if !online {
			allOnline = false
		}
	}

	// Display status table after each check
	displayServerStatusTable(cfg, serverOnline, startTime)

	return allOnline
}

// displayServerStatusTable shows a table with the current status of all servers
func displayServerStatusTable(cfg *config.CreateConfig, serverOnline map[string]bool, startTime time.Time) {
	elapsed := time.Since(startTime).Round(time.Second)

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📊"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Server Status Check (elapsed: %s)", elapsed)))

	// Define table styles
	headerStyle := lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Bold(true).
		Align(lipgloss.Center)

	cellStyle := lipgloss.NewStyle().
		Foreground(styles.TextColor).
		Align(lipgloss.Left).
		Padding(0, 1)

	statusStyle := lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Center)

	// Create table
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.PrimaryColor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}

			// Special styling for status column
			if col == 3 && row > 0 {
				serverIdx := row - 1
				if serverIdx < len(cfg.HetznerRobot.SelectedServers) {
					serverID := cfg.HetznerRobot.SelectedServers[serverIdx].ID
					if serverOnline[serverID] {
						return statusStyle.Copy().Foreground(lipgloss.Color("#00D4AA")) // Green
					}
					return statusStyle.Copy().Foreground(lipgloss.Color("#F59E0B")) // Orange
				}
			}

			return cellStyle
		}).
		Headers("Server", "ID", "IP Address", "Status")

	// Add server rows
	for _, server := range cfg.HetznerRobot.SelectedServers {
		status := "⏳ Checking..."
		if serverOnline[server.ID] {
			status = "✅ Online"
		}

		t.Row(
			server.Name,
			server.ID,
			server.IP,
			status,
		)
	}

	fmt.Println(t.Render())
}

// checkTalosAPIReady attempts to connect to the Talos API and verify it's responding
func checkTalosAPIReady(ip string, timeout time.Duration) bool {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create TLS config with InsecureSkipVerify for pre-bootstrap provisioning
	// We only use insecure connections before the cluster is bootstrapped
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Create Talos client with insecure TLS
	c, err := client.New(ctx,
		client.WithEndpoints(ip),
		client.WithTLSConfig(tlsConfig),
	)
	if err != nil {
		return false
	}
	defer func() { _ = c.Close() }()

	// Verify we can connect by trying a simple API call
	_, err = c.Version(ctx)
	if err != nil {
		// Check if error is "API is not implemented in maintenance mode"
		// This is actually a SUCCESS - the server is online, just in maintenance mode
		if strings.Contains(err.Error(), "maintenance mode") {
			return true
		}
		// Check for "Unimplemented" which also indicates maintenance mode
		if strings.Contains(err.Error(), "Unimplemented") {
			return true
		}
		return false
	}

	return true
}

// DiscoverServerNetworks connects to each server via Talos and retrieves network information
func DiscoverServerNetworks(ctx context.Context, cfg *config.CreateConfig) (*TalosDiscoveryResult, error) {
	if cfg.HetznerRobot == nil || len(cfg.HetznerRobot.SelectedServers) == 0 {
		return nil, fmt.Errorf("no servers configured")
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🔍"),
		styles.TitleStyle.Render("Discovering Server Network Configuration"))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Connecting to servers via Talos API to retrieve network information..."))

	result := &TalosDiscoveryResult{
		ServersInfo: make(map[string]*ServerNetworkInfo),
		AllOnline:   true,
		Success:     true,
	}

	for _, server := range cfg.HetznerRobot.SelectedServers {
		fmt.Printf("\n%s %s %s (%s)\n",
			styles.CommandStyle.Render("🔗"),
			styles.DescriptionStyle.Render("Connecting to:"),
			styles.CommandStyle.Render(server.Name),
			styles.DescriptionStyle.Render(server.IP))

		// Discover network info (servers should already be online from WaitForServersOnline)
		serverInfo, err := discoverSingleServer(ctx, server)
		if err != nil {
			fmt.Printf("%s %s: %v\n",
				styles.CommandStyle.Render("❌"),
				styles.CommandStyle.Render("Failed to discover network info"),
				err)

			serverInfo = &ServerNetworkInfo{
				ServerID:   server.ID,
				ServerIP:   server.IP,
				ServerName: server.Name,
				IsOnline:   false,
				Error:      err,
			}
			result.AllOnline = false
			result.Success = false
		} else {
			serverInfo.IsOnline = true
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render("Network information retrieved successfully"))
		}

		result.ServersInfo[server.ID] = serverInfo
	}

	// Display summary
	displayNetworkDiscoverySummary(result)

	return result, nil
}

// discoverSingleServer connects to a single server and retrieves its network information
func discoverSingleServer(ctx context.Context, server config.HetznerRobotServer) (*ServerNetworkInfo, error) {
	// Create TLS config with InsecureSkipVerify for pre-bootstrap provisioning
	// We only use insecure connections before the cluster is bootstrapped
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Create Talos client with insecure TLS
	c, err := client.New(ctx,
		client.WithEndpoints(server.IP),
		client.WithTLSConfig(tlsConfig),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Talos client: %w", err)
	}
	defer func() { _ = c.Close() }()

	// Verify we can connect by trying a simple API call
	_, err = c.Version(ctx)
	if err != nil {
		// Check if error is maintenance mode - this is acceptable during provisioning
		if !strings.Contains(err.Error(), "maintenance mode") && !strings.Contains(err.Error(), "Unimplemented") {
			return nil, fmt.Errorf("failed to connect to Talos API: %w", err)
		}
		// Server is in maintenance mode, which is expected - continue with discovery
	}

	// Use the new network discovery service
	discoveryService := NewNetworkDiscoveryService(c)
	networkInfo, err := discoveryService.DiscoverAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover network information: %w", err)
	}

	// DEBUG: Log raw discovery results
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🐛"),
		styles.CommandStyle.Render("DEBUG: Raw Discovery Results"))
	fmt.Printf("  %s %d\n", styles.DescriptionStyle.Render("Addresses:"), len(networkInfo.Addresses))
	for i, addr := range networkInfo.Addresses {
		if i < 10 { // Limit to first 10
			fmt.Printf("    [%d] %s - %s (Family: %s, Scope: %s)\n", i, addr.LinkName, addr.Address, addr.Family, addr.Scope)
		}
	}
	fmt.Printf("  %s %d\n", styles.DescriptionStyle.Render("Links:"), len(networkInfo.Links))
	for i, link := range networkInfo.Links {
		if i < 10 { // Limit to first 10
			fmt.Printf("    [%d] %s - %s (MTU: %d)\n", i, link.Name, link.State, link.MTU)
		}
	}
	fmt.Printf("  %s %d\n", styles.DescriptionStyle.Render("Routes:"), len(networkInfo.Routes))
	for i, route := range networkInfo.Routes {
		if i < 10 { // Limit to first 10
			fmt.Printf("    [%d] Dest: %s, GW: %s, Link: %s\n", i, route.Destination, route.Gateway, route.OutLinkName)
		}
	}

	// === NEW TYPESCRIPT-STYLE DETECTION ===
	// Use FindPublicInterface to detect public interface using default route
	// This matches the TypeScript findPublicInterface() method exactly
	publicInterfaceInfo := FindPublicInterface(networkInfo.Links, networkInfo.Addresses, networkInfo.Routes, server.IP)

	var publicInterface, publicGateway, publicCIDR string
	if publicInterfaceInfo != nil {
		publicInterface = publicInterfaceInfo.Name
		publicGateway = publicInterfaceInfo.Gateway
		publicCIDR = publicInterfaceInfo.AddressSubnet

		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.CommandStyle.Render("Public Interface Detected (via default route):"))
		fmt.Printf("  %s %s\n", styles.DescriptionStyle.Render("Interface:"), publicInterface)
		fmt.Printf("  %s %s\n", styles.DescriptionStyle.Render("Gateway:"), publicGateway)
		fmt.Printf("  %s %s\n", styles.DescriptionStyle.Render("Address:"), publicCIDR)
	} else {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.CommandStyle.Render("Warning: Could not detect public interface via default route"))
	}

	// Detect private interface and private CIDR
	// Private interfaces are detected using the secondary method (IP range-based)
	interfaces := DetectNetworkInterfaces(networkInfo.Addresses)
	privateInterface := interfaces.PrivateInterface

	privateCIDR := ""
	if privateInterface != "" {
		privateCIDR = GetPrimaryPrivateCIDR(networkInfo.Addresses, privateInterface)
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.CommandStyle.Render("Private Interface Detected:"))
		fmt.Printf("  %s %s\n", styles.DescriptionStyle.Render("Interface:"), privateInterface)
		fmt.Printf("  %s %s\n", styles.DescriptionStyle.Render("Address:"), privateCIDR)
	}

	// Detect private gateway from routes
	gateways := DetectGateways(networkInfo.Routes, networkInfo.Addresses)
	privateGateway := gateways.PrivateGateway

	// Build server info
	serverInfo := &ServerNetworkInfo{
		ServerID:         server.ID,
		ServerIP:         server.IP,
		ServerName:       server.Name,
		NetworkInfo:      networkInfo,
		PublicInterface:  publicInterface,
		PrivateInterface: privateInterface,
		PublicGW:         publicGateway,
		PrivateGW:        privateGateway,
		PublicCIDR:       publicCIDR,
		PrivateCIDR:      privateCIDR,
	}

	// Log discovered information
	logServerNetworkInfo(serverInfo)

	return serverInfo, nil
}


// logServerNetworkInfo logs the discovered network information for a server
func logServerNetworkInfo(info *ServerNetworkInfo) {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📊"),
		styles.CommandStyle.Render("Network Information:"))

	if info.NetworkInfo == nil {
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  → Status:"),
			styles.DescriptionStyle.Render("No network information available"))
		return
	}

	// Log detected interfaces
	fmt.Printf("%s %s\n",
		styles.DescriptionStyle.Render("  → Detected Interfaces:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d found", len(info.NetworkInfo.Interfaces.AllInterfaces))))

	if info.PublicInterface != "" {
		fmt.Printf("    %s Public: %s (%s)\n",
			styles.DescriptionStyle.Render("·"),
			styles.CommandStyle.Render(info.PublicInterface),
			styles.DescriptionStyle.Render(info.PublicCIDR))
	}

	if info.PrivateInterface != "" {
		fmt.Printf("    %s Private: %s (%s)\n",
			styles.DescriptionStyle.Render("·"),
			styles.CommandStyle.Render(info.PrivateInterface),
			styles.DescriptionStyle.Render(info.PrivateCIDR))
	}

	// Log addresses summary
	if len(info.NetworkInfo.Addresses) > 0 {
		ipv4Count := 0
		for _, addr := range info.NetworkInfo.Addresses {
			if addr.Family == "inet4" {
				ipv4Count++
			}
		}
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  → Total IPv4 Addresses:"),
			styles.DescriptionStyle.Render(fmt.Sprintf("%d", ipv4Count)))
	}

	// Log links summary
	if len(info.NetworkInfo.Links) > 0 {
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  → Network Links:"),
			styles.DescriptionStyle.Render(fmt.Sprintf("%d found", len(info.NetworkInfo.Links))))
		for i, link := range info.NetworkInfo.Links {
			if i < 3 { // Limit to first 3 links
				fmt.Printf("    %s %s - %s (MTU: %d)\n",
					styles.DescriptionStyle.Render("·"),
					styles.CommandStyle.Render(link.Name),
					styles.DescriptionStyle.Render(link.State),
					link.MTU)
			}
		}
		if len(info.NetworkInfo.Links) > 3 {
			fmt.Printf("    %s\n",
				styles.DescriptionStyle.Render(fmt.Sprintf("... and %d more", len(info.NetworkInfo.Links)-3)))
		}
	}

	// Log gateways
	if info.PublicGW != "" {
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  → Public Gateway:"),
			styles.CommandStyle.Render(info.PublicGW))
	}
	if info.PrivateGW != "" {
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  → Private Gateway:"),
			styles.CommandStyle.Render(info.PrivateGW))
	}
}

// displayNetworkDiscoverySummary displays a summary of the network discovery process
func displayNetworkDiscoverySummary(result *TalosDiscoveryResult) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("📋"),
		styles.TitleStyle.Render("Network Discovery Summary"))

	onlineCount := 0
	for _, info := range result.ServersInfo {
		if info.IsOnline {
			onlineCount++
		}
	}

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("Total Servers:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d", len(result.ServersInfo))))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("Successfully Discovered:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d", onlineCount)))

	if !result.AllOnline {
		failedCount := len(result.ServersInfo) - onlineCount
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("Failed:"),
			styles.CommandStyle.Render(fmt.Sprintf("%d", failedCount)))
	}

	fmt.Println()
}
