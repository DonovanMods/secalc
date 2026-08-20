package output

import (
	"fmt"
	"io"
	"sort"
	"unicode/utf8"

	"github.com/DonovanMods/secalc/internal/config"
)

// Shortcuts lists the CLI-typable codes a config stack defines: storage
// container shortcuts and gravity presets. game is the uppercase display
// id (SE1/SE2).
func Shortcuts(w io.Writer, game string, cfg *config.Config) {
	fmt.Fprintf(w, "secalc — %s shortcuts\n", game)

	keys := cfg.ShortcutKeys()
	if len(keys) > 0 {
		fmt.Fprintf(w, "\nStorage (use in expressions, e.g. 2*%s):\n", keys[0])
		keyW, nameW := 0, 0
		for _, k := range keys {
			if n := utf8.RuneCountInString(k); n > keyW {
				keyW = n
			}
			if n := utf8.RuneCountInString(cfg.Containers[k].Name); n > nameW {
				nameW = n
			}
		}
		for _, k := range keys {
			c := cfg.Containers[k]
			fmt.Fprintf(w, "  %s  %s  %s empty, %s\n",
				pad(k, keyW), pad(c.Name, nameW), mass(c.Mass), capacity(c))
		}
	}

	if len(cfg.Gravity) > 0 {
		names := make([]string, 0, len(cfg.Gravity))
		nameW := 0
		for name := range cfg.Gravity {
			names = append(names, name)
			if n := utf8.RuneCountInString(name); n > nameW {
				nameW = n
			}
		}
		sort.Strings(names)
		fmt.Fprintf(w, "\nGravity presets (use with -g):\n")
		for _, name := range names {
			fmt.Fprintf(w, "  %s  %s g\n", pad(name, nameW), num(cfg.Gravity[name]))
		}
	}
}

// capacity describes what a full container holds, in the unit its game
// uses (kg for mass-limited, liters for volumetric).
func capacity(c config.Container) string {
	switch {
	case c.Capacity > 0:
		return mass(c.Capacity) + " capacity"
	case c.CapacityL > 0:
		return num(c.CapacityL) + " L capacity"
	default:
		return "holds no cargo"
	}
}
