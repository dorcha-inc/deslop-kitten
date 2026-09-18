// Package commands holds the cobra command tree for the deslop-kitten binary.
package commands

import "github.com/spf13/cobra"

// NewRoot returns the root command with every subcommand attached.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "deslop-kitten",
		Short:         "Check a pull request against the repository's contribution policy",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(newScore())
	root.AddCommand(newPresets())
	return root
}
