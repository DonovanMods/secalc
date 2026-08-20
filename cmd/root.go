// Package cmd wires the secalc command-line interface.
package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DonovanMods/secalc/internal/calc"
	"github.com/DonovanMods/secalc/internal/config"
	"github.com/DonovanMods/secalc/internal/output"
	"github.com/DonovanMods/secalc/internal/parse"
)

// Version is the semantic version of secalc.
const Version = "0.4.0"

// NewRootCmd builds a fresh root command. A new instance is built per call
// so tests never share flag state.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "secalc [flags] <expression>...",
		Short: "Space Engineers thruster and mass calculator (SE2 and SE1)",
		Long: `secalc reports how many thrusters of each type lift a Space Engineers
ship — SE2 by default, SE1 with --game se1 — and what they consume,
from a mass expression.

An expression is terms joined by "+": mass literals (1.23t, 1,230kg,
bare kg) and storage shortcuts from the selected game's config (2*s15m
for SE2, 2*lgl for SE1), matched case-insensitively. 

Run "secalc init" to write both games' default configs for editing, and
"secalc shortcuts" to list the storage codes and gravity presets your
config defines.`,
		Example: `  secalc -g 0.5 1.23t + 2*s15m + s25m
  secalc --full 1t + 2*s15m
  secalc --game se1 -g moon --full 1t + 2*lgl`,
		Version: Version,
		// ArbitraryArgs, not unset: with a subcommand present, cobra's
		// default arg validator would reject "secalc 1t" as an unknown
		// command.
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			flags := cmd.Flags()
			game, cfg, err := loadGameConfig(cmd)
			if err != nil {
				return err
			}

			terms, err := parse.Expression(args, cfg.ShortcutKeys())
			if err != nil {
				return err
			}

			gravityRaw, _ := flags.GetString("gravity")
			gravity, err := resolveGravity(gravityRaw, cfg.Gravity)
			if err != nil {
				return err
			}
			full, _ := flags.GetBool("full")
			margin := cfg.Settings.Margin
			if flags.Changed("margin") {
				margin, _ = flags.GetFloat64("margin")
			}

			plan, err := calc.Build(calc.Input{
				Game:         strings.ToUpper(game),
				Terms:        terms,
				Cfg:          cfg,
				GravityMult:  gravity,
				Margin:       margin,
				Full:         full,
				CargoDensity: cfg.Settings.CargoDensity,
			})
			if err != nil {
				return err
			}

			output.Render(cmd.OutOrStdout(), plan)
			return nil
		},
	}
	root.Flags().String("game", "se2", "which game's config stack to use: se2 | se1")
	root.Flags().StringP("gravity", "g", "1", "gravity multiplier relative to Earth (1 g), or a named preset from config")
	root.Flags().BoolP("full", "f", false, "use loaded container masses")
	root.Flags().Float64P("margin", "m", 0, "target thrust-to-weight ratio (default: from config)")
	root.Flags().String("config", "", "alternate config file")
	root.AddCommand(newInitCmd())
	root.AddCommand(newShortcutsCmd())
	return root
}

// loadGameConfig resolves a command's shared --game/--config flags into
// the loaded config stack for that game.
func loadGameConfig(cmd *cobra.Command) (string, *config.Config, error) {
	game, _ := cmd.Flags().GetString("game")
	path, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(game, path)
	return game, cfg, err
}

// resolveGravity turns the -g value into a multiplier: a number is used
// as-is; anything else is a case-insensitive lookup in the config's
// [gravity] presets.
func resolveGravity(raw string, presets map[string]float64) (float64, error) {
	if v, err := strconv.ParseFloat(raw, 64); err == nil {
		return v, nil
	}
	if v, ok := presets[strings.ToLower(raw)]; ok {
		return v, nil
	}
	names := make([]string, 0, len(presets))
	for name := range presets {
		names = append(names, name)
	}
	sort.Strings(names)
	return 0, fmt.Errorf("unknown gravity %q (use a number or one of: %s)", raw, strings.Join(names, ", "))
}

// Execute runs secalc and returns any execution error (cobra has already
// printed it).
func Execute() error {
	return NewRootCmd().Execute()
}
