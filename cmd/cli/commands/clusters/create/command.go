package create

import (
	"fmt"
	"os"

	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
	"github.com/spf13/cobra"
)

// NewCommand creates and returns the clusters create command with all flags configured
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Kubernetes cluster",
		Long:  "Create a new Kubernetes cluster on your chosen cloud provider.",
		Run:   runCreate,
	}

	// Configure flags
	setupFlags(cmd)

	// Override help command behavior
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		PrintHelp()
	})

	return cmd
}

// setupFlags configures all command-line flags for the create command
func setupFlags(cmd *cobra.Command) {
	// Only configuration file flag - all other settings must be in YAML
	cmd.Flags().StringP("configuration", "c", "", "Path to YAML configuration file (required)")

	// Mark configuration as required
	cmd.MarkFlagRequired("configuration")
}

// runCreate is the main execution function for the create command
func runCreate(cmd *cobra.Command, args []string) {
	// Print banner
	styles.PrintBanner()
	fmt.Printf("%s %s\n\n",
		styles.TitleStyle.Render("🚀"),
		styles.TitleStyle.Render("Kibaship Cluster Creation"))

	// Get configuration file path
	configuration, _ := cmd.Flags().GetString("configuration")

	// Load and validate configuration from YAML file
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📄"),
		styles.HelpStyle.Render("Loading cluster configuration..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Configuration file: %s", configuration)))

	config, err := LoadConfigFromYAML(configuration)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Failed to load configuration: %v", err)))
		os.Exit(1)
	}

	// Configuration validated successfully
	fmt.Printf("%s %s\n\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cluster creation validated successfully!"))

	fmt.Printf("%s %s\n",
		styles.HelpStyle.Render("📋"),
		styles.HelpStyle.Render("Configuration Summary:"))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Domain:"),
		styles.DescriptionStyle.Render(config.Domain))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Cluster Name:"),
		styles.DescriptionStyle.Render(config.Name))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Email:"),
		styles.DescriptionStyle.Render(config.Email))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Provider:"),
		styles.DescriptionStyle.Render(config.Provider))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("PaaS Features:"),
		styles.DescriptionStyle.Render(config.PaaSFeatures))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Terraform State Bucket:"),
		styles.DescriptionStyle.Render(config.TerraformState.S3Bucket))
	fmt.Printf("   %s %s\n",
		styles.CommandStyle.Render("Terraform State Region:"),
		styles.DescriptionStyle.Render(config.TerraformState.S3Region))

	fmt.Printf("\n%s %s %s\n",
		styles.CommandStyle.Render("📄"),
		styles.DescriptionStyle.Render("Configuration loaded from YAML file:"),
		styles.TitleStyle.Render(configuration))

	// Show provider-specific configuration
	switch config.Provider {
	case "digital-ocean":
		if config.DigitalOcean != nil {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("🌊"),
				styles.HelpStyle.Render("DigitalOcean Configuration:"))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Nodes:"),
				styles.DescriptionStyle.Render(config.DigitalOcean.Nodes))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Node Size:"),
				styles.DescriptionStyle.Render(config.DigitalOcean.NodesSize))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Region:"),
				styles.DescriptionStyle.Render(config.DigitalOcean.Region))
		}
	case "aws":
		if config.AWS != nil {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("☁️"),
				styles.HelpStyle.Render("AWS Configuration:"))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Region:"),
				styles.DescriptionStyle.Render(config.AWS.Region))
		}
	case "hetzner":
		if config.Hetzner != nil {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("🇩🇪"),
				styles.DescriptionStyle.Render("Hetzner Cloud Configuration: Ready"))
		}
	case "hetzner-robot":
		if config.HetznerRobot != nil {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("🤖"),
				styles.DescriptionStyle.Render("Hetzner Robot Configuration: Ready"))
		}
	case "linode":
		if config.Linode != nil {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("🔗"),
				styles.DescriptionStyle.Render("Linode Configuration: Ready"))
		}
	case "gcloud":
		if config.GCloud != nil {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("☁️"),
				styles.HelpStyle.Render("Google Cloud Configuration:"))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Project ID:"),
				styles.DescriptionStyle.Render(config.GCloud.ProjectID))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Region:"),
				styles.DescriptionStyle.Render(config.GCloud.Region))
		}
	case "kind":
		if config.Kind != nil {
			fmt.Printf("\n%s %s\n",
				styles.HelpStyle.Render("🐳"),
				styles.HelpStyle.Render("Kind Configuration:"))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Nodes:"),
				styles.DescriptionStyle.Render(config.Kind.Nodes))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Type:"),
				styles.DescriptionStyle.Render("Local Docker containers"))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Network:"),
				styles.DescriptionStyle.Render("kind"))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("CNI:"),
				styles.DescriptionStyle.Render("Disabled (ready for Cilium)"))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Kube-proxy:"),
				styles.DescriptionStyle.Render("Disabled"))
			fmt.Printf("   %s %s\n",
				styles.CommandStyle.Render("Longhorn Storage:"),
				styles.DescriptionStyle.Render(fmt.Sprintf("%s per node", config.Kind.Storage+"GB")))
		}
	}

	// Build Terraform files
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
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📁"),
		styles.DescriptionStyle.Render(fmt.Sprintf("Files created in: .kibaship/%s/", config.Name)))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("• provision/main.tf"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("• provision/vars.tf"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("• bootstrap/main.tf"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("• bootstrap/vars.tf"))

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

	// Run Terraform validate
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Validating Terraform configuration..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform validate"))

	if err := runTerraformValidate(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Terraform validate failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Terraform configuration is valid!"))

	// Run Terraform apply
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🚀"),
		styles.HelpStyle.Render("Provisioning cluster infrastructure..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform apply -auto-approve"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("⚠️"),
		styles.DescriptionStyle.Render("This may take several minutes to complete..."))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🕰️"),
		styles.DescriptionStyle.Render("Please wait while the cluster is being created..."))

	if err := runTerraformApply(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Terraform apply failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cluster provisioning completed successfully!"))
	fmt.Printf("%s %s\n",
		styles.TitleStyle.Render("🎉"),
		styles.TitleStyle.Render("Your Kubernetes cluster is now live!"))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Next steps: Configure kubectl and deploy your applications"))
}
