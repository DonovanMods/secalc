package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points os.UserConfigDir at a temp dir so the developer's real
// ~/.config/secalc/config-*.toml never leaks into tests. It returns the
// temp config dir.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func writeUserConfig(t *testing.T, dir, game, content string) {
	t.Helper()
	cfgDir := filepath.Join(dir, "secalc")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config-"+game+".toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDefaults(t *testing.T) {
	isolate(t)

	cfg, err := Load("se2", "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Settings.Margin != 1.5 {
		t.Errorf("Margin = %g, want 1.5", cfg.Settings.Margin)
	}
	s15m, ok := cfg.Containers["s15m"]
	if !ok {
		t.Fatalf("containers missing s15m; have %v", cfg.ShortcutKeys())
	}
	if s15m.Mass != 245.17 || s15m.Capacity != 16800 {
		t.Errorf("s15m = %+v, want mass 245.17 capacity 16800", s15m)
	}
	atmo, ok := cfg.Thrusters["atmospheric"]
	if !ok {
		t.Fatal("thrusters missing atmospheric")
	}
	if atmo.PowerUnit != "MW" || len(atmo.Sizes) != 4 {
		t.Errorf("atmospheric = %+v, want power_unit MW and 4 sizes", atmo)
	}
	if got := atmo.Sizes["s1m"].Thrust; got != 40000 {
		t.Errorf("atmospheric s1m thrust = %g, want 40000", got)
	}
	if _, ok := cfg.Thrusters["ion"]; ok {
		t.Error("ion thrusters must be commented out in defaults")
	}
}

func TestLoadUserOverrideMergesPerKey(t *testing.T) {
	dir := isolate(t)
	writeUserConfig(t, dir, "se2", "[settings]\nmargin = 2.0\n")

	cfg, err := Load("se2", "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Settings.Margin != 2.0 {
		t.Errorf("Margin = %g, want 2.0 from user file", cfg.Settings.Margin)
	}
	// Everything not overridden must survive the merge.
	if len(cfg.Containers) != 3 {
		t.Errorf("containers = %v, want the 3 defaults", cfg.ShortcutKeys())
	}
}

func TestLoadOverrideFileWinsAndMergesDeep(t *testing.T) {
	dir := isolate(t)
	writeUserConfig(t, dir, "se2", "[settings]\nmargin = 2.0\n")

	override := filepath.Join(t.TempDir(), "override.toml")
	if err := os.WriteFile(override, []byte("[containers.s15m]\nmass = 999\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("se2", override)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Settings.Margin != 2.0 {
		t.Errorf("Margin = %g, want 2.0 (user file survives)", cfg.Settings.Margin)
	}
	if cfg.Containers["s15m"].Mass != 999 {
		t.Errorf("s15m.Mass = %g, want 999 from override", cfg.Containers["s15m"].Mass)
	}
	if cfg.Containers["s15m"].Capacity != 16800 {
		t.Errorf("s15m.Capacity = %g, want 16800 (sibling key survives deep merge)", cfg.Containers["s15m"].Capacity)
	}
}

func TestLoadMissingOverrideFileErrors(t *testing.T) {
	isolate(t)
	if _, err := Load("se2", filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("Load with missing --config file: want error, got nil")
	}
}

func TestLoadValidatesMargin(t *testing.T) {
	dir := isolate(t)
	writeUserConfig(t, dir, "se2", "[settings]\nmargin = -1\n")

	_, err := Load("se2", "")
	if err == nil || !strings.Contains(err.Error(), "settings.margin") {
		t.Fatalf("want settings.margin validation error, got %v", err)
	}
}

func TestLoadRejectsDigitLeadingContainerKey(t *testing.T) {
	dir := isolate(t)
	writeUserConfig(t, dir, "se2", "[containers.2m]\nname = \"Two Metre Box\"\nmass = 500\ncapacity = 1000\n")

	_, err := Load("se2", "")
	if err == nil || !strings.Contains(err.Error(), "containers.2m") {
		t.Fatalf("want containers.2m validation error, got %v", err)
	}
}

func TestUserConfigPath(t *testing.T) {
	dir := isolate(t)
	got, err := UserConfigPath("se1")
	if err != nil {
		t.Fatalf("UserConfigPath: %v", err)
	}
	want := filepath.Join(dir, "secalc", "config-se1.toml")
	if got != want {
		t.Errorf("UserConfigPath = %q, want %q", got, want)
	}
}

func TestLoadDefaultGravityPresets(t *testing.T) {
	isolate(t)
	cfg, err := Load("se2", "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := map[string]float64{
		"verdure":  1.0,
		"palatine": 0.33,
		"caligo":   0.416,
		"space":    0,
	}
	for name, mult := range want {
		got, ok := cfg.Gravity[name]
		if !ok {
			t.Errorf("gravity preset %q missing from defaults", name)
			continue
		}
		if got != mult {
			t.Errorf("gravity[%q] = %g, want %g", name, got, mult)
		}
	}
}

func TestLoadRejectsBadGravityPresets(t *testing.T) {
	tests := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{"digit-leading key", "[gravity]\n\"2x\" = 0.5\n", "gravity.2x"},
		{"negative value", "[gravity]\npit = -1\n", "gravity.pit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := isolate(t)
			writeUserConfig(t, dir, "se2", tt.toml)
			_, err := Load("se2", "")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestShortcutKeysSorted(t *testing.T) {
	isolate(t)
	cfg, err := Load("se2", "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.ShortcutKeys()
	want := []string{"s15m", "s25m", "s75m"}
	if len(got) != len(want) {
		t.Fatalf("ShortcutKeys = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ShortcutKeys = %v, want %v", got, want)
		}
	}
}

func TestContainerFullCargoKg(t *testing.T) {
	tests := []struct {
		name string
		c    Container
		dens float64
		want float64
	}{
		{"mass-limited", Container{Capacity: 16800}, 0, 16800},
		{"volumetric", Container{CapacityL: 1000}, 2.5, 2500},
		{"holds nothing", Container{}, 2.5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.FullCargoKg(tt.dens); got != tt.want {
				t.Errorf("FullCargoKg = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestLoadValidatesVolumetricCargo(t *testing.T) {
	tests := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{
			"both capacities set",
			"[containers.dual]\nname = \"Dual\"\nmass = 1\ncapacity = 5\ncapacity_l = 5\n",
			"containers.dual",
		},
		{
			"capacity_l without density",
			"[containers.vol]\nname = \"Vol\"\nmass = 1\ncapacity_l = 5\n",
			"cargo_density",
		},
		{
			"negative capacity_l",
			"[settings]\ncargo_density = 2.5\n[containers.vol]\nname = \"Vol\"\nmass = 1\ncapacity_l = -1\n",
			"containers.vol",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := isolate(t)
			writeUserConfig(t, dir, "se2", tt.toml)
			_, err := Load("se2", "")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestLoadAcceptsVolumetricContainer(t *testing.T) {
	dir := isolate(t)
	writeUserConfig(t, dir, "se2",
		"[settings]\ncargo_density = 2.7027\n[containers.vol]\nname = \"Vol Box\"\nmass = 100\ncapacity_l = 1000\n")
	cfg, err := Load("se2", "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Containers["vol"].CapacityL != 1000 || cfg.Settings.CargoDensity != 2.7027 {
		t.Errorf("volumetric fields not loaded: %+v, density %g",
			cfg.Containers["vol"], cfg.Settings.CargoDensity)
	}
}

func TestGamesOrder(t *testing.T) {
	got := Games()
	want := []string{"se2", "se1"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Games() = %v, want %v", got, want)
	}
}

func TestDefaultTOMLUnknownGame(t *testing.T) {
	if _, err := DefaultTOML("se3"); err == nil ||
		!strings.Contains(err.Error(), "se3") || !strings.Contains(err.Error(), "se1") {
		t.Fatalf("want unknown-game error naming se3 and listing valid ids, got %v", err)
	}
}

func TestLoadSE1Defaults(t *testing.T) {
	isolate(t)
	cfg, err := Load("se1", "")
	if err != nil {
		t.Fatalf("Load(se1): %v", err)
	}
	if cfg.Settings.CargoDensity != 2.7027 {
		t.Errorf("cargo_density = %g, want 2.7027", cfg.Settings.CargoDensity)
	}
	lgl, ok := cfg.Containers["lgl"]
	if !ok || lgl.CapacityL != 421000 || lgl.Mass != 2593.6 {
		t.Errorf("lgl = %+v, want mass 2593.6 capacity_l 421000", lgl)
	}
	if len(cfg.Thrusters) != 6 {
		t.Errorf("thruster families = %d, want 6", len(cfg.Thrusters))
	}
	if got := cfg.Thrusters["hydro_lg"].Sizes["large"].Thrust; got != 7200000 {
		t.Errorf("hydro_lg large thrust = %g, want 7200000", got)
	}
	if cfg.Gravity["pertam"] != 1.2 || cfg.Gravity["moon"] != 0.25 {
		t.Errorf("gravity presets wrong: %+v", cfg.Gravity)
	}
}

func TestLoadUnknownGame(t *testing.T) {
	isolate(t)
	if _, err := Load("se3", ""); err == nil {
		t.Fatal("Load(se3): want error")
	}
}

func TestPerGameUserFilesAreIsolated(t *testing.T) {
	dir := isolate(t)
	// An SE2 override must not leak into SE1 loads.
	writeUserConfig(t, dir, "se2", "[settings]\nmargin = 9.9\n") // writes config-se2.toml
	se1, err := Load("se1", "")
	if err != nil {
		t.Fatalf("Load(se1): %v", err)
	}
	if se1.Settings.Margin == 9.9 {
		t.Error("SE2 user file leaked into SE1 load")
	}
	se2, err := Load("se2", "")
	if err != nil {
		t.Fatalf("Load(se2): %v", err)
	}
	if se2.Settings.Margin != 9.9 {
		t.Errorf("SE2 user file not applied to SE2 load: margin %g", se2.Settings.Margin)
	}
}
