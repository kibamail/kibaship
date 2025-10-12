package create

import (
	"fmt"

	"github.com/kibamail/kibaship-operator/cmd/cli/internal/styles"
)

// PrintHelp displays the comprehensive help documentation for clusters create command
func PrintHelp() {
	styles.PrintBanner()
	fmt.Println(styles.TitleStyle.Render("🚀 Kibaship clusters create"))
	fmt.Println()
	fmt.Println(styles.DescriptionStyle.Render("Create a new kibaship cluster on your chosen cloud provider."))
	fmt.Println()
	fmt.Println(styles.DescriptionStyle.Render("All cluster configuration must be provided via a YAML configuration file."))
	fmt.Println()

	// Supported Providers section
	fmt.Println(styles.HelpStyle.Render("Supported Providers:"))
	providers := []struct {
		name        string
		description string
	}{
		{"aws", "Amazon Web Services"},
		{"digital-ocean", "DigitalOcean"},
		{"hetzner", "Hetzner Cloud"},
		{"hetzner-robot", "Hetzner Robot (Dedicated Servers)"},
		{"linode", "Linode"},
		{"gcloud", "Google Cloud Platform"},
		{"kind", "Kind (Kubernetes in Docker) - Local Development"},
	}

	for _, provider := range providers {
		fmt.Printf("  %s  %s\n",
			styles.CommandStyle.Render(provider.name),
			styles.DescriptionStyle.Render(provider.description))
	}
	fmt.Println()

	// Usage section
	fmt.Println(styles.HelpStyle.Render("Usage:"))
	fmt.Printf("  %s\n", styles.CommandStyle.Render("kibaship clusters create --configuration <config-file.yaml>"))
	fmt.Println()

	// Flags section
	fmt.Println(styles.HelpStyle.Render("Required Flags:"))
	fmt.Printf("  %s  %s\n",
		styles.CommandStyle.Render("-c, --configuration"),
		styles.DescriptionStyle.Render("Path to YAML configuration file"))
	fmt.Println()

	// YAML Configuration section
	fmt.Println(styles.HelpStyle.Render("YAML Configuration:"))
	fmt.Printf("  %s\n", styles.DescriptionStyle.Render("All cluster settings must be defined in a YAML configuration file."))
	fmt.Printf("  %s\n", styles.DescriptionStyle.Render("See cmd/cli/examples/digitalocean-cluster.yaml for a complete example."))
	fmt.Println()

	// Example section
	fmt.Println(styles.HelpStyle.Render("Example:"))
	fmt.Printf("  %s\n", styles.CommandStyle.Render("kibaship clusters create --configuration my-cluster.yaml"))
	fmt.Println()

	fmt.Println(styles.HelpStyle.Render("Sample YAML Configuration:"))
	fmt.Printf("  %s\n", styles.DescriptionStyle.Render("cluster:"))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("domain: \"app.kibaship.com\""))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("email: \"admin@kibaship.com\""))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("paas-features: \"mysql,valkey,postgres\""))
	fmt.Printf("  %s\n", styles.DescriptionStyle.Render("provider:"))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("name: \"digital-ocean\""))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("digital-ocean:"))
	fmt.Printf("      %s\n", styles.DescriptionStyle.Render("token: \"your-do-token\""))
	fmt.Printf("      %s\n", styles.DescriptionStyle.Render("region: \"nyc3\""))
	fmt.Printf("      %s\n", styles.DescriptionStyle.Render("nodes: \"3\""))
	fmt.Printf("      %s\n", styles.DescriptionStyle.Render("nodes-size: \"s-4vcpu-8gb\""))
	fmt.Printf("  %s\n", styles.DescriptionStyle.Render("terraform-state:"))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("s3-bucket: \"my-terraform-state\""))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("s3-region: \"us-east-1\""))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("s3-access-key: \"AKIAIOSFODNN7EXAMPLE\""))
	fmt.Printf("    %s\n", styles.DescriptionStyle.Render("s3-access-secret: \"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\""))
}
