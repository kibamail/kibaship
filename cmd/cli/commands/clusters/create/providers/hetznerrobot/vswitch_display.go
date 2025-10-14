package hetznerrobot

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// VSwitchDisplayOptions configures how vswitches are displayed
type VSwitchDisplayOptions struct {
	ShowCancelled bool
	MaxNameWidth  int
}

// DefaultVSwitchDisplayOptions returns sensible defaults for vswitch display
func DefaultVSwitchDisplayOptions() VSwitchDisplayOptions {
	return VSwitchDisplayOptions{
		ShowCancelled: false, // Hide cancelled vswitches by default
		MaxNameWidth:  25,    // Truncate long vswitch names
	}
}

// DisplayVSwitchesTable fetches and displays vswitches in a beautiful table format
func DisplayVSwitchesTable(ctx context.Context, client *Client, options VSwitchDisplayOptions) ([]VSwitch, error) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🔗"),
		styles.TitleStyle.Render("Fetching Hetzner Robot vswitches..."))

	vswitches, err := client.ListVSwitches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vswitches: %w", err)
	}

	if len(vswitches) == 0 {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("ℹ️"),
			styles.DescriptionStyle.Render("No vswitches found in your Hetzner Robot account."))
		return vswitches, nil
	}

	// Filter vswitches based on options
	filteredVSwitches := filterVSwitches(vswitches, options)

	if len(filteredVSwitches) == 0 {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("ℹ️"),
			styles.DescriptionStyle.Render("No available vswitches found (cancelled vswitches are hidden)."))
		return vswitches, nil
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render(fmt.Sprintf("Found %d available vswitches:", len(filteredVSwitches))))

	// Create and display the table
	displayTable := createVSwitchesTable(filteredVSwitches, options)
	fmt.Println(displayTable)

	return filteredVSwitches, nil
}

// filterVSwitches applies filtering based on display options
func filterVSwitches(vswitches []VSwitch, options VSwitchDisplayOptions) []VSwitch {
	if options.ShowCancelled {
		return vswitches
	}

	filtered := make([]VSwitch, 0, len(vswitches))
	for _, vswitch := range vswitches {
		if !vswitch.Cancelled {
			filtered = append(filtered, vswitch)
		}
	}
	return filtered
}

// createVSwitchesTable creates a beautifully formatted table of vswitches
func createVSwitchesTable(vswitches []VSwitch, options VSwitchDisplayOptions) string {
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
			if col == 3 && row > 0 && row-1 < len(vswitches) { // Status column
				return statusStyle.Copy().Foreground(getVSwitchStatusColor(vswitches[row-1].Cancelled))
			}

			return cellStyle
		}).
		Headers("ID", "Name", "VLAN", "Status")

	// Add vswitch rows
	for _, vswitch := range vswitches {
		t.Row(
			vswitch.ID,
			truncateString(vswitch.Name, options.MaxNameWidth),
			strconv.Itoa(vswitch.VLAN),
			formatVSwitchStatus(vswitch.Cancelled),
		)
	}

	return t.Render()
}

// getVSwitchStatusColor returns appropriate color for vswitch status
func getVSwitchStatusColor(cancelled bool) lipgloss.Color {
	if cancelled {
		return lipgloss.Color("#EF4444") // Red for cancelled
	}
	return lipgloss.Color("#00D4AA") // Green for active
}

// formatVSwitchStatus adds emoji indicators to status
func formatVSwitchStatus(cancelled bool) string {
	if cancelled {
		return "❌ Cancelled"
	}
	return "✅ Active"
}

// DisplayVSwitchesSummary shows a brief summary of available vswitches
func DisplayVSwitchesSummary(ctx context.Context, client *Client) error {
	vswitches, err := client.ListVSwitches(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch vswitches: %w", err)
	}

	availableCount := 0
	cancelledCount := 0

	for _, vswitch := range vswitches {
		if vswitch.Cancelled {
			cancelledCount++
		} else {
			availableCount++
		}
	}

	fmt.Printf("\n%s %s\n",
		styles.HelpStyle.Render("📊"),
		styles.HelpStyle.Render("VSwitch Summary:"))

	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Total vswitches:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d", len(vswitches))))

	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Available:"),
		styles.DescriptionStyle.Render(fmt.Sprintf("%d", availableCount)))

	if cancelledCount > 0 {
		fmt.Printf("   %s %s\n",
			styles.CommandStyle.Render("Cancelled:"),
			styles.DescriptionStyle.Render(fmt.Sprintf("%d", cancelledCount)))
	}

	return nil
}

// displayCompactVSwitchList shows vswitches in a compact, readable format
func displayCompactVSwitchList(vswitches []VSwitch) {
	for i, vswitch := range vswitches {
		statusColor := getVSwitchStatusColor(vswitch.Cancelled)

		vswitchInfo := lipgloss.NewStyle().
			Foreground(styles.TextColor).
			Render(fmt.Sprintf("  %d. %s (%s) - VLAN %d",
				i+1,
				vswitch.Name,
				vswitch.ID,
				vswitch.VLAN))

		statusBadge := lipgloss.NewStyle().
			Foreground(statusColor).
			Bold(true).
			Render(fmt.Sprintf("[%s]", formatVSwitchStatus(vswitch.Cancelled)))

		fmt.Printf("%s %s\n", vswitchInfo, statusBadge)
	}
	fmt.Println()
}

// DisplayNoVSwitchesMessage shows a message when no vswitches are found
func DisplayNoVSwitchesMessage() {
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔗"),
		styles.DescriptionStyle.Render("No vswitches found in your account."))

	fmt.Printf("%s %s\n",
		styles.HelpStyle.Render("ℹ️"),
		styles.HelpStyle.Render("We will create a new vswitch to enable private networking between your servers."))

	fmt.Printf("%s %s\n",
		styles.DescriptionStyle.Render("📝"),
		styles.DescriptionStyle.Render("A vswitch allows your servers to communicate privately using internal IP addresses."))
}
