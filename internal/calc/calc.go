// Package calc computes ship mass totals and per-size thruster
// requirements. Pure math: no I/O.
package calc

import (
	"fmt"
	"math"
	"sort"

	"github.com/DonovanMods/secalc/internal/config"
	"github.com/DonovanMods/secalc/internal/parse"
)

// g0 is m/s^2 at 1 g (community-verified for SE2; see spec).
const g0 = 9.81

// epsilon guards math.Ceil against float error on exact divisions.
const epsilon = 1e-9

// MassItem is one line of the mass breakdown.
type MassItem struct {
	Label  string  // e.g. "2 x Cargo Container 1.5 m (full)" or the raw mass token
	MassKg float64 // total contribution of this line
}

// SizeResult is one independent thruster solution.
type SizeResult struct {
	Name   string
	Count  int
	Power  float64 // total consumption, in the family's PowerUnit
	Viable bool    // false: cannot lift its own weight at this gravity/margin
}

// FamilyResult groups the size solutions of one thruster family.
type FamilyResult struct {
	Name      string
	PowerUnit string
	Sizes     []SizeResult // ascending thrust
}

// Plan is the full calculation result, ready to render.
type Plan struct {
	Game        string // uppercase display id, e.g. "SE2"
	GravityMult float64
	Margin      float64
	Full        bool
	Items       []MassItem
	TotalKg     float64
	Families    []FamilyResult // sorted by family name; empty in zero gravity
}

// Input carries everything Build needs. Margin must already be resolved
// (config default or -m flag).
type Input struct {
	Game         string // uppercase display id, e.g. "SE2"; passthrough to Plan
	Terms        []parse.Term
	Cfg          *config.Config
	GravityMult  float64
	Margin       float64
	Full         bool
	CargoDensity float64 // kg/L, resolved from config settings; used by volumetric containers
}

// Build computes the mass breakdown and per-size thruster requirements.
func Build(in Input) (*Plan, error) {
	if in.GravityMult < 0 {
		return nil, fmt.Errorf("gravity multiplier must be >= 0 (got %g)", in.GravityMult)
	}
	if in.Margin <= 0 {
		return nil, fmt.Errorf("margin must be > 0 (got %g)", in.Margin)
	}

	p := &Plan{Game: in.Game, GravityMult: in.GravityMult, Margin: in.Margin, Full: in.Full}
	for _, t := range in.Terms {
		item, err := massItem(t, in.Cfg, in.Full, in.CargoDensity)
		if err != nil {
			return nil, err
		}
		p.Items = append(p.Items, item)
		p.TotalKg += item.MassKg
	}

	if in.GravityMult == 0 {
		return p, nil // nothing to lift; render notes zero gravity
	}
	p.Families = families(p.TotalKg, in.GravityMult, in.Margin, in.Cfg)
	return p, nil
}

func massItem(t parse.Term, cfg *config.Config, full bool, density float64) (MassItem, error) {
	switch t.Kind {
	case parse.Mass:
		return MassItem{Label: t.Raw, MassKg: float64(t.Count) * t.MassKg}, nil
	case parse.Container:
		c, ok := cfg.Containers[t.Key]
		if !ok {
			return MassItem{}, fmt.Errorf("container %q not in config", t.Key)
		}
		unit := c.Mass
		label := c.Name
		if full {
			unit += c.FullCargoKg(density)
			label += " (full)"
		}
		if t.Count > 1 {
			label = fmt.Sprintf("%d x %s", t.Count, label)
		}
		return MassItem{Label: label, MassKg: float64(t.Count) * unit}, nil
	default:
		return MassItem{}, fmt.Errorf("unknown term kind %d", t.Kind)
	}
}

func families(totalKg, gravityMult, margin float64, cfg *config.Config) []FamilyResult {
	// weightFactor converts kg to required newtons: g_eff * margin.
	weightFactor := g0 * gravityMult * margin
	needN := totalKg * weightFactor

	out := make([]FamilyResult, 0, len(cfg.Thrusters))
	for _, fam := range cfg.Thrusters {
		sizes := make([]config.ThrusterSize, 0, len(fam.Sizes))
		for _, s := range fam.Sizes {
			sizes = append(sizes, s)
		}
		sort.Slice(sizes, func(i, j int) bool { return sizes[i].Thrust < sizes[j].Thrust })

		fr := FamilyResult{Name: fam.Name, PowerUnit: fam.PowerUnit}
		for _, s := range sizes {
			fr.Sizes = append(fr.Sizes, sizeResult(needN, weightFactor, s))
		}
		out = append(out, fr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// sizeResult solves n*thrust >= (M + n*mass) * gEff * margin for the
// smallest integer n. denom <= 0 means the thruster cannot lift its own
// weight under these conditions.
func sizeResult(needN, weightFactor float64, s config.ThrusterSize) SizeResult {
	denom := s.Thrust - s.Mass*weightFactor
	if denom <= 0 {
		return SizeResult{Name: s.Name, Viable: false}
	}
	n := int(math.Ceil(needN/denom - epsilon))
	if n < 0 {
		n = 0
	}
	return SizeResult{Name: s.Name, Count: n, Power: float64(n) * s.Power, Viable: true}
}
