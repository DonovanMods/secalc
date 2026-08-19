// Package cmd wires the se2calc command-line interface.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/DonovanMods/se2calc/internal/calc"
	"github.com/DonovanMods/se2calc/internal/config"
	"github.com/DonovanMods/se2calc/internal/output"
	"github.com/DonovanMods/se2calc/internal/parse"
)

// Version is the semantic version of se2calc.
const Version = "0.1.0"

// NewRootCmd builds a fresh root command. A new instance is built per call
// so tests never share flag state.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "se2calc [flags] <expression>...",
		Short:        "Space Engineers 2 thruster and mass calculator",
		Version:      Version,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			configPath, _ := flags.GetString("config")
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			terms, err := parse.Expression(args, cfg.ShortcutKeys())
			if err != nil {
				return err
			}

			gravity, _ := flags.GetFloat64("gravity")
			full, _ := flags.GetBool("full")
			margin := cfg.Settings.Margin
			if flags.Changed("margin") {
				margin, _ = flags.GetFloat64("margin")
			}

			plan, err := calc.Build(calc.Input{
				Terms:       terms,
				Cfg:         cfg,
				GravityMult: gravity,
				Margin:      margin,
				Full:        full,
			})
			if err != nil {
				return err
			}

			output.Render(cmd.OutOrStdout(), plan)
			return nil
		},
	}
	root.Flags().Float64P("gravity", "g", 1.0, "gravity multiplier relative to Earth (1 g)")
	root.Flags().BoolP("full", "f", false, "use loaded container masses (empty mass + capacity)")
	root.Flags().Float64P("margin", "m", 0, "target thrust-to-weight ratio (default: from config)")
	root.Flags().String("config", "", "alternate config file")
	root.AddCommand(newInitCmd())
	return root
}

// Execute runs se2calc and returns any execution error (cobra has already
// printed it).
func Execute() error {
	return NewRootCmd().Execute()
}
