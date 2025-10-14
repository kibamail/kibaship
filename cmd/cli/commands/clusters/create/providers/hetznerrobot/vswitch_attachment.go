package hetznerrobot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// AttachmentStatus represents the status of server attachment to vswitch
type AttachmentStatus struct {
	ServerID   string
	ServerName string
	ServerIP   string
	Status     string // "ready", "in process", "failed", "not attached"
	IsAttached bool
}

// VSwitchAttachmentResult contains the result of the attachment process
type VSwitchAttachmentResult struct {
	Success         bool
	AttachedServers []AttachmentStatus
	FailedServers   []AttachmentStatus
	TimeoutReached  bool
}

const (
	AttachmentTimeout = 6 * time.Minute
	CheckInterval     = 15 * time.Second
	StatusReady       = "ready"
	StatusInProcess   = "in process"
	StatusFailed      = "failed"
	StatusNotAttached = "not attached"
)

// ProcessVSwitchAttachment handles the complete vswitch attachment process
func ProcessVSwitchAttachment(ctx context.Context, client *Client, vswitchResult *VSwitchSelectionResult, selectedServers []Server) (*VSwitchAttachmentResult, error) {
	// Step 1: Create vswitch if needed
	if vswitchResult.CreateNew {
		fmt.Printf("\n%s %s\n",
			styles.TitleStyle.Render("🔗"),
			styles.TitleStyle.Render("Creating VSwitch..."))

		vswitch, err := client.CreateVSwitch(ctx, vswitchResult.NewVSwitchName, vswitchResult.NewVSwitchVLAN)
		if err != nil {
			return nil, fmt.Errorf("failed to create vswitch: %w", err)
		}

		// Update the result with the created vswitch
		vswitchResult.SelectedVSwitch = vswitch
		vswitchResult.CreateNew = false

		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("✅"),
			styles.DescriptionStyle.Render(fmt.Sprintf("VSwitch '%s' created successfully (ID: %s, VLAN: %d)",
				vswitch.Name, vswitch.ID, vswitch.VLAN)))

		// Wait 30 seconds for vSwitch to become active before attaching servers
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("⏳"),
			styles.DescriptionStyle.Render("Waiting 30 seconds for vSwitch to become active..."))

		waitDuration := 30 * time.Second
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
				return nil, ctx.Err()
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
			styles.TitleStyle.Render("VSwitch is now ready for server attachment"))
	}

	// Step 2: Get current vswitch details and check attachment status
	vswitchDetails, err := client.GetVSwitchDetails(ctx, vswitchResult.SelectedVSwitch.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vswitch details: %w", err)
	}

	// Step 3: Determine which servers need to be attached
	attachmentStatuses := determineAttachmentStatuses(selectedServers, vswitchDetails.AttachedServers)

	// Step 4: Display initial status table
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("📊"),
		styles.TitleStyle.Render("Server Attachment Status"))

	displayAttachmentTable(attachmentStatuses)

	// Step 5: Attach servers that are not attached
	serversToAttach := getServersToAttach(attachmentStatuses)
	if len(serversToAttach) > 0 {
		if err := attachServersToVSwitch(ctx, client, vswitchResult.SelectedVSwitch.ID, serversToAttach); err != nil {
			return nil, fmt.Errorf("failed to attach servers: %w", err)
		}
	}

	// Step 6: Monitor attachment status until all are ready or timeout
	result, err := monitorAttachmentStatus(ctx, client, vswitchResult.SelectedVSwitch.ID, selectedServers)
	if err != nil {
		return nil, fmt.Errorf("failed to monitor attachment status: %w", err)
	}

	return result, nil
}

// determineAttachmentStatuses compares selected servers with attached servers to determine status
func determineAttachmentStatuses(selectedServers []Server, attachedServers []VSwitchServer) []AttachmentStatus {
	statuses := make([]AttachmentStatus, 0, len(selectedServers))

	for _, server := range selectedServers {
		status := AttachmentStatus{
			ServerID:   server.ID,
			ServerName: server.Name,
			ServerIP:   server.IP,
			Status:     StatusNotAttached,
			IsAttached: false,
		}

		// Check if server is already attached
		serverIDInt, _ := strconv.Atoi(server.ID)
		for _, attached := range attachedServers {
			if attached.ServerNumber == serverIDInt {
				status.Status = attached.Status
				status.IsAttached = true
				break
			}
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// getServersToAttach returns servers that need to be attached
func getServersToAttach(statuses []AttachmentStatus) []AttachmentStatus {
	serversToAttach := make([]AttachmentStatus, 0)
	for _, status := range statuses {
		if !status.IsAttached {
			serversToAttach = append(serversToAttach, status)
		}
	}
	return serversToAttach
}

// attachServersToVSwitch attaches multiple servers to the vswitch in a single API call
func attachServersToVSwitch(ctx context.Context, client *Client, vswitchID string, serversToAttach []AttachmentStatus) error {
	if len(serversToAttach) == 0 {
		return nil
	}

	// Display which servers are being attached
	serverNames := make([]string, len(serversToAttach))
	serverIDs := make([]string, len(serversToAttach))
	for i, server := range serversToAttach {
		serverNames[i] = server.ServerName
		serverIDs[i] = server.ServerID
	}

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔗"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%s not attached to vswitch. Attaching now...",
			strings.Join(serverNames, ", "))))

	fmt.Printf("%s %s\n",
		styles.DescriptionStyle.Render("  →"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Attaching %d servers in single API call...", len(serversToAttach))))

	// Attach all servers in a single API call
	if err := client.AttachServersToVSwitch(ctx, vswitchID, serverIDs); err != nil {
		return fmt.Errorf("failed to attach servers: %w", err)
	}

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("✅"),
		styles.DescriptionStyle.Render("Attachment request sent successfully for all servers"))

	return nil
}

// displayAttachmentTable shows the current attachment status in a table
func displayAttachmentTable(statuses []AttachmentStatus) {
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
			if col == 3 && row > 0 && row-1 < len(statuses) {
				return statusStyle.Copy().Foreground(getAttachmentStatusColor(statuses[row-1].Status))
			}

			return cellStyle
		}).
		Headers("Server", "ID", "IP Address", "Attachment Status")

	// Add server rows
	for _, status := range statuses {
		t.Row(
			status.ServerName,
			status.ServerID,
			status.ServerIP,
			formatAttachmentStatus(status.Status),
		)
	}

	fmt.Println(t.Render())
}

// getAttachmentStatusColor returns appropriate color for attachment status
func getAttachmentStatusColor(status string) lipgloss.Color {
	switch status {
	case StatusReady:
		return lipgloss.Color("#00D4AA") // Green for ready
	case StatusInProcess:
		return lipgloss.Color("#F59E0B") // Orange for in process
	case StatusFailed:
		return lipgloss.Color("#EF4444") // Red for failed
	case StatusNotAttached:
		return lipgloss.Color("#6B7280") // Gray for not attached
	default:
		return lipgloss.Color("#6B7280") // Gray for unknown
	}
}

// formatAttachmentStatus adds emoji indicators to status
func formatAttachmentStatus(status string) string {
	switch status {
	case StatusReady:
		return "✅ Ready"
	case StatusInProcess:
		return "⏳ In Process"
	case StatusFailed:
		return "❌ Failed"
	case StatusNotAttached:
		return "⚪ Not Attached"
	default:
		return "❓ " + status
	}
}

// monitorAttachmentStatus monitors the attachment status until all servers are ready or timeout
func monitorAttachmentStatus(ctx context.Context, client *Client, vswitchID string, selectedServers []Server) (*VSwitchAttachmentResult, error) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("⏳"),
		styles.TitleStyle.Render("Monitoring attachment status..."))

	startTime := time.Now()
	ticker := time.NewTicker(CheckInterval)
	defer ticker.Stop()

	for {
		// Check if timeout reached
		if time.Since(startTime) >= AttachmentTimeout {
			fmt.Printf("\n%s %s\n",
				styles.CommandStyle.Render("⏰"),
				styles.CommandStyle.Render("Timeout reached (6 minutes). Some servers may not be ready."))

			// Get final status
			finalStatuses, err := getCurrentAttachmentStatuses(ctx, client, vswitchID, selectedServers)
			if err != nil {
				return nil, fmt.Errorf("failed to get final attachment status: %w", err)
			}

			return &VSwitchAttachmentResult{
				Success:        false,
				TimeoutReached: true,
				FailedServers:  getFailedServers(finalStatuses),
			}, fmt.Errorf("attachment timeout: not all servers attached within 6 minutes")
		}

		// Get current attachment status
		currentStatuses, err := getCurrentAttachmentStatuses(ctx, client, vswitchID, selectedServers)
		if err != nil {
			return nil, fmt.Errorf("failed to get attachment status: %w", err)
		}

		// Display current status
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("🔄"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Checking attachment status... (%.0fs elapsed)",
				time.Since(startTime).Seconds())))

		displayAttachmentTable(currentStatuses)

		// Check if all servers are ready or if any failed
		allReady, anyFailed, failedServers := analyzeAttachmentStatuses(currentStatuses)

		if anyFailed {
			return &VSwitchAttachmentResult{
				Success:       false,
				FailedServers: failedServers,
			}, fmt.Errorf("server attachment failed: %s", getFailedServerNames(failedServers))
		}

		if allReady {
			fmt.Printf("\n%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render("All servers successfully attached to vswitch!"))

			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🚀"),
				styles.DescriptionStyle.Render("Proceeding to install Talos..."))

			return &VSwitchAttachmentResult{
				Success:         true,
				AttachedServers: currentStatuses,
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

// getCurrentAttachmentStatuses fetches the current attachment status for all selected servers
func getCurrentAttachmentStatuses(ctx context.Context, client *Client, vswitchID string, selectedServers []Server) ([]AttachmentStatus, error) {
	vswitchDetails, err := client.GetVSwitchDetails(ctx, vswitchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vswitch details: %w", err)
	}

	return determineAttachmentStatuses(selectedServers, vswitchDetails.AttachedServers), nil
}

// analyzeAttachmentStatuses checks if all servers are ready or if any failed
func analyzeAttachmentStatuses(statuses []AttachmentStatus) (allReady bool, anyFailed bool, failedServers []AttachmentStatus) {
	allReady = true
	anyFailed = false
	failedServers = make([]AttachmentStatus, 0)

	for _, status := range statuses {
		if status.Status == StatusFailed {
			anyFailed = true
			failedServers = append(failedServers, status)
		} else if status.Status != StatusReady {
			allReady = false
		}
	}

	return allReady, anyFailed, failedServers
}

// getFailedServers returns servers that failed to attach
func getFailedServers(statuses []AttachmentStatus) []AttachmentStatus {
	failedServers := make([]AttachmentStatus, 0)
	for _, status := range statuses {
		if status.Status == StatusFailed {
			failedServers = append(failedServers, status)
		}
	}
	return failedServers
}

// getFailedServerNames returns a comma-separated list of failed server names
func getFailedServerNames(failedServers []AttachmentStatus) string {
	names := make([]string, len(failedServers))
	for i, server := range failedServers {
		names[i] = server.ServerName
	}
	return strings.Join(names, ", ")
}
