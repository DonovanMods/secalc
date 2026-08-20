package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/DonovanMods/secalc/internal/config"
)

// newInitCmd builds `secalc init`, which writes every game's default
// config to the user config directory for editing.
func newInitCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "init",
		Short: "Write the default configs (one per game) to the user config dir for editing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			for _, game := range config.Games() {
				path, err := config.UserConfigPath(game)
				if err != nil {
					return err
				}
				if !force {
					if _, err := os.Stat(path); err == nil {
						fmt.Fprintf(cmd.OutOrStdout(), "Skipped %s (exists; use --force to overwrite)\n", path)
						continue
					}
				}
				defaults, err := config.DefaultTOML(game)
				if err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					return fmt.Errorf("creating config dir: %w", err)
				}
				if err := os.WriteFile(path, defaults, 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", path, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", path)
			}
			return nil
		},
	}
	c.Flags().Bool("force", false, "overwrite existing config files")
	return c
}
