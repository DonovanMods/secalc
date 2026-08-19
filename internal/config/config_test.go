package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points os.UserConfigDir at a temp dir so the developer's real
// ~/.config/se2calc/config.toml never leaks into tests. It returns the
// temp config dir.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func writeUserConfig(t *testing.T, dir, content string) {
	t.Helper()
	cfgDir := filepath.Join(dir, "se2calc")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDefaults(t *testing.T) {
	isolate(t)

	cfg, err := Load("")
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
	writeUserConfig(t, dir, "[settings]\nmargin = 2.0\n")

	cfg, err := Load("")
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
	writeUserConfig(t, dir, "[settings]\nmargin = 2.0\n")

	override := filepath.Join(t.TempDir(), "override.toml")
	if err := os.WriteFile(override, []byte("[containers.s15m]\nmass = 999\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(override)
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
	if _, err := Load(filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("Load with missing --config file: want error, got nil")
	}
}

func TestLoadValidatesMargin(t *testing.T) {
	dir := isolate(t)
	writeUserConfig(t, dir, "[settings]\nmargin = -1\n")

	_, err := Load("")
	if err == nil || !strings.Contains(err.Error(), "settings.margin") {
		t.Fatalf("want settings.margin validation error, got %v", err)
	}
}

func TestUserConfigPath(t *testing.T) {
	dir := isolate(t)
	got, err := UserConfigPath()
	if err != nil {
		t.Fatalf("UserConfigPath: %v", err)
	}
	want := filepath.Join(dir, "se2calc", "config.toml")
	if got != want {
		t.Errorf("UserConfigPath = %q, want %q", got, want)
	}
}

func TestShortcutKeysSorted(t *testing.T) {
	isolate(t)
	cfg, err := Load("")
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
