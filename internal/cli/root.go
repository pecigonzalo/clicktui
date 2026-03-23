// Package cli wires together all CLI commands.
package cli

import (
	"github.com/spf13/cobra"
)

// profileFlag is the global --profile flag value shared across commands.
var profileFlag string

// New builds and returns the root cobra command.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "clicktui",
		Short: "A terminal UI and CLI for ClickUp",
		Long: `clicktui lets you browse your ClickUp workspace hierarchy and manage
tasks from the terminal.`,
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVarP(
		&profileFlag,
		"profile", "p",
		"default",
		"config profile to use",
	)

	root.AddCommand(newAuthCmd())

	return root
}
