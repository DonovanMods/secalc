package cmd

import (
	"bytes"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("--version: unexpected error: %v", err)
	}
	want := "se2calc version 0.1.0\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}
