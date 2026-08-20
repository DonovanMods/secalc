package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DonovanMods/secalc/internal/config"
)

func runInit(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"init"}, args...))
	err := root.Execute()
	return out.String(), err
}

func mustDefault(t *testing.T, game string) []byte {
	t.Helper()
	b, err := config.DefaultTOML(game)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestInitWritesBothGames(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	out, err := runInit(t)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	for _, game := range config.Games() {
		path := filepath.Join(dir, "secalc", "config-"+game+".toml")
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if !bytes.Equal(got, mustDefault(t, game)) {
			t.Errorf("%s differs from embedded %s defaults", path, game)
		}
		if !strings.Contains(out, path) {
			t.Errorf("init output should name %s:\n%s", path, out)
		}
	}
}

func TestInitSkipsExistingWithoutForce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if _, err := runInit(t); err != nil {
		t.Fatalf("first init: %v", err)
	}
	// Corrupt one file, then re-run without --force: both skipped, exit 0.
	se2Path := filepath.Join(dir, "secalc", "config-se2.toml")
	if err := os.WriteFile(se2Path, []byte("# user-tweaked"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runInit(t)
	if err != nil {
		t.Fatalf("second init should not error: %v", err)
	}
	if !strings.Contains(out, "Skipped") || !strings.Contains(out, "--force") {
		t.Errorf("want per-file skip notes mentioning --force:\n%s", out)
	}
	got, _ := os.ReadFile(se2Path)
	if !bytes.Equal(got, []byte("# user-tweaked")) {
		t.Error("skip must not touch the existing file")
	}
}

func TestInitForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "secalc", "config-se1.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runInit(t, "--force"); err != nil {
		t.Fatalf("init --force: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, mustDefault(t, "se1")) {
		t.Error("--force should overwrite with embedded se1 defaults")
	}
}
