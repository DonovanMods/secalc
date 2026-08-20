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

func TestInitWritesDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	out, err := runInit(t)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	path := filepath.Join(dir, "secalc", "config-se2.toml")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	want, err := config.DefaultTOML("se2")
	if err != nil {
		t.Fatalf("DefaultTOML(se2): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Error("written config differs from embedded defaults")
	}
	if !strings.Contains(out, path) {
		t.Errorf("init should print the written path %q, got:\n%s", path, out)
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if _, err := runInit(t); err != nil {
		t.Fatalf("first init: %v", err)
	}
	if _, err := runInit(t); err == nil {
		t.Fatal("second init without --force: want error")
	} else if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention --force: %v", err)
	}
}

func TestInitForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "secalc", "config-se2.toml")
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
	want, err := config.DefaultTOML("se2")
	if err != nil {
		t.Fatalf("DefaultTOML(se2): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Error("--force should overwrite with embedded defaults")
	}
}
