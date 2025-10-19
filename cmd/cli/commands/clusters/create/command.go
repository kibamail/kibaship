package create

import (
	"fmt"
	"os"

	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/automation"
	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/config"
	"github.com/kibamail/kibaship/cmd/cli/commands/clusters/create/providers/hetznerrobot"
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

	// Reset flag to destroy existing infrastructure before creating
	cmd.Flags().Bool("reset", false, "Destroy existing cluster infrastructure before creating new one")

	// Resume flag to skip to a specific phase
	cmd.Flags().String("resume", "", "Resume from a specific phase (ubuntu, microk8s)")

	// Mark configuration as required
	_ = cmd.MarkFlagRequired("configuration")
}

// runCreate is the main execution function for the create command
func runCreate(cmd *cobra.Command, args []string) {
	// Print banner
	styles.PrintBanner()
	fmt.Printf("%s %s\n\n",
		styles.TitleStyle.Render("🚀"),
		styles.TitleStyle.Render("Kibaship Cluster Creation"))

	// Get configuration file path and flags
	configuration, _ := cmd.Flags().GetString("configuration")
	reset, _ := cmd.Flags().GetBool("reset")
	resume, _ := cmd.Flags().GetString("resume")

	// Load and validate configuration from YAML file
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

	// Set resume flag from command line if provided
	if resume != "" {
		config.Resume = resume
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("⏭️"),
			styles.TitleStyle.Render(fmt.Sprintf("RESUME MODE: Starting from %s phase", resume)))
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

	fmt.Printf("\n%s %s %s\n",
		styles.CommandStyle.Render("📄"),
		styles.DescriptionStyle.Render("Configuration loaded from YAML file:"),
		styles.TitleStyle.Render(configuration))

	// Show provider-specific configuration
	switch config.Provider {
	case ProviderDigitalOcean:
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

			// Perform server selection for Hetzner Robot
			if err := hetznerrobot.PerformServerSelection(config); err != nil {
				fmt.Fprintf(os.Stderr, "\n%s %s\n",
					styles.CommandStyle.Render("❌"),
					styles.CommandStyle.Render(fmt.Sprintf("Server selection failed: %v", err)))
				os.Exit(1)
			}

			// Execute Hetzner Robot specific flow and exit
			hetznerrobot.RunClusterCreationFlow(config)
			return
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
	}

	// Handle reset flag - destroy existing infrastructure
	if reset {
		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("⚠️"),
			styles.TitleStyle.Render("RESET MODE ENABLED"))
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("🔥"),
			styles.DescriptionStyle.Render("This will DESTROY all existing cluster infrastructure before recreating it."))
		fmt.Printf("%s %s\n",
			styles.CommandStyle.Render("📦"),
			styles.DescriptionStyle.Render(fmt.Sprintf("Cluster: %s", config.Name)))
		fmt.Printf("%s %s\n\n",
			styles.CommandStyle.Render("⚠️"),
			styles.DescriptionStyle.Render("WARNING: This action cannot be undone!"))

		fmt.Printf("%s %s ",
			styles.CommandStyle.Render("❓"),
			styles.HelpStyle.Render("Press ENTER to confirm and proceed with reset, or Ctrl+C to cancel:"))

		// Wait for user to press Enter
		fmt.Scanln()

		fmt.Printf("\n%s %s\n",
			styles.CommandStyle.Render("🔥"),
			styles.HelpStyle.Render("Destroying existing cluster infrastructure..."))

		if err := automation.RunTerraformDestroy(config); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s %s\n",
				styles.CommandStyle.Render("❌"),
				styles.CommandStyle.Render(fmt.Sprintf("Terraform destroy failed: %v", err)))
			fmt.Fprintf(os.Stderr, "%s %s\n",
				styles.CommandStyle.Render("💡"),
				styles.DescriptionStyle.Render("Continuing with cluster creation anyway..."))
		} else {
			fmt.Printf("\n%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render("Existing infrastructure destroyed successfully!"))
		}
	}

	// Build Terraform files
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔨"),
		styles.HelpStyle.Render("Building Terraform files..."))
	if err := automation.BuildTerraformFilesForConfig(config); err != nil {
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
	if err := automation.CheckTerraformInstalled(); err != nil {
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
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform init with local backend configuration"))

	if err := automation.RunTerraformInit(config); err != nil {
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

	if err := automation.RunTerraformValidate(config); err != nil {
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

	if err := automation.RunTerraformApply(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Terraform apply failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cluster infrastructure provisioned successfully!"))

	// Bootstrap cluster with essential components (Cilium, cert-manager)
	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🔧"),
		styles.HelpStyle.Render("Bootstrapping cluster with essential components..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📦"),
		styles.DescriptionStyle.Render("Installing Cilium CNI and cert-manager"))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("⚠️"),
		styles.DescriptionStyle.Render("This may take a few minutes to complete..."))

	// Run bootstrap Terraform init
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("🚀"),
		styles.HelpStyle.Render("Initializing bootstrap Terraform..."))
	if err := automation.RunBootstrapTerraformInit(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Bootstrap Terraform init failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Bootstrap Terraform initialization completed!"))

	// Run bootstrap Terraform validate
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🔍"),
		styles.HelpStyle.Render("Validating bootstrap Terraform configuration..."))
	if err := automation.RunBootstrapTerraformValidate(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Bootstrap Terraform validate failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Bootstrap Terraform configuration is valid!"))

	// Run bootstrap Terraform apply
	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("🚀"),
		styles.HelpStyle.Render("Applying bootstrap configuration..."))
	fmt.Printf("%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Running: terraform apply -auto-approve"))
	fmt.Printf("%s %s\n\n",
		styles.CommandStyle.Render("🕰️"),
		styles.DescriptionStyle.Render("Installing Cilium and cert-manager..."))

	if err := automation.RunBootstrapTerraformApply(config); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			styles.CommandStyle.Render("❌"),
			styles.CommandStyle.Render(fmt.Sprintf("Bootstrap Terraform apply failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("✅"),
		styles.TitleStyle.Render("Cluster bootstrap completed successfully!"))

	fmt.Printf("\n%s %s\n",
		styles.TitleStyle.Render("🎉"),
		styles.TitleStyle.Render("Your Kubernetes cluster is now fully operational!"))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📦"),
		styles.DescriptionStyle.Render("Installed components:"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("• Cilium CNI - Container networking"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("• cert-manager - Certificate management"))
	fmt.Printf("   %s\n", styles.DescriptionStyle.Render("• kibaship-system namespace - Ready for PaaS services"))

	fmt.Printf("\n%s %s\n",
		styles.CommandStyle.Render("📝"),
		styles.DescriptionStyle.Render("Next steps: Configure kubectl and deploy your applications"))
}
