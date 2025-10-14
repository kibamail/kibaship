package hetznerrobot

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// VSwitchSelectionResult contains the user's vswitch selection choices
type VSwitchSelectionResult struct {
	SelectedVSwitch *VSwitch
	CreateNew       bool
	NewVSwitchName  string
	NewVSwitchVLAN  int
}

// VSwitchOption represents different vswitch selection options
type VSwitchOption struct {
	Value       string
	Label       string
	Description string
}

// SelectVSwitchInteractive presents an interactive form for vswitch selection
func SelectVSwitchInteractive(ctx context.Context, client *Client, clusterName string) (*VSwitchSelectionResult, error) {
	// First, fetch and display ALL vswitches
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🔗"),
		styles.TitleStyle.Render("Hetzner Robot VSwitch Selection"))

	vswitches, err := DisplayVSwitchesTable(ctx, client, DefaultVSwitchDisplayOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vswitches: %w", err)
	}

	// If no vswitches, show message and return create new option
	if len(vswitches) == 0 {
		DisplayNoVSwitchesMessage()

		// Generate a random VLAN ID in the valid range (4000-4091)
		rand.Seed(time.Now().UnixNano())
		randomVLAN := 4000 + rand.Intn(92) // 4000 to 4091

		return &VSwitchSelectionResult{
			CreateNew:      true,
			NewVSwitchName: clusterName,
			NewVSwitchVLAN: randomVLAN,
		}, nil
	}

	// Create vswitch selection options
	vswitchOptions := make([]huh.Option[string], 0, len(vswitches)+1)

	// Add existing vswitches as options
	for _, vswitch := range vswitches {
		if !vswitch.Cancelled {
			label := fmt.Sprintf("%s (%s) - VLAN %d",
				vswitch.Name,
				vswitch.ID,
				vswitch.VLAN)
			vswitchOptions = append(vswitchOptions, huh.NewOption(label, vswitch.ID))
		}
	}

	// Add "Create New" option with cluster name
	createNewLabel := fmt.Sprintf("🆕 Create one for %s", clusterName)
	vswitchOptions = append(vswitchOptions, huh.NewOption(createNewLabel, "CREATE_NEW"))

	// Form variables
	var selectedVSwitchID string

	// Create the interactive form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("🔗 VSwitch Selection").
				Description("Select an existing vswitch for private networking between your servers,\n"+
					"or choose to create a new one. VSwitches enable secure internal communication."),

			huh.NewSelect[string]().
				Title("Choose vswitch option").
				Description("Select an existing vswitch or create a new one").
				Options(vswitchOptions...).
				Value(&selectedVSwitchID),
		),
	).WithTheme(createFormTheme())

	// Run the form
	err = form.Run()
	if err != nil {
		return nil, fmt.Errorf("vswitch selection failed: %w", err)
	}

	// Process the selection
	if selectedVSwitchID == "CREATE_NEW" {
		// Generate a random VLAN ID in the valid range (4000-4091)
		rand.Seed(time.Now().UnixNano())
		randomVLAN := 4000 + rand.Intn(92) // 4000 to 4091

		return &VSwitchSelectionResult{
			CreateNew:      true,
			NewVSwitchName: clusterName,
			NewVSwitchVLAN: randomVLAN,
		}, nil
	}

	// Find the selected vswitch
	for _, vswitch := range vswitches {
		if vswitch.ID == selectedVSwitchID {
			return &VSwitchSelectionResult{
				SelectedVSwitch: &vswitch,
				CreateNew:       false,
			}, nil
		}
	}

	return nil, fmt.Errorf("selected vswitch not found")
}

// DisplayVSwitchSelectionSummary shows a summary of the vswitch selection
func DisplayVSwitchSelectionSummary(result *VSwitchSelectionResult) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("VSwitch Selection Summary"))

	if result.CreateNew {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("Action:"),
			styles.DescriptionStyle.Render("Create new vswitch"))

		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("Name:"),
			styles.DescriptionStyle.Render(result.NewVSwitchName))

		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("VLAN ID:"),
			styles.DescriptionStyle.Render(strconv.Itoa(result.NewVSwitchVLAN)))
	} else {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("Action:"),
			styles.DescriptionStyle.Render("Use existing vswitch"))

		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("Name:"),
			styles.DescriptionStyle.Render(result.SelectedVSwitch.Name))

		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("ID:"),
			styles.DescriptionStyle.Render(result.SelectedVSwitch.ID))

		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("VLAN ID:"),
			styles.DescriptionStyle.Render(strconv.Itoa(result.SelectedVSwitch.VLAN)))
	}
	fmt.Println()
}

// CreateVSwitchIfNeeded creates a new vswitch if the user selected the create option
func CreateVSwitchIfNeeded(ctx context.Context, client *Client, result *VSwitchSelectionResult) error {
	if !result.CreateNew {
		return nil // Nothing to create
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🔗"),
		styles.TitleStyle.Render("Creating new vswitch..."))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Creating vswitch '%s' with VLAN %d",
			result.NewVSwitchName, result.NewVSwitchVLAN)))

	vswitch, err := client.CreateVSwitch(ctx, result.NewVSwitchName, result.NewVSwitchVLAN)
	if err != nil {
		return fmt.Errorf("failed to create vswitch: %w", err)
	}

	// Update the result with the created vswitch
	result.SelectedVSwitch = vswitch
	result.CreateNew = false // Mark as no longer needing creation

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("VSwitch created successfully!"))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("ID:"),
		styles.DescriptionStyle.Render(vswitch.ID))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("Name:"),
		styles.DescriptionStyle.Render(vswitch.Name))

	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("VLAN:"),
		styles.DescriptionStyle.Render(strconv.Itoa(vswitch.VLAN)))

	return nil
}

// DisplayFinalConfirmation shows the final configuration summary and asks for confirmation
func DisplayFinalConfirmation(serverResult *ServerSelectionResult, vswitchResult *VSwitchSelectionResult) (bool, error) {
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.TitleStyle.Render("Final Configuration Summary"))

	// Create a table showing servers and their vswitch configuration
	serverTable := createFinalServerTable(serverResult, vswitchResult)
	fmt.Println(serverTable)

	// Display vswitch information
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("Network Configuration:"),
		styles.DescriptionStyle.Render("Private networking via vswitch"))

	if vswitchResult.CreateNew {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.MutedColor)
		fmt.Printf("   %s %s (VLAN %d) - %s\n",
			styles.DescriptionStyle.Render("VSwitch:"),
			vswitchResult.NewVSwitchName,
			vswitchResult.NewVSwitchVLAN,
			mutedStyle.Render("Will be created"))
	} else if vswitchResult.SelectedVSwitch != nil {
		fmt.Printf("   %s %s (%s) - VLAN %d\n",
			styles.DescriptionStyle.Render("VSwitch:"),
			vswitchResult.SelectedVSwitch.Name,
			vswitchResult.SelectedVSwitch.ID,
			vswitchResult.SelectedVSwitch.VLAN)
	}

	// Create confirmation form with Enter/Esc prompt
	var proceed bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("⚡ Ready to Deploy").
				Description("Review the configuration above.\n"+
					"Press Enter to proceed with cluster creation or Esc to cancel."),

			huh.NewConfirm().
				Title("Proceed with cluster creation?").
				Description("This will start the Kubernetes cluster deployment process").
				Value(&proceed),
		),
	).WithTheme(createFormTheme())

	err := form.Run()
	if err != nil {
		return false, fmt.Errorf("confirmation failed: %w", err)
	}

	return proceed, nil
}

// createFinalServerTable creates a table showing selected servers with their vswitch configuration
func createFinalServerTable(serverResult *ServerSelectionResult, vswitchResult *VSwitchSelectionResult) string {
	// Define table styles
	headerStyle := lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Bold(true).
		Align(lipgloss.Center)

	cellStyle := lipgloss.NewStyle().
		Foreground(styles.TextColor).
		Align(lipgloss.Left).
		Padding(0, 1)

	// Determine vswitch name for display
	vswitchName := ""
	if vswitchResult.CreateNew {
		vswitchName = fmt.Sprintf("%s (will be created)", vswitchResult.NewVSwitchName)
	} else if vswitchResult.SelectedVSwitch != nil {
		vswitchName = vswitchResult.SelectedVSwitch.Name
	}

	// Get cluster type label
	clusterTypes := GetClusterTypeOptions()
	var clusterTypeLabel string
	for _, ct := range clusterTypes {
		if ct.Value == serverResult.ClusterType {
			clusterTypeLabel = ct.Label
			break
		}
	}

	// Create table
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.PrimaryColor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			return cellStyle
		}).
		Headers("Server", "ID", "IP Address", "Role", "VSwitch")

	// Add server rows
	for i, server := range serverResult.SelectedServers {
		role := getServerRole(i, serverResult.ClusterType, len(serverResult.SelectedServers))
		t.Row(
			server.Name,
			server.ID,
			server.IP,
			role,
			vswitchName,
		)
	}

	// Add cluster type info above the table
	clusterInfo := fmt.Sprintf("\n%s %s\n",
		styles.CommandStyle.Render("Cluster Type:"),
		styles.DescriptionStyle.Render(clusterTypeLabel))

	return clusterInfo + t.Render()
}
