// Package parse turns the CLI mass expression ("1.23t + 2*s15m + s25m")
// into typed terms. It is pure: no I/O, no config dependency.
package parse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Kind discriminates Term variants.
type Kind int

const (
	// Mass is a literal mass term such as "1.23t" or "500".
	Mass Kind = iota
	// Container is a config-defined storage shortcut such as "s15m".
	Container
)

// Term is one parsed expression term.
type Term struct {
	Kind   Kind
	Count  int     // multiplier, >= 1
	Raw    string  // term text as typed, count prefix included
	MassKg float64 // set when Kind == Mass; per unit (total = Count*MassKg)
	Key    string  // set when Kind == Container; lowercased config key
}

// ParseError reports a term that does not match the grammar.
type ParseError struct {
	Token string
	Msg   string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("cannot parse %q: %s", e.Token, e.Msg)
}

// UnknownShortcutError reports a shortcut absent from the config.
type UnknownShortcutError struct {
	Token string
	Known []string
}

func (e *UnknownShortcutError) Error() string {
	return fmt.Sprintf("unknown shortcut %q (available: %s)",
		e.Token, strings.Join(e.Known, ", "))
}

var (
	countRe = regexp.MustCompile(`^(\d+)\s*[*xX]\s*(\S.*)$`)
	massRe  = regexp.MustCompile(`^(\d[\d,]*(?:\.\d+)?)\s*([a-zA-Z]*)$`)
	nameRe  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
)

// Expression parses CLI args (joined with spaces, split on "+") into
// terms. shortcuts holds the known container keys, lowercase.
func Expression(args []string, shortcuts []string) ([]Term, error) {
	known := make(map[string]bool, len(shortcuts))
	for _, s := range shortcuts {
		known[strings.ToLower(s)] = true
	}

	joined := strings.Join(args, " ")
	var terms []Term
	for _, raw := range strings.Split(joined, "+") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, &ParseError{Token: joined, Msg: "empty term (check for a stray '+')"}
		}
		term, err := parseTerm(raw, known, shortcuts)
		if err != nil {
			return nil, err
		}
		terms = append(terms, term)
	}
	return terms, nil
}

func parseTerm(raw string, known map[string]bool, shortcuts []string) (Term, error) {
	count := 1
	body := raw
	if m := countRe.FindStringSubmatch(raw); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 1 {
			return Term{}, &ParseError{Token: raw, Msg: "count must be a positive integer"}
		}
		count = n
		body = strings.TrimSpace(m[2])
	}

	if m := massRe.FindStringSubmatch(body); m != nil {
		kg, err := massKg(m[1], m[2])
		if err != nil {
			return Term{}, &ParseError{Token: raw, Msg: err.Error()}
		}
		return Term{Kind: Mass, Count: count, Raw: raw, MassKg: kg}, nil
	}

	if !nameRe.MatchString(body) {
		return Term{}, &ParseError{Token: raw, Msg: "not a mass (e.g. 500kg, 1.23t) or a storage shortcut"}
	}
	key := strings.ToLower(body)
	if !known[key] {
		return Term{}, &UnknownShortcutError{Token: body, Known: shortcuts}
	}
	return Term{Kind: Container, Count: count, Raw: raw, Key: key}, nil
}

func massKg(num, unit string) (float64, error) {
	v, err := strconv.ParseFloat(strings.ReplaceAll(num, ",", ""), 64)
	if err != nil {
		return 0, fmt.Errorf("bad number %q", num)
	}
	switch strings.ToLower(unit) {
	case "", "kg":
		return v, nil
	case "t":
		return v * 1000, nil
	default:
		return 0, fmt.Errorf("unknown unit %q (use kg or t)", unit)
	}
}
