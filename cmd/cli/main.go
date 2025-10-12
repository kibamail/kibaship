package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kibamail/kibaship-operator/cmd/cli/commands/clusters"
	"github.com/kibamail/kibaship-operator/cmd/cli/internal/styles"
)

var (
	// version is set via ldflags at build time
	version = "dev"
)

func printHelp() {
	styles.PrintBanner()
	fmt.Println(styles.TitleStyle.Render("🚀 Kibaship CLI"))
	fmt.Println()
	fmt.Println()
	fmt.Println(styles.HelpStyle.Render("Available Commands:"))

	commands := []struct {
		name        string
		description string
	}{
		{"clusters", "Manage Kubernetes clusters"},
		{"version", "Show version information"},
	}

	for _, cmd := range commands {
		fmt.Printf("  %s  %s\n",
			styles.CommandStyle.Render(cmd.name),
			styles.DescriptionStyle.Render(cmd.description))
	}

	fmt.Println()
	fmt.Println(styles.HelpStyle.Render("Flags:"))
	fmt.Printf("  %s  %s\n",
		styles.CommandStyle.Render("-h, --help"),
		styles.DescriptionStyle.Render("Show help for any command"))
	fmt.Println()
	fmt.Printf("%s %s %s\n",
		styles.HelpStyle.Render("Use"),
		styles.CommandStyle.Render("kibaship [command] --help"),
		styles.HelpStyle.Render("for more information about a command."))
}

var rootCmd = &cobra.Command{
	Use:   "kibaship",
	Short: "The complete paas platform for kubernetes.",
	Long:  `Kibaship is a complete paas platform for deploying and managing applications, databases and workloads, powered by Kubernetes.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show help and exit
		printHelp()
	},
}

func init() {
	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			styles.PrintBanner()
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("Kibaship CLI"),
				styles.CommandStyle.Render("v"+version))
		},
	}

	// Override root command help to use our styled help
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		printHelp()
	})

	// Add commands to root
	rootCmd.AddCommand(clusters.NewCommand())
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
