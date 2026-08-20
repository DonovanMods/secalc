package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// testConfigTOML keeps e2e expectations hand-checkable: one thruster
// (50 kN, 1 t self-mass, 0.5 MW), two containers.
const testConfigTOML = `
[settings]
margin = 1.5

[containers.s1]
name = "Small Box"
mass = 100
capacity = 900

[containers.s2]
name = "Big Box"
mass = 300
capacity = 2700

[thrusters.lift]
name = "Lift"
power_unit = "MW"

[thrusters.lift.sizes.a]
name = "A"
thrust = 50000
mass = 1000
power = 0.5
`

// run executes the root command with args, returning stdout+stderr and
// the error. It isolates XDG_CONFIG_HOME so the developer's real user
// config never leaks in.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfgPath := filepath.Join(t.TempDir(), "test.toml")
	if err := os.WriteFile(cfgPath, []byte(testConfigTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"--config", cfgPath}, args...))
	err := root.Execute()
	return out.String(), err
}

func TestE2EEmptyContainers(t *testing.T) {
	// M = 1000 + 2*100 + 300 = 1500 kg at 0.5 g, margin 1.5:
	// need = 1500*4.905*1.5 = 11036.25 N, denom = 50000-1000*7.3575
	// = 42642.5 -> ceil(0.259) = 1 thruster, 0.5 MW.
	out, err := run(t, "-g", "0.5", "1t", "+", "2*S1", "+", "S2")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	for _, want := range []string{
		"Gravity: 0.5 g   Target TWR: 1.5",
		"2 x Small Box",
		"Total ship mass: 1.50 t",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if !regexp.MustCompile(`A\s+1\s+0\.5 MW`).MatchString(out) {
		t.Errorf("want 1 thruster at 0.5 MW:\n%s", out)
	}
}

func TestE2EFullContainers(t *testing.T) {
	// M = 1000 + 2*(100+900) + (300+2700) = 6000 kg:
	// need = 6000*7.3575 = 44145 N -> ceil(1.035) = 2 thrusters, 1 MW.
	out, err := run(t, "-g", "0.5", "--full", "1t", "+", "2*S1", "+", "S2")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	for _, want := range []string{
		"2 x Small Box (full)",
		"Total ship mass: 6.00 t (full)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if !regexp.MustCompile(`A\s+2\s+1 MW`).MatchString(out) {
		t.Errorf("want 2 thrusters at 1 MW:\n%s", out)
	}
}

func TestE2EMarginFlagOverridesConfig(t *testing.T) {
	// Margin 1.0 instead of config's 1.5: need = 1500*4.905 = 7357.5 N,
	// denom = 50000 - 1000*4.905 = 45095 -> still 1, but the header
	// must show the flag value.
	out, err := run(t, "-g", "0.5", "-m", "1.0", "1t", "+", "2*S1", "+", "S2")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	// Note the \n: a bare "Target TWR: 1" would also match "1.5".
	if !strings.Contains(out, "Target TWR: 1\n") {
		t.Errorf("want margin 1 from flag in header:\n%s", out)
	}
}

func TestE2EGravityPresetName(t *testing.T) {
	// Presets come from the embedded defaults, which the temp --config
	// merges over — palatine is defined there as 0.33.
	out, err := run(t, "-g", "palatine", "1t")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Gravity: 0.33 g") {
		t.Errorf("want palatine resolved to 0.33 g in header:\n%s", out)
	}
}

func TestE2EGravityPresetCaseInsensitive(t *testing.T) {
	out, err := run(t, "-g", "Palatine", "1t")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Gravity: 0.33 g") {
		t.Errorf("want Palatine (mixed case) resolved to 0.33 g:\n%s", out)
	}
}

func TestE2EGravityPresetSpace(t *testing.T) {
	out, err := run(t, "-g", "space", "1t")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Zero gravity — no thrust needed to hover.") {
		t.Errorf("want space preset to trigger zero-gravity message:\n%s", out)
	}
}

func TestE2EGravityPresetUnknown(t *testing.T) {
	out, err := run(t, "-g", "nonsense", "1t")
	if err == nil {
		t.Fatalf("want error for unknown gravity preset, got output:\n%s", out)
	}
	msg := err.Error()
	if !strings.Contains(msg, "nonsense") || !strings.Contains(msg, "palatine") {
		t.Errorf("error should name the bad preset and list known ones: %q", msg)
	}
}

func TestE2EZeroGravity(t *testing.T) {
	out, err := run(t, "-g", "0", "1t")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Zero gravity — no thrust needed to hover.") {
		t.Errorf("missing zero-gravity message:\n%s", out)
	}
}

func TestE2EUnknownShortcut(t *testing.T) {
	out, err := run(t, "3*S9")
	if err == nil {
		t.Fatalf("want error for unknown shortcut, got output:\n%s", out)
	}
	msg := err.Error()
	if !strings.Contains(msg, "S9") || !strings.Contains(msg, "s1") || !strings.Contains(msg, "s2") {
		t.Errorf("error should name S9 and list s1, s2: %q", msg)
	}
}

func TestE2EDefaultConfigSmoke(t *testing.T) {
	// No --config: embedded SE2 defaults must load and produce a report.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	// Uppercase S15M/S25M also exercises case-insensitive matching.
	root.SetArgs([]string{"-g", "0.5", "1.23t", "+", "2*S15M", "+", "S25M"})
	if err := root.Execute(); err != nil {
		t.Fatalf("run with defaults: %v\n%s", err, out.String())
	}
	for _, want := range []string{"Atmospheric", "Hydrogen", "Cargo Container 1.5 m", "MW", "units/s H2"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("default-config output missing %q:\n%s", want, out.String())
		}
	}
}
