package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/DonovanMods/secalc/internal/output"
)

// newShortcutsCmd builds `secalc shortcuts`, which lists the storage codes
// and gravity presets the selected config stack defines.
func newShortcutsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "shortcuts",
		Short: "List the storage codes and gravity presets your config defines",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			game, cfg, err := loadGameConfig(cmd)
			if err != nil {
				return err
			}
			output.Shortcuts(cmd.OutOrStdout(), strings.ToUpper(game), cfg)
			return nil
		},
	}
	c.Flags().String("game", "se2", "which game's config stack to use: se2 | se1")
	c.Flags().String("config", "", "alternate config file")
	return c
}
