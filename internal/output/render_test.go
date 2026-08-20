package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DonovanMods/secalc/internal/calc"
)

func TestRenderFullReport(t *testing.T) {
	p := &calc.Plan{
		Game:        "SE2",
		GravityMult: 1,
		Margin:      1.5,
		Items: []calc.MassItem{
			{Label: "1t", MassKg: 1000},
			{Label: "2 x Small Box", MassKg: 200},
		},
		TotalKg: 1200,
		Families: []calc.FamilyResult{
			{Name: "Lift", PowerUnit: "MW", Sizes: []calc.SizeResult{
				{Name: "A", Count: 1, Power: 0.5, Viable: true},
				{Name: "BB", Viable: false},
			}},
		},
	}

	var buf bytes.Buffer
	Render(&buf, p)

	// Label column is 13 runes wide ("2 x Small Box"); mass column is
	// right-aligned in 9. Spelled out via Repeat so the padding is
	// explicit: 11 pad + 2 gap + 3 right-align = 16 spaces after "1t".
	want := strings.Join([]string{
		"secalc — SE2",
		strings.Repeat("─", 42),
		"Gravity: 1 g   Target TWR: 1.5",
		"",
		"Mass breakdown:",
		"  1t" + strings.Repeat(" ", 16) + "1.00 t",
		"  2 x Small Box" + strings.Repeat(" ", 5) + "200 kg",
		strings.Repeat("─", 42),
		"Total ship mass: 1.20 t",
		"",
		"Thrusters needed to overcome gravity",
		"(each line is a complete, independent solution):",
		"",
		"  Lift",
		// size column 2 runes wide ("BB"): "A " + 2 gap + %4d.
		"    A " + "  " + "   1" + "   " + "0.5 MW",
		"    BB  not viable (cannot lift own weight at this gravity/margin)",
		"",
	}, "\n")

	if buf.String() != want {
		t.Errorf("Render mismatch.\n got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestRenderZeroGravity(t *testing.T) {
	p := &calc.Plan{
		Game:        "SE2",
		GravityMult: 0,
		Margin:      1.5,
		Items:       []calc.MassItem{{Label: "500", MassKg: 500}},
		TotalKg:     500,
	}

	var buf bytes.Buffer
	Render(&buf, p)

	if !strings.Contains(buf.String(), "Zero gravity — no thrust needed to hover.") {
		t.Errorf("missing zero-gravity message in:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "Thrusters needed") {
		t.Errorf("thruster section must be omitted in zero gravity:\n%s", buf.String())
	}
}

func TestRenderFullSuffix(t *testing.T) {
	p := &calc.Plan{
		Game:        "SE2",
		GravityMult: 1,
		Margin:      1.5,
		Full:        true,
		Items:       []calc.MassItem{{Label: "Small Box (full)", MassKg: 1000}},
		TotalKg:     1000,
	}

	var buf bytes.Buffer
	Render(&buf, p)

	if !strings.Contains(buf.String(), "Total ship mass: 1.00 t (full)") {
		t.Errorf("missing (full) suffix on total in:\n%s", buf.String())
	}
}

func TestRenderPowerFloatNoise(t *testing.T) {
	p := &calc.Plan{
		Game:        "SE2",
		GravityMult: 1,
		Margin:      1.5,
		Items:       []calc.MassItem{{Label: "1t", MassKg: 1000}},
		TotalKg:     1000,
		Families: []calc.FamilyResult{
			{Name: "Lift", PowerUnit: "MW", Sizes: []calc.SizeResult{
				{Name: "A", Count: 3, Power: 3 * 0.65, Viable: true}, // 1.9500000000000002
			}},
		},
	}

	var buf bytes.Buffer
	Render(&buf, p)

	if !strings.Contains(buf.String(), "1.95 MW") {
		t.Errorf("power must render as 1.95 MW, got:\n%s", buf.String())
	}
}

func TestRenderFractionalKilograms(t *testing.T) {
	p := &calc.Plan{
		Game:        "SE2",
		GravityMult: 0,
		Margin:      1.5,
		Items:       []calc.MassItem{{Label: "1.34", MassKg: 1.34}},
		TotalKg:     1.34,
	}

	var buf bytes.Buffer
	Render(&buf, p)

	if !strings.Contains(buf.String(), "1.34 kg") {
		t.Errorf("fractional kilograms must not be truncated, got:\n%s", buf.String())
	}
}
