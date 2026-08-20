package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DonovanMods/secalc/internal/config"
)

func TestShortcutsListing(t *testing.T) {
	cfg := &config.Config{
		Containers: map[string]config.Container{
			"box":   {Name: "Kg Box", Mass: 100, Capacity: 900},
			"vat":   {Name: "Liter Vat", Mass: 2500, CapacityL: 125},
			"shelf": {Name: "Empty Shelf", Mass: 50},
		},
		Gravity: map[string]float64{"moon": 0.25, "earthlike": 1.0},
	}

	var buf bytes.Buffer
	Shortcuts(&buf, "SE1", cfg)

	// Key column 5 runes ("shelf"), name column 11 ("Empty Shelf");
	// capacity renders kg/t for mass-limited, liters for volumetric,
	// "holds no cargo" when neither is set.
	want := strings.Join([]string{
		"secalc — SE1 shortcuts",
		"",
		"Storage (use in expressions, e.g. 2*box):",
		"  box    Kg Box       100 kg empty, 900 kg capacity",
		"  shelf  Empty Shelf  50 kg empty, holds no cargo",
		"  vat    Liter Vat    2.50 t empty, 125 L capacity",
		"",
		"Gravity presets (use with -g):",
		"  earthlike  1 g",
		"  moon       0.25 g",
		"",
	}, "\n")

	if buf.String() != want {
		t.Errorf("Shortcuts mismatch.\n got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestShortcutsOmitsEmptySections(t *testing.T) {
	cfg := &config.Config{
		Containers: map[string]config.Container{
			"box": {Name: "Kg Box", Mass: 100, Capacity: 900},
		},
	}

	var buf bytes.Buffer
	Shortcuts(&buf, "SE2", cfg)

	if strings.Contains(buf.String(), "Gravity presets") {
		t.Errorf("empty gravity section must be omitted:\n%s", buf.String())
	}
}
