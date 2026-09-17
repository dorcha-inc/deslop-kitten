// Package commands holds the cobra command tree for the vetkitten binary.
package commands

import "github.com/spf13/cobra"

// NewRoot returns the root command with every subcommand attached.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "vetkitten",
		Short:         "Score a pull request for review cost and policy compliance",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(newScore())
	root.AddCommand(newPresets())
	return root
}
