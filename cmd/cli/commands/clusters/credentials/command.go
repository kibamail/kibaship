package credentials

import (
	"fmt"
	"os"

	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
	"github.com/spf13/cobra"
)

// NewCommand creates and returns the clusters credentials command
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Extract cluster credentials from Terraform state",
		Long:  "Extract cluster credentials (kubeconfig, etc.) from Terraform state and save them to local files.",
		Run:   runCredentialsCommand,
	}

	setupFlags(cmd)
	return cmd
}

// setupFlags configures all command-line flags for the credentials command
func setupFlags(cmd *cobra.Command) {
	// Only configuration file flag - same as create and delete commands
	cmd.Flags().StringP("configuration", "c", "", "Path to YAML configuration file (required)")

	// Mark configuration as required
	_ = cmd.MarkFlagRequired("configuration")
}

// runCredentialsCommand executes the credentials extraction process
func runCredentialsCommand(cmd *cobra.Command, args []string) {
	// Print banner
	styles.PrintBanner()
	fmt.Printf("%s %s\n\n",
		styles.TitleStyle.Render("🔑"),
		styles.TitleStyle.Render("Kibaship Cluster Credentials"))

	// Get configuration file path
	configuration, _ := cmd.Flags().GetString("configuration")

	// Load and validate configuration
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📄"),
		styles.HelpStyle.Render("Loading cluster configuration..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Configuration file: %s", configuration)))

	config, err := create.LoadConfigFromYAML(configuration)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Failed to load configuration: %v", err)))
		os.Exit(1)
	}

	// Configuration loaded successfully
	fmt.Printf("%s %s\n\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Configuration loaded successfully!"))

	fmt.Printf("%s %s\n",
		styles.HelpStyle.Render("📋"),
		styles.HelpStyle.Render("Cluster Information:"))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Domain:"),
		styles.DescriptionStyle.Render(config.Domain))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Cluster Name:"),
		styles.DescriptionStyle.Render(config.Name))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Provider:"),
		styles.DescriptionStyle.Render(config.Provider))

	// Check if Terraform is installed
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Checking Terraform installation..."))
	if err := checkTerraformInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(err.Error()))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Terraform is installed and available"))

	// Check if cluster directory exists
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📂"),
		styles.HelpStyle.Render("Checking cluster directory..."))
	clusterDir := fmt.Sprintf(".kibaship/%s", config.Name)
	if _, err := os.Stat(clusterDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Cluster directory not found: %s", clusterDir)))
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("💡"),
			styles.DescriptionStyle.Render("Run 'kibaship clusters create' first to provision the cluster."))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cluster directory found"))

	// Create credentials directory
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.HelpStyle.Render("Creating credentials directory..."))
	credentialsDir := fmt.Sprintf("%s/credentials", clusterDir)
	if err := os.MkdirAll(credentialsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Failed to create credentials directory: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Credentials directory created"))

	// Extract credentials from Terraform output
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🔑"),
		styles.HelpStyle.Render("Extracting cluster credentials..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform output"))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📂"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Working directory: %s/provision", clusterDir)))

	if err := extractCredentials(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Failed to extract credentials: %v", err)))
		os.Exit(1)
	}

	// Success message and usage instructions
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Credentials extracted successfully!"))
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("🎉"),
		styles.TitleStyle.Render("Cluster credentials are ready!"))

	// Show usage instructions
	showUsageInstructions(config)
}
