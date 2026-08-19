// Package cmd wires the se2calc command-line interface.
package cmd

import (
	"github.com/spf13/cobra"
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
			return nil // wired up in a later task
		},
	}
	root.Flags().Float64P("gravity", "g", 1.0, "gravity multiplier relative to Earth (1 g)")
	root.Flags().BoolP("full", "f", false, "use loaded container masses (empty mass + capacity)")
	root.Flags().Float64P("margin", "m", 0, "target thrust-to-weight ratio (default: from config)")
	root.Flags().String("config", "", "alternate config file")
	return root
}

// Execute runs se2calc and returns any execution error (cobra has already
// printed it).
func Execute() error {
	return NewRootCmd().Execute()
}
