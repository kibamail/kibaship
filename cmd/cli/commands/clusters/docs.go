package clusters

import (
	"fmt"

	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// PrintHelp displays the help documentation for the clusters command
func PrintHelp() {
	styles.PrintBanner()
	fmt.Println(styles.TitleStyle.Render("🚀 Kibaship Clusters"))
	fmt.Println()
	fmt.Println(styles.DescriptionStyle.Render("Create and manage Kubernetes clusters across different cloud providers including AWS, Digital Ocean, Hetzner, Hetzner Robot, Linode, and Google Cloud."))
	fmt.Println()
	fmt.Println(styles.HelpStyle.Render("Available Commands:"))

	commands := []struct {
		name        string
		description string
	}{
		{"create", "Create a new kibaship cluster"},
		{"credentials", "Extract cluster credentials (kubeconfig, etc.)"},
		{"delete", "Delete an existing kibaship cluster"},
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
		styles.CommandStyle.Render("kibaship clusters [command] --help"),
		styles.HelpStyle.Render("for more information about a command."))
}
