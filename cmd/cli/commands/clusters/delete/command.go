package delete

import (
	"fmt"
	"os"

	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create"
	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
	"github.com/spf13/cobra"
)

// NewCommand creates and returns the clusters delete command
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a Kubernetes cluster",
		Long:  "Delete a Kubernetes cluster using the same configuration file used to create it.",
		Run:   runDeleteCommand,
	}

	setupFlags(cmd)
	return cmd
}

// setupFlags configures all command-line flags for the delete command
func setupFlags(cmd *cobra.Command) {
	// Core flags
	cmd.Flags().StringP("configuration", "c", "", "Path to YAML configuration file used to create the cluster")

	// Mark configuration as required
	cmd.MarkFlagRequired("configuration")
}

// runDeleteCommand executes the cluster deletion process
func runDeleteCommand(cmd *cobra.Command, args []string) {
	// Print banner
	styles.PrintBanner()
	fmt.Printf("%s %s\n\n",
		styles.TitleStyle.Render("🗑️"),
		styles.TitleStyle.Render("Kibaship Cluster Deletion"))

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
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Terraform State Bucket:"),
		styles.DescriptionStyle.Render(config.TerraformState.S3Bucket))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Terraform State Region:"),
		styles.DescriptionStyle.Render(config.TerraformState.S3Region))

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
		styles.DescriptionStyle.Render("This will permanently delete the Kubernetes cluster and all its resources!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("💾"),
		styles.DescriptionStyle.Render("Make sure you have backed up any important data."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🔄"),
		styles.DescriptionStyle.Render("This action cannot be undone."))

	// Confirmation prompt
	fmt.Printf("%s %s",
		styles.CommandStyle.Render("❓"),
		styles.HelpStyle.Render("Are you sure you want to delete this cluster? (type 'yes' to confirm): "))

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("🛑"),
			styles.DescriptionStyle.Render("Cluster deletion cancelled."))
		os.Exit(0)
	}

	// Proceed with deletion
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Proceeding with cluster deletion..."))

	// Build Terraform files for deletion
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
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform init with S3 backend configuration"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📄"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Backend: s3://%s/clusters/%s/provision.terraform.tfstate", config.TerraformState.S3Bucket, config.Name)))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🌍"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Region: %s", config.TerraformState.S3Region)))

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
		styles.TitleStyle.Render("Kubernetes cluster has been deleted!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🧹"),
		styles.DescriptionStyle.Render("All infrastructure resources have been removed."))
}
