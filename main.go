// secalc calculates Space Engineers thruster requirements from a ship
// mass expression for SE2 and SE1.
package main

import (
	"os"

	"github.com/DonovanMods/secalc/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
