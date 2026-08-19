package calc

import (
	"testing"

	"github.com/DonovanMods/se2calc/internal/config"
	"github.com/DonovanMods/se2calc/internal/parse"
)

// testCfg builds a minimal config with one container and one
// single-size thruster family, keeping expectations hand-checkable.
func testCfg(thrust, thrusterMass, power float64) *config.Config {
	return &config.Config{
		Settings: config.Settings{Margin: 1.5},
		Containers: map[string]config.Container{
			"s1": {Name: "Small Box", Mass: 100, Capacity: 900},
		},
		Thrusters: map[string]config.ThrusterFamily{
			"lift": {
				Name:      "Lift",
				PowerUnit: "MW",
				Sizes: map[string]config.ThrusterSize{
					"a": {Name: "A", Thrust: thrust, Mass: thrusterMass, Power: power},
				},
			},
		},
	}
}

func massTerm(raw string, kg float64) parse.Term {
	return parse.Term{Kind: parse.Mass, Count: 1, Raw: raw, MassKg: kg}
}

func TestBuildMassBreakdown(t *testing.T) {
	cfg := testCfg(50000, 0, 0.5)
	in := Input{
		Terms: []parse.Term{
			massTerm("1t", 1000),
			{Kind: parse.Container, Count: 2, Raw: "2*s1", Key: "s1"},
		},
		Cfg:         cfg,
		GravityMult: 1,
		Margin:      1.5,
	}

	p, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if p.TotalKg != 1200 {
		t.Errorf("TotalKg = %g, want 1200 (1000 + 2*100 empty)", p.TotalKg)
	}
	if len(p.Items) != 2 {
		t.Fatalf("Items = %+v, want 2 entries", p.Items)
	}
	if p.Items[0].Label != "1t" || p.Items[0].MassKg != 1000 {
		t.Errorf("Items[0] = %+v, want label 1t / 1000 kg", p.Items[0])
	}
	if p.Items[1].Label != "2 x Small Box" || p.Items[1].MassKg != 200 {
		t.Errorf("Items[1] = %+v, want label \"2 x Small Box\" / 200 kg", p.Items[1])
	}
}

func TestBuildFullContainers(t *testing.T) {
	cfg := testCfg(50000, 0, 0.5)
	in := Input{
		Terms:       []parse.Term{{Kind: parse.Container, Count: 2, Raw: "2*s1", Key: "s1"}},
		Cfg:         cfg,
		GravityMult: 1,
		Margin:      1.5,
		Full:        true,
	}

	p, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if p.TotalKg != 2000 {
		t.Errorf("TotalKg = %g, want 2000 (2 * (100 + 900))", p.TotalKg)
	}
	if p.Items[0].Label != "2 x Small Box (full)" {
		t.Errorf("Label = %q, want \"2 x Small Box (full)\"", p.Items[0].Label)
	}
}

func TestBuildThrusterCounts(t *testing.T) {
	tests := []struct {
		name         string
		thrust       float64 // N
		thrusterMass float64 // kg
		shipKg       float64
		gravity      float64
		margin       float64
		wantCount    int
		wantViable   bool
	}{
		{
			// need = 10000*9.81*1.5 = 147150 N.
			// denom = 50000 - 1000*14.715 = 35285. 147150/35285 = 4.17 -> 5.
			// (Ignoring thruster self-mass would give ceil(2.943) = 3 —
			// this case proves self-mass feedback is applied.)
			name: "self-mass feedback", thrust: 50000, thrusterMass: 1000,
			shipKg: 10000, gravity: 1, margin: 1.5,
			wantCount: 5, wantViable: true,
		},
		{
			// denom = 10000 - 1000*14.715 < 0: cannot lift its own weight.
			name: "not viable", thrust: 10000, thrusterMass: 1000,
			shipKg: 10000, gravity: 1, margin: 1.5,
			wantViable: false,
		},
		{
			// Exact division: need = 1000*9.81 = 9810, denom = 9810.
			// The epsilon must keep this at exactly 1, not 2.
			name: "exact division", thrust: 9810, thrusterMass: 0,
			shipKg: 1000, gravity: 1, margin: 1,
			wantCount: 1, wantViable: true,
		},
		{
			// Low gravity scales the requirement: need = 10000*4.905*1.5
			// = 73575, denom = 50000 - 1000*7.3575 = 42642.5 -> ceil(1.725) = 2.
			name: "half gravity", thrust: 50000, thrusterMass: 1000,
			shipKg: 10000, gravity: 0.5, margin: 1.5,
			wantCount: 2, wantViable: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testCfg(tt.thrust, tt.thrusterMass, 0.5)
			in := Input{
				Terms:       []parse.Term{massTerm("m", tt.shipKg)},
				Cfg:         cfg,
				GravityMult: tt.gravity,
				Margin:      tt.margin,
			}
			p, err := Build(in)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if len(p.Families) != 1 || len(p.Families[0].Sizes) != 1 {
				t.Fatalf("Families = %+v, want 1 family with 1 size", p.Families)
			}
			got := p.Families[0].Sizes[0]
			if got.Viable != tt.wantViable {
				t.Fatalf("Viable = %v, want %v", got.Viable, tt.wantViable)
			}
			if tt.wantViable && got.Count != tt.wantCount {
				t.Errorf("Count = %d, want %d", got.Count, tt.wantCount)
			}
			if tt.wantViable {
				wantPower := float64(tt.wantCount) * 0.5
				if got.Power != wantPower {
					t.Errorf("Power = %g, want %g", got.Power, wantPower)
				}
			}
		})
	}
}

func TestBuildZeroGravity(t *testing.T) {
	cfg := testCfg(50000, 0, 0.5)
	in := Input{
		Terms:       []parse.Term{massTerm("1t", 1000)},
		Cfg:         cfg,
		GravityMult: 0,
		Margin:      1.5,
	}
	p, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(p.Families) != 0 {
		t.Errorf("Families = %+v, want none in zero gravity", p.Families)
	}
}

func TestBuildSortsFamiliesAndSizes(t *testing.T) {
	cfg := testCfg(50000, 0, 0.5)
	cfg.Thrusters["lift"] = config.ThrusterFamily{
		Name:      "Lift",
		PowerUnit: "MW",
		Sizes: map[string]config.ThrusterSize{
			"big":   {Name: "Big", Thrust: 90000, Mass: 0, Power: 2},
			"small": {Name: "Small", Thrust: 10000, Mass: 0, Power: 0.1},
		},
	}
	cfg.Thrusters["aux"] = config.ThrusterFamily{
		Name:      "Aux",
		PowerUnit: "MW",
		Sizes: map[string]config.ThrusterSize{
			"only": {Name: "Only", Thrust: 20000, Mass: 0, Power: 1},
		},
	}

	p, err := Build(Input{
		Terms:       []parse.Term{massTerm("1t", 1000)},
		Cfg:         cfg,
		GravityMult: 1,
		Margin:      1.5,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if p.Families[0].Name != "Aux" || p.Families[1].Name != "Lift" {
		t.Errorf("family order = %s, %s; want Aux, Lift", p.Families[0].Name, p.Families[1].Name)
	}
	lift := p.Families[1]
	if lift.Sizes[0].Name != "Small" || lift.Sizes[1].Name != "Big" {
		t.Errorf("size order = %s, %s; want Small (ascending thrust), Big", lift.Sizes[0].Name, lift.Sizes[1].Name)
	}
}

func TestBuildValidatesInputs(t *testing.T) {
	cfg := testCfg(50000, 0, 0.5)
	terms := []parse.Term{massTerm("1t", 1000)}

	if _, err := Build(Input{Terms: terms, Cfg: cfg, GravityMult: -1, Margin: 1.5}); err == nil {
		t.Error("negative gravity: want error")
	}
	if _, err := Build(Input{Terms: terms, Cfg: cfg, GravityMult: 1, Margin: 0}); err == nil {
		t.Error("zero margin: want error")
	}
}
