package hetznerrobot

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/go-ping/ping"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// RescueManagementResult contains the result of the rescue management process
type RescueManagementResult struct {
	Success          bool
	ReadyServers     []ServerRescueStatus
	FailedServers    []ServerRescueStatus
	TimeoutReached   bool
	RescuePasswords  map[string]string // serverID -> password
}

const (
	RescueTimeout       = 6 * time.Minute
	RescueCheckInterval = 15 * time.Second
	ConnectivityTimeout = 6 * time.Minute
	ConnectivityCheckInterval = 10 * time.Second

	// Rescue status constants
	RescueStatusChecking     = "checking"
	RescueStatusDeactivating = "deactivating"
	RescueStatusRebooting    = "rebooting"
	RescueStatusConnecting   = "connecting"
	RescueStatusActivating   = "activating"
	RescueStatusEnabling     = "enabling"
	RescueStatusResetting    = "resetting"
	RescueStatusPinging      = "pinging"
	RescueStatusReady        = "ready"
	RescueStatusFailed       = "failed"
)

// ProcessServerRescueMode handles the enhanced server rescue mode process
func ProcessServerRescueMode(ctx context.Context, client *Client, selectedServers []Server) (*RescueManagementResult, error) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🛠️"),
		styles.TitleStyle.Render("Enhanced Server Rescue Mode Management"))

	// Step 0: Check if all servers support rescue mode
	if err := validateRescueAvailability(ctx, client, selectedServers); err != nil {
		return nil, err
	}

	// Step 1: Check current rescue status for all servers
	rescueStatuses, err := checkAllRescueStatuses(ctx, client, selectedServers)
	if err != nil {
		return nil, fmt.Errorf("failed to check rescue statuses: %w", err)
	}

	// Step 2: Display initial rescue status table
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("📊"),
		styles.TitleStyle.Render("Initial Server Rescue Mode Status"))

	displayRescueStatusTable(rescueStatuses)

	// Step 3: Deactivate rescue mode on servers that already have it active
	deactivatedServers, err := deactivateExistingRescueMode(ctx, client, rescueStatuses)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate existing rescue mode: %w", err)
	}

	// Step 4: Reboot ONLY servers that were deactivated and wait for them to come back online
	if len(deactivatedServers) > 0 {
		if err := rebootAndWaitForServers(ctx, client, deactivatedServers, rescueStatuses); err != nil {
			return nil, fmt.Errorf("failed to reboot servers: %w", err)
		}
	}

	// Step 5: Activate rescue mode on ALL servers
	if err := activateRescueModeOnAllServers(ctx, client, rescueStatuses); err != nil {
		return nil, fmt.Errorf("failed to activate rescue mode on all servers: %w", err)
	}

	// Step 6: Reset all servers again
	if err := resetAllServers(ctx, client, rescueStatuses); err != nil {
		return nil, fmt.Errorf("failed to reset servers: %w", err)
	}

	// Step 7: Monitor server readiness in rescue mode
	result, err := monitorServerReadiness(ctx, client, rescueStatuses)
	if err != nil {
		return nil, fmt.Errorf("failed to monitor server readiness: %w", err)
	}

	return result, nil
}

// validateRescueAvailability checks if all servers support rescue mode
func validateRescueAvailability(ctx context.Context, client *Client, selectedServers []Server) error {
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.DescriptionStyle.Render("Checking rescue mode availability for all servers..."))

	unsupportedServers, err := client.CheckAllServersRescueAvailability(ctx, selectedServers)
	if err != nil {
		return fmt.Errorf("failed to check rescue availability: %w", err)
	}

	if len(unsupportedServers) > 0 {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render("Error: Some servers do not support rescue mode"))

		for _, server := range unsupportedServers {
			fmt.Printf("%s %s (%s) - %s\n",
				styles.DescriptionStyle.Render("  →"),
				server.Name,
				server.ID,
				styles.DescriptionStyle.Render("Rescue mode not available"))
		}

		return fmt.Errorf("rescue mode is not available for %d server(s): %s",
			len(unsupportedServers), getServerNames(unsupportedServers))
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("All servers support rescue mode"))

	return nil
}

// checkAllRescueStatuses fetches rescue status for all selected servers
func checkAllRescueStatuses(ctx context.Context, client *Client, selectedServers []Server) ([]ServerRescueStatus, error) {
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.DescriptionStyle.Render("Checking rescue mode status for all servers..."))

	statuses := make([]ServerRescueStatus, 0, len(selectedServers))

	for _, server := range selectedServers {
		status, err := client.GetServerRescueStatus(ctx, server.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get rescue status for server %s: %w", server.Name, err)
		}
		
		// Set server name from our selected servers list
		status.ServerName = server.Name
		if status.ServerIP == "" {
			status.ServerIP = server.IP
		}
		
		statuses = append(statuses, *status)
	}

	return statuses, nil
}

// deactivateExistingRescueMode deactivates rescue mode on servers that already have it active
// Returns the list of servers that were deactivated
func deactivateExistingRescueMode(ctx context.Context, client *Client, statuses []ServerRescueStatus) ([]ServerRescueStatus, error) {
	// Find servers that already have rescue mode active
	serversToDeactivate := getServersToDeactivate(statuses)

	if len(serversToDeactivate) == 0 {
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("ℹ️"),
			styles.DescriptionStyle.Render("No servers have active rescue mode - skipping deactivation"))
		return []ServerRescueStatus{}, nil
	}

	// Display which servers need deactivation
	serverNames := make([]string, len(serversToDeactivate))
	for i, server := range serversToDeactivate {
		serverNames[i] = server.ServerName
	}

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔄"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Found %d server(s) with active rescue mode: %s",
			len(serversToDeactivate), strings.Join(serverNames, ", "))))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⚠️"),
		styles.DescriptionStyle.Render("Deactivating rescue mode to ensure clean state..."))

	// Deactivate rescue mode for each server
	for _, server := range serversToDeactivate {
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  →"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Deactivating rescue mode for %s (%s)...",
				server.ServerName, server.ServerID)))

		if err := client.DisableServerRescueMode(ctx, server.ServerID); err != nil {
			return nil, fmt.Errorf("failed to deactivate rescue mode for server %s: %w", server.ServerName, err)
		}

		// Update status
		for j := range statuses {
			if statuses[j].ServerID == server.ServerID {
				statuses[j].Status = RescueStatusDeactivating
				break
			}
		}
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("Rescue mode deactivation initiated for all required servers"))

	// Monitor deactivation completion
	if err := monitorRescueDeactivation(ctx, client, statuses, serversToDeactivate); err != nil {
		return nil, err
	}

	// Return the servers that were deactivated
	return serversToDeactivate, nil
}

// getServersToDeactivate returns servers that have active rescue mode
func getServersToDeactivate(statuses []ServerRescueStatus) []ServerRescueStatus {
	serversToDeactivate := make([]ServerRescueStatus, 0)
	for _, status := range statuses {
		if status.InRescueMode {
			serversToDeactivate = append(serversToDeactivate, status)
		}
	}
	return serversToDeactivate
}

// monitorRescueDeactivation monitors the deactivation process until completion
func monitorRescueDeactivation(ctx context.Context, client *Client, allStatuses []ServerRescueStatus, serversToDeactivate []ServerRescueStatus) error {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("⏳"),
		styles.DescriptionStyle.Render("Monitoring rescue mode deactivation (up to 6 minutes)..."))

	startTime := time.Now()
	ticker := time.NewTicker(RescueCheckInterval)
	defer ticker.Stop()

	for {
		// Check if timeout reached
		if time.Since(startTime) >= RescueTimeout {
			return fmt.Errorf("timeout: rescue mode deactivation did not complete within 6 minutes")
		}

		// Check deactivation status for all servers that were being deactivated
		allDeactivated := true
		for _, serverToCheck := range serversToDeactivate {
			status, err := client.GetServerRescueStatus(ctx, serverToCheck.ServerID)
			if err != nil {
				return fmt.Errorf("failed to check rescue status for server %s: %w", serverToCheck.ServerName, err)
			}

			// Update status in main list
			for j := range allStatuses {
				if allStatuses[j].ServerID == serverToCheck.ServerID {
					allStatuses[j].InRescueMode = status.InRescueMode
					if !status.InRescueMode {
						allStatuses[j].Status = RescueStatusChecking
					}
					break
				}
			}

			if status.InRescueMode {
				allDeactivated = false
			}
		}

		// Display current status
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("🔄"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Checking deactivation status... (%.0fs elapsed)",
				time.Since(startTime).Seconds())))

		displayRescueStatusTable(allStatuses)

		if allDeactivated {
			fmt.Printf("\n%s %s\n",
				styles.CommandStyle.Render("✅"),
				styles.DescriptionStyle.Render("Rescue mode successfully deactivated on all servers"))
			return nil
		}

		// Wait for next check
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}

// getServersToEnable returns servers that need rescue mode enabled
func getServersToEnable(statuses []ServerRescueStatus) []ServerRescueStatus {
	serversToEnable := make([]ServerRescueStatus, 0)
	for _, status := range statuses {
		if !status.InRescueMode {
			serversToEnable = append(serversToEnable, status)
		}
	}
	return serversToEnable
}

// rebootAndWaitForServers reboots specific servers and waits for them to come back online
func rebootAndWaitForServers(ctx context.Context, client *Client, serversToReboot []ServerRescueStatus, allStatuses []ServerRescueStatus) error {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔄"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Rebooting %d server(s) that had rescue mode deactivated...", len(serversToReboot))))

	// Update statuses and perform reboot only for servers in serversToReboot
	for _, serverToReboot := range serversToReboot {
		// Find and update the status in allStatuses
		for i := range allStatuses {
			if allStatuses[i].ServerID == serverToReboot.ServerID {
				allStatuses[i].Status = RescueStatusRebooting

				fmt.Printf("%s %s\n",
					styles.DescriptionStyle.Render("  →"),
					styles.DescriptionStyle.Render(fmt.Sprintf("Rebooting %s (%s)...",
						allStatuses[i].ServerName, allStatuses[i].ServerID)))

				if err := client.ResetServer(ctx, allStatuses[i].ServerID); err != nil {
					return fmt.Errorf("failed to reboot server %s: %w", allStatuses[i].ServerName, err)
				}
				break
			}
		}
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Reboot initiated for %d server(s)", len(serversToReboot))))

	// Monitor connectivity only for the rebooted servers
	return monitorServerConnectivity(ctx, serversToReboot, allStatuses)
}

// monitorServerConnectivity monitors specific servers until they come back online
func monitorServerConnectivity(ctx context.Context, serversToMonitor []ServerRescueStatus, allStatuses []ServerRescueStatus) error {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("⏳"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Monitoring connectivity for %d server(s) (up to 6 minutes)...", len(serversToMonitor))))

	startTime := time.Now()
	ticker := time.NewTicker(ConnectivityCheckInterval)
	defer ticker.Stop()

	// Update statuses to connecting for servers being monitored
	for _, serverToMonitor := range serversToMonitor {
		for i := range allStatuses {
			if allStatuses[i].ServerID == serverToMonitor.ServerID {
				allStatuses[i].Status = RescueStatusConnecting
				break
			}
		}
	}

	for {
		// Check if timeout reached
		if time.Since(startTime) >= ConnectivityTimeout {
			return fmt.Errorf("timeout: servers did not come back online within 6 minutes")
		}

		// Check connectivity for servers being monitored
		allOnline := true
		for _, serverToMonitor := range serversToMonitor {
			for i := range allStatuses {
				if allStatuses[i].ServerID == serverToMonitor.ServerID {
					if allStatuses[i].Status == RescueStatusConnecting {
						if pingServer(allStatuses[i].ServerIP) {
							allStatuses[i].Status = RescueStatusChecking
						} else {
							allOnline = false
						}
					}
					break
				}
			}
		}

		// Display current status (show all servers)
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("🔄"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Checking connectivity... (%.0fs elapsed)",
				time.Since(startTime).Seconds())))

		displayRescueStatusTable(allStatuses)

		if allOnline {
			fmt.Printf("\n%s %s\n",
				styles.CommandStyle.Render("✅"),
				styles.DescriptionStyle.Render(fmt.Sprintf("%d server(s) back online", len(serversToMonitor))))
			return nil
		}

		// Wait for next check
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}

// activateRescueModeOnAllServers activates rescue mode on ALL selected servers
func activateRescueModeOnAllServers(ctx context.Context, client *Client, statuses []ServerRescueStatus) error {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🛠️"),
		styles.DescriptionStyle.Render("Activating rescue mode on ALL servers..."))

	// Clear any existing rescue passwords since we're starting fresh
	for i := range statuses {
		statuses[i].RescuePassword = ""
		statuses[i].Status = RescueStatusActivating
	}

	// Activate rescue mode for each server
	for i := range statuses {
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  →"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Activating rescue mode for %s (%s)...",
				statuses[i].ServerName, statuses[i].ServerID)))

		rescueStatus, err := client.EnableServerRescueMode(ctx, statuses[i].ServerID)
		if err != nil {
			return fmt.Errorf("failed to activate rescue mode for server %s: %w", statuses[i].ServerName, err)
		}

		// Update the status with new rescue information
		statuses[i].InRescueMode = rescueStatus.InRescueMode
		statuses[i].RescuePassword = rescueStatus.RescuePassword
		statuses[i].Status = RescueStatusEnabling

		if rescueStatus.RescuePassword != "" {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🔑"),
				styles.DescriptionStyle.Render(fmt.Sprintf("Rescue password captured for %s",
					statuses[i].ServerName)))
		}
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("Rescue mode activated on all servers"))

	return nil
}

// enableRescueModeForServers enables rescue mode for servers that need it
func enableRescueModeForServers(ctx context.Context, client *Client, serversToEnable []ServerRescueStatus, allStatuses []ServerRescueStatus) error {
	if len(serversToEnable) == 0 {
		return nil
	}

	// Display which servers need rescue mode enabled
	serverNames := make([]string, len(serversToEnable))
	for i, server := range serversToEnable {
		serverNames[i] = server.ServerName
	}

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🛠️"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%s not in rescue mode. Enabling rescue mode...",
			strings.Join(serverNames, ", "))))

	// Enable rescue mode for each server
	for _, server := range serversToEnable {
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  →"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Enabling rescue mode for %s (%s)...", server.ServerName, server.ServerID)))

		rescueStatus, err := client.EnableServerRescueMode(ctx, server.ServerID)
		if err != nil {
			return fmt.Errorf("failed to enable rescue mode for server %s: %w", server.ServerName, err)
		}

		// Update the status in our main list
		for j := range allStatuses {
			if allStatuses[j].ServerID == server.ServerID {
				allStatuses[j].InRescueMode = rescueStatus.InRescueMode
				allStatuses[j].RescuePassword = rescueStatus.RescuePassword
				allStatuses[j].Status = RescueStatusEnabling
				break
			}
		}

		if rescueStatus.RescuePassword != "" {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🔑"),
				styles.DescriptionStyle.Render(fmt.Sprintf("Rescue password captured for %s",
					server.ServerName)))
		}
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("Rescue mode enabled for all required servers"))

	return nil
}

// resetAllServers performs hardware reset on all servers
func resetAllServers(ctx context.Context, client *Client, statuses []ServerRescueStatus) error {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔄"),
		styles.DescriptionStyle.Render("Performing hardware reset on all servers..."))

	// Reset all servers
	for i := range statuses {
		statuses[i].Status = RescueStatusResetting
		
		fmt.Printf("%s %s\n",
			styles.DescriptionStyle.Render("  →"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Resetting %s (%s)...", statuses[i].ServerName, statuses[i].ServerID)))

		if err := client.ResetServer(ctx, statuses[i].ServerID); err != nil {
			return fmt.Errorf("failed to reset server %s: %w", statuses[i].ServerName, err)
		}
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("Hardware reset initiated for all servers"))

	return nil
}

// displayRescueStatusTable shows the current rescue status in a table
func displayRescueStatusTable(statuses []ServerRescueStatus) {
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
			
			// Special styling for rescue status column
			if col == 3 && row > 0 && row-1 < len(statuses) {
				return statusStyle.Copy().Foreground(getRescueStatusColor(statuses[row-1].InRescueMode, statuses[row-1].Status))
			}
			
			return cellStyle
		}).
		Headers("Server", "ID", "IP Address", "Rescue Status")

	// Add server rows
	for _, status := range statuses {
		t.Row(
			status.ServerName,
			status.ServerID,
			status.ServerIP,
			formatRescueStatus(status.InRescueMode, status.Status),
		)
	}

	fmt.Println(t.Render())
}

// getRescueStatusColor returns appropriate color for rescue status
func getRescueStatusColor(inRescueMode bool, status string) lipgloss.Color {
	switch status {
	case RescueStatusReady:
		return lipgloss.Color("#00D4AA") // Green for ready
	case RescueStatusPinging:
		return lipgloss.Color("#F59E0B") // Orange for pinging
	case RescueStatusResetting:
		return lipgloss.Color("#8B5CF6") // Purple for resetting
	case RescueStatusEnabling:
		return lipgloss.Color("#3B82F6") // Blue for enabling
	case RescueStatusActivating:
		return lipgloss.Color("#3B82F6") // Blue for activating
	case RescueStatusDeactivating:
		return lipgloss.Color("#F59E0B") // Orange for deactivating
	case RescueStatusRebooting:
		return lipgloss.Color("#8B5CF6") // Purple for rebooting
	case RescueStatusConnecting:
		return lipgloss.Color("#F59E0B") // Orange for connecting
	case RescueStatusFailed:
		return lipgloss.Color("#EF4444") // Red for failed
	default:
		if inRescueMode {
			return lipgloss.Color("#00D4AA") // Green for in rescue
		}
		return lipgloss.Color("#6B7280") // Gray for not in rescue
	}
}

// formatRescueStatus adds emoji indicators to rescue status
func formatRescueStatus(inRescueMode bool, status string) string {
	switch status {
	case RescueStatusReady:
		return "✅ Ready"
	case RescueStatusPinging:
		return "🏓 Pinging"
	case RescueStatusResetting:
		return "🔄 Resetting"
	case RescueStatusEnabling:
		return "🛠️ Enabling"
	case RescueStatusActivating:
		return "🚀 Activating"
	case RescueStatusDeactivating:
		return "🔄 Deactivating"
	case RescueStatusRebooting:
		return "🔄 Rebooting"
	case RescueStatusConnecting:
		return "🔗 Connecting"
	case RescueStatusFailed:
		return "❌ Failed"
	default:
		if inRescueMode {
			return "✅ In Rescue"
		}
		return "⚪ Not in Rescue"
	}
}

// monitorServerReadiness monitors servers until they are ready or timeout
func monitorServerReadiness(ctx context.Context, client *Client, statuses []ServerRescueStatus) (*RescueManagementResult, error) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("⏳"),
		styles.TitleStyle.Render("Monitoring server readiness..."))

	startTime := time.Now()
	ticker := time.NewTicker(RescueCheckInterval)
	defer ticker.Stop()

	// Update all statuses to pinging
	for i := range statuses {
		statuses[i].Status = RescueStatusPinging
	}

	for {
		// Check if timeout reached
		if time.Since(startTime) >= RescueTimeout {
			fmt.Printf("\n%s %s\n",
				styles.CommandStyle.Render("⏰"),
				styles.CommandStyle.Render("Timeout reached (6 minutes). Some servers may not be ready."))

			return &RescueManagementResult{
				Success:        false,
				TimeoutReached: true,
				FailedServers:  getFailedRescueServers(statuses),
				RescuePasswords: extractRescuePasswords(statuses),
			}, fmt.Errorf("rescue timeout: not all servers ready within 6 minutes")
		}

		// Check server readiness via ICMP ping
		updateServerReadiness(statuses)

		// Display current status
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("🔄"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Checking server readiness... (%.0fs elapsed)",
				time.Since(startTime).Seconds())))

		displayRescueStatusTable(statuses)

		// Check if all servers are ready
		allReady, anyFailed, failedServers := analyzeRescueStatuses(statuses)

		if anyFailed {
			return &RescueManagementResult{
				Success:       false,
				FailedServers: failedServers,
				RescuePasswords: extractRescuePasswords(statuses),
			}, fmt.Errorf("server rescue failed: %s", getFailedRescueServerNames(failedServers))
		}

		if allReady {
			fmt.Printf("\n%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render("All servers are ready in rescue mode!"))

			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🎯"),
				styles.DescriptionStyle.Render("Rescue passwords saved for Talos installation"))

			return &RescueManagementResult{
				Success:         true,
				ReadyServers:    statuses,
				RescuePasswords: extractRescuePasswords(statuses),
			}, nil
		}

		// Wait for next check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}

// updateServerReadiness checks if servers are responding via ICMP ping
func updateServerReadiness(statuses []ServerRescueStatus) {
	for i := range statuses {
		if statuses[i].Status == RescueStatusFailed || statuses[i].Status == RescueStatusReady {
			continue // Skip servers that are already in final state
		}

		// Try to ping the server
		if pingServer(statuses[i].ServerIP) {
			statuses[i].Status = RescueStatusReady
		} else {
			// Keep current status if not ready yet
			if statuses[i].Status != RescueStatusPinging {
				statuses[i].Status = RescueStatusPinging
			}
		}
	}
}

// pingServer attempts to ping a server via ICMP
func pingServer(serverIP string) bool {
	// Create a new pinger
	pinger, err := ping.NewPinger(serverIP)
	if err != nil {
		return false // Failed to create pinger
	}

	// For unprivileged mode (works without root)
	pinger.SetPrivileged(false)

	// Set timeout and count
	pinger.Timeout = 3 * time.Second
	pinger.Count = 1

	// Resolve the address to ensure it's valid
	if _, err := net.ResolveIPAddr("ip4", serverIP); err != nil {
		return false // Invalid IP address
	}

	// Run the ping
	err = pinger.Run()
	if err != nil {
		return false // Ping failed
	}

	// Check if we received any packets
	stats := pinger.Statistics()
	return stats.PacketsRecv > 0
}

// analyzeRescueStatuses checks if all servers are ready or if any failed
func analyzeRescueStatuses(statuses []ServerRescueStatus) (allReady bool, anyFailed bool, failedServers []ServerRescueStatus) {
	allReady = true
	anyFailed = false
	failedServers = make([]ServerRescueStatus, 0)

	for _, status := range statuses {
		if status.Status == RescueStatusFailed {
			anyFailed = true
			failedServers = append(failedServers, status)
		} else if status.Status != RescueStatusReady {
			allReady = false
		}
	}

	return allReady, anyFailed, failedServers
}

// getFailedRescueServers returns servers that failed
func getFailedRescueServers(statuses []ServerRescueStatus) []ServerRescueStatus {
	failedServers := make([]ServerRescueStatus, 0)
	for _, status := range statuses {
		if status.Status == RescueStatusFailed {
			failedServers = append(failedServers, status)
		}
	}
	return failedServers
}

// getFailedRescueServerNames returns a comma-separated list of failed server names
func getFailedRescueServerNames(failedServers []ServerRescueStatus) string {
	names := make([]string, len(failedServers))
	for i, server := range failedServers {
		names[i] = server.ServerName
	}
	return strings.Join(names, ", ")
}

// extractRescuePasswords creates a map of server IDs to rescue passwords
func extractRescuePasswords(statuses []ServerRescueStatus) map[string]string {
	passwords := make(map[string]string)
	for _, status := range statuses {
		if status.RescuePassword != "" {
			passwords[status.ServerID] = status.RescuePassword
		}
	}
	return passwords
}

// getServerNames returns a comma-separated list of server names
func getServerNames(servers []Server) string {
	names := make([]string, len(servers))
	for i, server := range servers {
		names[i] = server.Name
	}
	return strings.Join(names, ", ")
}
