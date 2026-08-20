package cmd

import (
	"bytes"
	"strings"
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
	want := "secalc version 0.2.0\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func TestNoArgsShowsHelp(t *testing.T) {
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{})

	if err := root.Execute(); err != nil {
		t.Fatalf("no args: want help with nil error, got %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"secalc [flags] <expression>...",
		"secalc -g 0.5 1.23t + 2*s15m + s25m",
		"secalc --game se1 -g moon --full 1t + 2*lgl",
		"Available Commands:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("help output missing %q:\n%s", want, got)
		}
	}
}
