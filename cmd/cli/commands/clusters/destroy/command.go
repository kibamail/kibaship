package destroy

import (
	"fmt"
	"os"

	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/config"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
	"github.com/spf13/cobra"
)

// NewCommand creates and returns the clusters destroy command
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a Kubernetes cluster",
		Long:  "Destroy a Kubernetes cluster using the same configuration file used to create it.",
		Run:   runDestroyCommand,
	}

	setupFlags(cmd)
	return cmd
}

// setupFlags configures all command-line flags for the destroy command
func setupFlags(cmd *cobra.Command) {
	// Core flags
	cmd.Flags().StringP("configuration", "c", "", "Path to YAML configuration file used to create the cluster")

	// Mark configuration as required
	_ = cmd.MarkFlagRequired("configuration")
}

// runDestroyCommand executes the cluster destruction process
func runDestroyCommand(cmd *cobra.Command, args []string) {
	// Print banner
	styles.PrintBanner()
	fmt.Printf("%s %s\n\n",
		styles.TitleStyle.Render("🗑️"),
		styles.TitleStyle.Render("Kibaship Cluster Destruction"))

	// Get configuration file path
	configuration, _ := cmd.Flags().GetString("configuration")

	// Load and validate configuration
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📄"),
		styles.HelpStyle.Render("Loading cluster configuration..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Configuration file: %s", configuration)))

	config, err := config.LoadConfigFromYAML(configuration)
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

	// Warning about destructive action
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("⚠️"),
		styles.HelpStyle.Render("DESTRUCTIVE ACTION WARNING"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🚨"),
		styles.DescriptionStyle.Render("This will permanently destroy the Kubernetes cluster and all its resources!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("💾"),
		styles.DescriptionStyle.Render("Make sure you have backed up any important data."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🔄"),
		styles.DescriptionStyle.Render("This action cannot be undone."))

	// Confirmation prompt
	fmt.Printf("%s %s",
		styles.CommandStyle.Render("❓"),
		styles.HelpStyle.Render("Are you sure you want to destroy this cluster? (type 'yes' to confirm): "))

	var confirmation string
	_, _ = fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("🛑"),
			styles.DescriptionStyle.Render("Cluster destruction cancelled."))
		os.Exit(0)
	}

	// Proceed with destruction
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Proceeding with cluster destruction..."))

	// Build Terraform files for destruction
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔨"),
		styles.HelpStyle.Render("Building Terraform files..."))
	if err := buildTerraformFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Error building Terraform files: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Terraform files built successfully!"))

	// Run Terraform init
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Initializing Terraform..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform init with local backend configuration"))

	if err := runTerraformInit(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Terraform init failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Terraform initialization completed successfully!"))

	// Run Terraform destroy
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🗑️"),
		styles.HelpStyle.Render("Destroying cluster infrastructure..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform destroy -auto-approve"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⚠️"),
		styles.DescriptionStyle.Render("This may take several minutes to complete..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🕰️"),
		styles.DescriptionStyle.Render("Please wait while the cluster is being destroyed..."))

	if err := runTerraformDestroy(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Terraform destroy failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cluster destruction completed successfully!"))
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("🎉"),
		styles.TitleStyle.Render("Kubernetes cluster has been destroyed!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🧹"),
		styles.DescriptionStyle.Render("All infrastructure resources have been removed."))
}
