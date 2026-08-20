package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/DonovanMods/secalc/internal/config"
)

// newInitCmd builds `secalc init`, which writes the embedded default
// config to the user config path for editing.
func newInitCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "init",
		Short: "Write the default config to the user config path for editing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			path, err := config.UserConfigPath()
			if err != nil {
				return err
			}
			if !force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("%s already exists (use --force to overwrite)", path)
				}
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("creating config dir: %w", err)
			}
			if err := os.WriteFile(path, config.DefaultTOML, 0o644); err != nil {
				return fmt.Errorf("writing config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", path)
			return nil
		},
	}
	c.Flags().Bool("force", false, "overwrite an existing config file")
	return c
}
