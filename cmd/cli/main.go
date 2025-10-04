package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kibamail/kibaship-operator/cmd/cli/clusters"
	"github.com/spf13/cobra"
)

var (
	// version is set via ldflags at build time
	version = "dev"

	// Color palette
	primaryColor = lipgloss.Color("#00D4AA")
	accentColor  = lipgloss.Color("#F59E0B")
	textColor    = lipgloss.Color("#E5E7EB")
	mutedColor   = lipgloss.Color("#9CA3AF")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	bannerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Align(lipgloss.Left).
			Padding(1, 0)

	helpStyle = lipgloss.NewStyle().
			Foreground(textColor)

	commandStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	descriptionStyle = lipgloss.NewStyle().
				Foreground(mutedColor)
)

// ASCII art banner for Kibaship
const banner = `
██╗  ██╗██╗██████╗  █████╗ ███████╗██╗  ██╗██╗██████╗
██║ ██╔╝██║██╔══██╗██╔══██╗██╔════╝██║  ██║██║██╔══██╗
█████╔╝ ██║██████╔╝███████║███████╗███████║██║██████╔╝
██╔═██╗ ██║██╔══██╗██╔══██║╚════██║██╔══██║██║██╔═══╝
██║  ██╗██║██████╔╝██║  ██║███████║██║  ██║██║██║
╚═╝  ╚═╝╚═╝╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝╚═╝
`

const subtitle = "⚡ A simple and intuitive way to manage your kibaship clusters ⚡"

func printBanner() {
	fmt.Print(bannerStyle.Render(banner))
	fmt.Print(bannerStyle.Render(subtitle))
	fmt.Println()
}

func printHelp() {
	printBanner()
	fmt.Println(titleStyle.Render("🚀 Kibaship Operator CLI"))
	fmt.Println()
	fmt.Println(helpStyle.Render("A beautiful CLI tool for managing Kibaship operator clusters"))
	fmt.Println()
	fmt.Println(helpStyle.Render("Available Commands:"))

	commands := []struct {
		name        string
		description string
	}{
		{"clusters", "Manage Kubernetes clusters (create, list, destroy)"},
		{"projects", "Manage Kibaship projects (create, list, destroy)"},
		{"applications", "Manage applications (create, list, destroy)"},
		{"version", "Show version information"},
	}

	for _, cmd := range commands {
		fmt.Printf("  %s  %s\n",
			commandStyle.Render(cmd.name),
			descriptionStyle.Render(cmd.description))
	}

	fmt.Println()
	fmt.Println(helpStyle.Render("Flags:"))
	fmt.Printf("  %s  %s\n",
		commandStyle.Render("-h, --help"),
		descriptionStyle.Render("Show help for any command"))
	fmt.Printf("  %s  %s\n",
		commandStyle.Render("--config"),
		descriptionStyle.Render("Specify config file path"))
	fmt.Println()
	fmt.Printf("%s %s %s\n",
		helpStyle.Render("Use"),
		commandStyle.Render("kibaship [command] --help"),
		helpStyle.Render("for more information about a command."))
}

// Custom help template with banner
func customHelpTemplate() string {
	return bannerStyle.Render(banner) + bannerStyle.Render(subtitle) + "\n" +
		titleStyle.Render("🚀 Kibaship Operator CLI") + "\n\n" +
		helpStyle.Render("{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}{{end}}") + "\n\n" +
		"{{if or .Runnable .HasSubCommands}}" +
		helpStyle.Render("Usage:") + "\n" +
		"  " + commandStyle.Render("{{.UseLine}}") + "\n" +
		"{{end}}" +
		"{{if .HasAvailableSubCommands}}\n" +
		helpStyle.Render("Available Commands:") + "\n" +
		"{{range .Commands}}" +
		"{{if (or .IsAvailableCommand (eq .Name \"help\"))}}" +
		"  " + commandStyle.Render("{{rpad .Name .NamePadding}}") + " " + descriptionStyle.Render("{{.Short}}") + "\n" +
		"{{end}}" +
		"{{end}}" +
		"{{end}}" +
		"{{if .HasAvailableLocalFlags}}\n" +
		helpStyle.Render("Flags:") + "\n" +
		"{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}" +
		"{{end}}" +
		"{{if .HasAvailableInheritedFlags}}\n" +
		helpStyle.Render("Global Flags:") + "\n" +
		"{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}" +
		"{{end}}" +
		"{{if .HasHelpSubCommands}}\n" +
		helpStyle.Render("Additional help topics:") + "\n" +
		"{{range .Commands}}" +
		"{{if .IsAdditionalHelpTopicCommand}}" +
		"  " + commandStyle.Render("{{rpad .CommandPath .CommandPathPadding}}") + " " +
		descriptionStyle.Render("{{.Short}}") + "\n" +
		"{{end}}" +
		"{{end}}" +
		"{{end}}" +
		"{{if .HasAvailableSubCommands}}\n" +
		helpStyle.Render("Use \"{{.CommandPath}} [command] --help\" for more information about a command.") + "\n" +
		"{{end}}"
}

// Styled output functions
func printSuccess(message string) {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	fmt.Printf("✅ %s\n", successStyle.Render(message))
}

func printError(message string) {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	fmt.Printf("❌ %s\n", errorStyle.Render(message))
}

func printInfo(message string) {
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true)
	fmt.Printf("ℹ️  %s\n", infoStyle.Render(message))
}

func printWarning(message string) {
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	fmt.Printf("⚠️  %s\n", warningStyle.Render(message))
}

func printStep(step int, message string) {
	stepStyle := lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	fmt.Printf("%s %s\n", stepStyle.Render(fmt.Sprintf("[%d]", step)), helpStyle.Render(message))
}

func printProgress(message string) {
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")).Bold(true)
	fmt.Printf("🔄 %s\n", progressStyle.Render(message))
}

func printTable(headers []string, rows [][]string) {
	headerStyle := lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	cellStyle := lipgloss.NewStyle().Foreground(textColor)

	// Print headers
	for i, header := range headers {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%-20s", headerStyle.Render(header))
	}
	fmt.Println()

	// Print separator
	for i := range headers {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Print(strings.Repeat("─", 20))
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Print("  ")
			}
			fmt.Printf("%-20s", cellStyle.Render(cell))
		}
		fmt.Println()
	}
}

var rootCmd = &cobra.Command{
	Use:   "kibaship",
	Short: "A beautiful CLI tool for managing Kibaship operator clusters",
	Long: `Kibaship CLI provides a clean and intuitive interface for managing 
Kubernetes clusters running the Kibaship operator. Get complete cluster 
overviews, manage deployments, and monitor your infrastructure with style.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show help and exit
		printHelp()
	},
}

func init() {
	// Set custom help template for all commands
	cobra.AddTemplateFunc("trimTrailingWhitespaces", func(s string) string {
		return strings.TrimRightFunc(s, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n'
		})
	})

	rootCmd.SetHelpTemplate(customHelpTemplate())

	// Clusters command group
	clustersCmd := &cobra.Command{
		Use:   "clusters",
		Short: "Manage Kubernetes clusters",
		Long:  "Create, list, and destroy Kubernetes clusters for Kibaship operator",
	}
	clustersCmd.SetHelpTemplate(customHelpTemplate())

	createClusterCmd := &cobra.Command{
		Use:   "create [cluster-name]",
		Short: "Create a new Kind cluster",
		Long: `Create a new Kind cluster with Kibaship operator infrastructure (Complete)

Required for full installation:
  --operator-domain: Base domain for applications (e.g., myapps.kibaship.com)
  --operator-webhook-url: Webhook URL for notifications (e.g., https://webhook.example.com/kibaship)

Examples:
  # Create with full infrastructure
  kibaship clusters create my-cluster \
    --operator-domain myapps.kibaship.com \
    --operator-webhook-url https://webhook.example.com/kibaship

  # Create with custom configuration
  kibaship clusters create my-cluster -c 3 -w 2 \
    --operator-domain myapps.kibaship.com \
    --operator-webhook-url https://webhook.example.com/kibaship

  # Basic cluster only (no infrastructure)
  kibaship clusters create my-cluster --skip-infrastructure`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Determine cluster name
			clusterName := "kibaship-cluster"
			if len(args) > 0 {
				clusterName = args[0]
			}

			// Get flag values
			controlPlaneNodes, _ := cmd.Flags().GetInt("control-plane-nodes")
			workerNodes, _ := cmd.Flags().GetInt("worker-nodes")
			ciliumVersion, _ := cmd.Flags().GetString("cilium-version")
			skipInfrastructure, _ := cmd.Flags().GetBool("skip-infrastructure")
			operatorDomain, _ := cmd.Flags().GetString("operator-domain")
			operatorACMEEmail, _ := cmd.Flags().GetString("operator-acme-email")
			operatorWebhookURL, _ := cmd.Flags().GetString("operator-webhook-url")

			// Create options
			opts := clusters.CreateOptions{
				Name:               clusterName,
				ControlPlaneNodes:  controlPlaneNodes,
				WorkerNodes:        workerNodes,
				CiliumVersion:      ciliumVersion,
				SkipInfrastructure: skipInfrastructure,
				OperatorDomain:     operatorDomain,
				OperatorACMEEmail:  operatorACMEEmail,
				OperatorWebhookURL: operatorWebhookURL,
				Version:            version,
			}

			// Additional validation for operator configuration when not skipping infrastructure
			if !skipInfrastructure {
				if operatorDomain == "" {
					printError("--operator-domain is required when installing full infrastructure")
					printInfo("Example: --operator-domain myapps.kibaship.com")
					return
				}
				if operatorWebhookURL == "" {
					printError("--operator-webhook-url is required when installing full infrastructure")
					printInfo("Example: --operator-webhook-url https://webhook.example.com/kibaship")
					return
				}
			}

			// Validate options
			if err := clusters.ValidateCreateOptions(opts); err != nil {
				printError(fmt.Sprintf("Invalid options: %v", err))
				return
			}

			// Display configuration
			printBanner()
			fmt.Println(titleStyle.Render("🚀 Creating Kibaship Cluster"))
			fmt.Println()
			printInfo(fmt.Sprintf("Cluster name: %s", opts.Name))
			printInfo(fmt.Sprintf("Control plane nodes: %d", opts.ControlPlaneNodes))
			printInfo(fmt.Sprintf("Worker nodes: %d", opts.WorkerNodes))
			printInfo(fmt.Sprintf("Cilium version: %s", opts.CiliumVersion))
			if opts.SkipInfrastructure {
				printInfo("Infrastructure installation: SKIPPED")
			}
			fmt.Println()

			// Create cluster with error handling
			err := clusters.CreateCluster(opts, printStep, printProgress, printSuccess, printError, printInfo)
			if err != nil {
				printError("Cluster creation failed!")
				clusters.CleanupOnFailure(opts.Name, printInfo)
				os.Exit(1)
			}
		},
	}
	createClusterCmd.SetHelpTemplate(customHelpTemplate())

	// Add flags to create command
	createClusterCmd.Flags().IntP("control-plane-nodes", "c", 1, "Number of control-plane nodes (max: 3)")
	createClusterCmd.Flags().IntP("worker-nodes", "w", 0, "Number of worker nodes (max: 10)")
	createClusterCmd.Flags().String("cilium-version", "1.18.0", "Cilium version to install")
	createClusterCmd.Flags().Bool("skip-infrastructure", false, "Skip infrastructure installation")

	// Operator configuration flags (required for full installation)
	createClusterCmd.Flags().String("operator-domain", "", "Base domain for applications (required, e.g., myapps.kibaship.com)")
	createClusterCmd.Flags().String("operator-acme-email", "", "Email for ACME certificate registration (optional)")
	createClusterCmd.Flags().String("operator-webhook-url", "", "Webhook URL for notifications (required, e.g., https://webhook.example.com/kibaship)")

	// Note: operator flags are validated conditionally in the command logic

	listClustersCmd := &cobra.Command{
		Use:   "list",
		Short: "List all Kibaship clusters",
		Long:  "List all Kibaship-managed Kind clusters with their status and information",
		Run: func(cmd *cobra.Command, args []string) {
			printBanner()
			fmt.Println(titleStyle.Render("📋 Kibaship Clusters"))
			fmt.Println()

			err := clusters.ListClusters(printInfo, printTable, printSuccess, printWarning)
			if err != nil {
				printError(fmt.Sprintf("Failed to list clusters: %v", err))
				os.Exit(1)
			}
		},
	}
	listClustersCmd.SetHelpTemplate(customHelpTemplate())

	destroyClusterCmd := &cobra.Command{
		Use:   "destroy [cluster-name]",
		Short: "Destroy a Kibaship cluster",
		Long: `Destroy a Kibaship-managed Kind cluster and clean up all resources

This command will permanently delete:
  • The Kind cluster and all nodes
  • All deployed applications and services
  • All persistent volumes and data
  • Cluster configuration files
  • kubectl context

Examples:
  kibaship clusters destroy my-cluster          # Destroy specific cluster (with confirmation)
  kibaship clusters destroy my-cluster --force  # Destroy without confirmation
  kibaship clusters destroy --all              # Destroy all Kibaship clusters
  kibaship clusters destroy --all --force      # Destroy all without confirmation`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get flags
			force, _ := cmd.Flags().GetBool("force")
			destroyAll, _ := cmd.Flags().GetBool("all")

			printBanner()
			fmt.Println(titleStyle.Render("💥 Destroying Kibaship Cluster"))
			fmt.Println()

			if destroyAll {
				// Destroy all clusters
				err := clusters.DestroyAllClusters(
					force, printStep, printProgress, printSuccess, printError, printInfo, printWarning,
				)
				if err != nil {
					printError("Failed to destroy all clusters!")
					os.Exit(1)
				}
				return
			}

			// Destroy specific cluster
			if len(args) == 0 {
				printError("Cluster name is required")
				printInfo("Use 'kibaship clusters destroy <cluster-name>' or 'kibaship clusters destroy --all'")
				os.Exit(1)
			}

			clusterName := args[0]
			opts := clusters.DestroyOptions{
				Name:  clusterName,
				Force: force,
			}

			// Validate options
			if err := clusters.ValidateDestroyOptions(opts); err != nil {
				printError(fmt.Sprintf("Invalid options: %v", err))
				os.Exit(1)
			}

			// Destroy cluster
			err := clusters.DestroyCluster(opts, printStep, printProgress, printSuccess, printError, printInfo, printWarning)
			if err != nil {
				printError("Cluster destruction failed!")
				os.Exit(1)
			}
		},
	}
	destroyClusterCmd.SetHelpTemplate(customHelpTemplate())

	// Add flags to destroy command
	destroyClusterCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	destroyClusterCmd.Flags().Bool("all", false, "Destroy all Kibaship clusters")

	clustersCmd.AddCommand(createClusterCmd)
	clustersCmd.AddCommand(listClustersCmd)
	clustersCmd.AddCommand(destroyClusterCmd)

	// Projects command group
	projectsCmd := &cobra.Command{
		Use:   "projects",
		Short: "Manage Kibaship projects",
		Long:  "Create, list, and destroy Kibaship projects",
	}
	projectsCmd.SetHelpTemplate(customHelpTemplate())

	createProjectCmd := &cobra.Command{
		Use:   "create [project-name]",
		Short: "Create a new project",
		Long:  "Create a new Kibaship project",
		Run: func(cmd *cobra.Command, args []string) {
			printWarning("Project creation coming soon!")
		},
	}
	createProjectCmd.SetHelpTemplate(customHelpTemplate())

	listProjectsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Long:  "List all Kibaship projects",
		Run: func(cmd *cobra.Command, args []string) {
			printWarning("Project listing coming soon!")
		},
	}
	listProjectsCmd.SetHelpTemplate(customHelpTemplate())

	destroyProjectCmd := &cobra.Command{
		Use:   "destroy [project-name]",
		Short: "Destroy a project",
		Long:  "Destroy a Kibaship project and clean up resources",
		Run: func(cmd *cobra.Command, args []string) {
			printWarning("Project destruction coming soon!")
		},
	}
	destroyProjectCmd.SetHelpTemplate(customHelpTemplate())

	projectsCmd.AddCommand(createProjectCmd)
	projectsCmd.AddCommand(listProjectsCmd)
	projectsCmd.AddCommand(destroyProjectCmd)

	// Applications command group
	applicationsCmd := &cobra.Command{
		Use:   "applications",
		Short: "Manage applications",
		Long:  "Create, list, and destroy applications",
	}
	applicationsCmd.SetHelpTemplate(customHelpTemplate())

	createAppCmd := &cobra.Command{
		Use:   "create [app-name]",
		Short: "Create a new application",
		Long:  "Create a new application deployment",
		Run: func(cmd *cobra.Command, args []string) {
			printWarning("Application creation coming soon!")
		},
	}
	createAppCmd.SetHelpTemplate(customHelpTemplate())

	listAppsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all applications",
		Long:  "List all deployed applications",
		Run: func(cmd *cobra.Command, args []string) {
			printWarning("Application listing coming soon!")
		},
	}
	listAppsCmd.SetHelpTemplate(customHelpTemplate())

	destroyAppCmd := &cobra.Command{
		Use:   "destroy [app-name]",
		Short: "Destroy an application",
		Long:  "Destroy an application and clean up resources",
		Run: func(cmd *cobra.Command, args []string) {
			printWarning("Application destruction coming soon!")
		},
	}
	destroyAppCmd.SetHelpTemplate(customHelpTemplate())

	applicationsCmd.AddCommand(createAppCmd)
	applicationsCmd.AddCommand(listAppsCmd)
	applicationsCmd.AddCommand(destroyAppCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			printBanner()
			fmt.Printf("%s %s\n",
				titleStyle.Render("Kibaship CLI"),
				commandStyle.Render("v"+version))
		},
	}
	versionCmd.SetHelpTemplate(customHelpTemplate())

	// Add all commands to root
	rootCmd.AddCommand(clustersCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(applicationsCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
