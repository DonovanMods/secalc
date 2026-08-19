// se2calc calculates Space Engineers 2 thruster requirements from a ship
// mass expression.
package main

import (
	"os"

	"github.com/DonovanMods/se2calc/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
