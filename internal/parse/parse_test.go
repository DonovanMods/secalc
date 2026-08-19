package parse

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

var shortcuts = []string{"s1", "s2", "s15", "s25"}

func TestExpressionValid(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []Term
	}{
		{
			name: "canonical example",
			args: []string{"1.23t", "+", "2*S1", "+", "S2"},
			want: []Term{
				{Kind: Mass, Count: 1, Raw: "1.23t", MassKg: 1230},
				{Kind: Container, Count: 2, Raw: "2*S1", Key: "s1"},
				{Kind: Container, Count: 1, Raw: "S2", Key: "s2"},
			},
		},
		{
			name: "bare number is kilograms",
			args: []string{"500"},
			want: []Term{{Kind: Mass, Count: 1, Raw: "500", MassKg: 500}},
		},
		{
			name: "comma separators and explicit kg",
			args: []string{"1,230kg"},
			want: []Term{{Kind: Mass, Count: 1, Raw: "1,230kg", MassKg: 1230}},
		},
		{
			name: "space before unit",
			args: []string{"1230", "kg"},
			want: []Term{{Kind: Mass, Count: 1, Raw: "1230 kg", MassKg: 1230}},
		},
		{
			name: "x multiplier and spaced asterisk",
			args: []string{"2xS1", "+", "3", "*", "s2"},
			want: []Term{
				{Kind: Container, Count: 2, Raw: "2xS1", Key: "s1"},
				{Kind: Container, Count: 3, Raw: "3 * s2", Key: "s2"},
			},
		},
		{
			name: "uppercase X multiplier and uppercase T unit",
			args: []string{"2XS1", "+", "1.5T"},
			want: []Term{
				{Kind: Container, Count: 2, Raw: "2XS1", Key: "s1"},
				{Kind: Mass, Count: 1, Raw: "1.5T", MassKg: 1500},
			},
		},
		{
			name: "count on a mass literal multiplies per-unit mass",
			args: []string{"3*500kg"},
			want: []Term{{Kind: Mass, Count: 3, Raw: "3*500kg", MassKg: 500}},
		},
		{
			name: "everything in one argument",
			args: []string{"1t+2*s1+s2"},
			want: []Term{
				{Kind: Mass, Count: 1, Raw: "1t", MassKg: 1000},
				{Kind: Container, Count: 2, Raw: "2*s1", Key: "s1"},
				{Kind: Container, Count: 1, Raw: "s2", Key: "s2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Expression(tt.args, shortcuts)
			if err != nil {
				t.Fatalf("Expression(%v): %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expression(%v)\n got %+v\nwant %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestExpressionUnknownShortcut(t *testing.T) {
	_, err := Expression([]string{"2*S9"}, shortcuts)
	var unknown *UnknownShortcutError
	if !errors.As(err, &unknown) {
		t.Fatalf("want UnknownShortcutError, got %v", err)
	}
	if unknown.Token != "S9" {
		t.Errorf("Token = %q, want S9", unknown.Token)
	}
	msg := err.Error()
	if !strings.Contains(msg, "s1") || !strings.Contains(msg, "s2") {
		t.Errorf("error should list known shortcuts, got %q", msg)
	}
}

func TestExpressionParseErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"trailing plus", []string{"1t", "+"}},
		{"leading plus", []string{"+", "1t"}},
		{"double plus", []string{"1t", "+", "+", "s1"}},
		{"unknown unit", []string{"5lbs"}},
		{"fractional count", []string{"2.5*s1"}},
		{"zero count", []string{"0*s1"}},
		{"missing operator between terms", []string{"s1 s2"}},
		{"empty input", []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Expression(tt.args, shortcuts)
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("Expression(%v): want ParseError, got %v", tt.args, err)
			}
		})
	}
}
