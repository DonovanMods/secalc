// Package output renders a calc.Plan as a human-readable report.
package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/DonovanMods/secalc/internal/calc"
)

// separator is built with Repeat so its rune count provably matches the
// golden test's strings.Repeat("─", 42).
var separator = strings.Repeat("─", 42)

// Render writes the report for p to w.
func Render(w io.Writer, p *calc.Plan) {
	fmt.Fprintf(w, "secalc — %s\n", p.Game)
	fmt.Fprintln(w, separator)
	fmt.Fprintf(w, "Gravity: %s g   Target TWR: %s\n\n", num(p.GravityMult), num(p.Margin))

	fmt.Fprintln(w, "Mass breakdown:")
	labelW := 0
	for _, it := range p.Items {
		if n := utf8.RuneCountInString(it.Label); n > labelW {
			labelW = n
		}
	}
	for _, it := range p.Items {
		fmt.Fprintf(w, "  %s  %9s\n", pad(it.Label, labelW), mass(it.MassKg))
	}
	fmt.Fprintln(w, separator)
	suffix := ""
	if p.Full {
		suffix = " (full)"
	}
	fmt.Fprintf(w, "Total ship mass: %s%s\n", mass(p.TotalKg), suffix)

	if p.GravityMult == 0 {
		fmt.Fprintln(w, "\nZero gravity — no thrust needed to hover.")
		return
	}

	fmt.Fprintln(w, "\nThrusters needed to overcome gravity")
	fmt.Fprintln(w, "(each line is a complete, independent solution):")
	for _, fam := range p.Families {
		fmt.Fprintf(w, "\n  %s\n", fam.Name)
		sizeW := 0
		for _, s := range fam.Sizes {
			if n := utf8.RuneCountInString(s.Name); n > sizeW {
				sizeW = n
			}
		}
		for _, s := range fam.Sizes {
			if !s.Viable {
				fmt.Fprintf(w, "    %s  not viable (cannot lift own weight at this gravity/margin)\n",
					pad(s.Name, sizeW))
				continue
			}
			fmt.Fprintf(w, "    %s  %4d   %s %s\n", pad(s.Name, sizeW), s.Count, num(s.Power), fam.PowerUnit)
		}
	}
}

// pad right-pads s with spaces to width runes. fmt's %-*s pads by bytes
// and would misalign labels containing multi-byte runes (user-configured
// names can hold any UTF-8).
func pad(s string, width int) string {
	if n := width - utf8.RuneCountInString(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// num formats a value with up to 3 decimals, trimming trailing zeros —
// never scientific notation, and float noise (1.9500000000000002)
// renders clean (1.95).
func num(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

// mass renders kilograms: kg with up-to-3 trimmed decimals below one
// tonne (so a typed 1.34 echoes as 1.34 kg, not 1 kg), tonnes with two
// decimals otherwise.
func mass(kg float64) string {
	if kg < 1000 {
		return num(kg) + " kg"
	}
	return fmt.Sprintf("%.2f t", kg/1000)
}
