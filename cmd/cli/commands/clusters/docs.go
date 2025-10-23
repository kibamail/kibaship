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
	fmt.Println(styles.DescriptionStyle.Render("Cluster management commands have been removed from this CLI."))
	fmt.Println()
	fmt.Println(styles.HelpStyle.Render("Available Commands:"))
	fmt.Println(styles.DescriptionStyle.Render("  No cluster commands are currently available."))
	fmt.Println()
	fmt.Println(styles.HelpStyle.Render("Flags:"))
	fmt.Printf("  %s  %s\n",
		styles.CommandStyle.Render("-h, --help"),
		styles.DescriptionStyle.Render("Show help for any command"))
}
