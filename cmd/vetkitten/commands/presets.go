package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jadidbourbaki/vetkitten/internal/rules"
)

func newPresets() *cobra.Command {
	return &cobra.Command{
		Use:   "presets",
		Short: "List the embedded policy presets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			names, err := rules.PresetNames()
			if err != nil {
				return err
			}
			for _, n := range names {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), n); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
