package clusters

import (
	"github.com/spf13/cobra"

	"github.com/kibamail/kibaship-operator/cmd/cli/commands/clusters/create"
	"github.com/kibamail/kibaship-operator/cmd/cli/commands/clusters/credentials"
	"github.com/kibamail/kibaship-operator/cmd/cli/commands/clusters/delete"
)

// NewCommand creates and returns the clusters command with all subcommands
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Manage kibaship clusters",
		Long:  "Manage kibaship clusters across different cloud providers including AWS, Digital Ocean, Hetzner, Hetzner Robot, Linode, and Google Cloud.",
		Run: func(cmd *cobra.Command, args []string) {
			PrintHelp()
		},
	}

	// Add subcommands
	cmd.AddCommand(create.NewCommand())
	cmd.AddCommand(credentials.NewCommand())
	cmd.AddCommand(delete.NewCommand())

	// Override help command behavior
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		PrintHelp()
	})

	return cmd
}
