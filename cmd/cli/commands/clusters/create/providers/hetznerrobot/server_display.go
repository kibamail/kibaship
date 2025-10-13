package hetznerrobot

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// ServerDisplayOptions configures how servers are displayed
type ServerDisplayOptions struct {
	ShowCancelled bool
	MaxNameWidth  int
}

// DefaultServerDisplayOptions returns sensible defaults for server display
func DefaultServerDisplayOptions() ServerDisplayOptions {
	return ServerDisplayOptions{
		ShowCancelled: false, // Hide cancelled servers by default
		MaxNameWidth:  20,    // Truncate long server names
	}
}

// DisplayServersTable fetches and displays servers in a beautiful table format
func DisplayServersTable(ctx context.Context, client *Client, options ServerDisplayOptions) error {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🤖"),
		styles.TitleStyle.Render("Fetching Hetzner Robot servers..."))

	servers, err := client.ListServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("ℹ️"),
			styles.DescriptionStyle.Render("No servers found in your Hetzner Robot account."))
		return nil
	}

	// Filter servers based on options
	filteredServers := filterServers(servers, options)

	if len(filteredServers) == 0 {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("ℹ️"),
			styles.DescriptionStyle.Render("No available servers found (cancelled servers are hidden)."))
		return nil
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render(fmt.Sprintf("Found %d available servers:", len(filteredServers))))

	// Create and display the table
	displayTable := createServersTable(filteredServers, options)
	fmt.Println(displayTable)

	return nil
}

// filterServers applies filtering based on display options
func filterServers(servers []Server, options ServerDisplayOptions) []Server {
	if options.ShowCancelled {
		return servers
	}

	filtered := make([]Server, 0, len(servers))
	for _, server := range servers {
		if !server.Cancelled {
			filtered = append(filtered, server)
		}
	}
	return filtered
}

// createServersTable creates a beautifully formatted table of servers
func createServersTable(servers []Server, options ServerDisplayOptions) string {
	// Define table styles using lipgloss
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

	// Create table with headers
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.PrimaryColor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}

			// Special styling for status column
			if col == 3 && row > 0 && row-1 < len(servers) { // Status column
				return statusStyle.Copy().Foreground(getStatusColor(servers[row-1].Status))
			}

			return cellStyle
		}).
		Headers("ID", "Name", "IP Address", "Status", "Product", "DC", "Traffic")

	// Add server rows
	for _, server := range servers {
		t.Row(
			server.ID,
			truncateString(server.Name, options.MaxNameWidth),
			server.IP,
			formatStatus(server.Status),
			server.Product,
			server.DC,
			server.Traffic,
		)
	}

	return t.Render()
}

// getStatusColor returns appropriate color for server status
func getStatusColor(status string) lipgloss.Color {
	switch strings.ToLower(status) {
	case "ready":
		return lipgloss.Color("#00D4AA") // Green for ready
	case "rescue":
		return lipgloss.Color("#F59E0B") // Orange for rescue
	case "installing":
		return lipgloss.Color("#3B82F6") // Blue for installing
	default:
		return lipgloss.Color("#EF4444") // Red for other statuses
	}
}

// formatStatus adds emoji indicators to status
func formatStatus(status string) string {
	switch strings.ToLower(status) {
	case "ready":
		return "✅ Ready"
	case "rescue":
		return "🔧 Rescue"
	case "installing":
		return "⚙️ Installing"
	default:
		return "❌ " + status
	}
}



// truncateString truncates a string to maxLength with ellipsis
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	if maxLength <= 3 {
		return s[:maxLength]
	}
	return s[:maxLength-3] + "..."
}

// DisplayServersSummary shows a brief summary of available servers
func DisplayServersSummary(ctx context.Context, client *Client) error {
	servers, err := client.ListServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}

	availableCount := 0
	cancelledCount := 0
	readyCount := 0

	for _, server := range servers {
		if server.Cancelled {
			cancelledCount++
		} else {
			availableCount++
			if strings.ToLower(server.Status) == "ready" {
				readyCount++
			}
		}
	}

	fmt.Printf("\n%s %s\n",
		styles.HelpStyle.Render("📊"),
		styles.HelpStyle.Render("Server Summary:"))
	
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Total servers:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d", len(servers))))
	
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Available:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d", availableCount)))
	
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Ready for use:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d", readyCount)))
	
	if cancelledCount > 0 {
		fmt.Printf("   %s %s\n",
			styles.CommandStyle.Render("Cancelled:"),
			styles.DescriptionStyle.Render(fmt.Sprintf("%d", cancelledCount)))
	}

	return nil
}
