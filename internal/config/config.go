// Package config loads secalc settings: embedded defaults merged with the
// user's config file and an optional explicit override. Viper is
// quarantined here — the rest of the program sees only plain structs.
package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

// containerKeyRe is the lowercase image of parse's shortcut grammar
// (nameRe = ^[A-Za-z][A-Za-z0-9_-]*$); Viper lowercases TOML keys before
// validate() sees them, so this is what a container key must match to ever
// be reachable as a CLI shortcut.
var containerKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

//go:embed se2.toml
var se2TOML []byte

//go:embed se1.toml
var se1TOML []byte

// defaultTOMLs maps each supported game id to its embedded default
// configuration, written verbatim by `secalc init`.
var defaultTOMLs = map[string][]byte{"se2": se2TOML, "se1": se1TOML}

// Games returns the supported game ids, primary first.
func Games() []string { return []string{"se2", "se1"} }

// DefaultTOML returns the embedded default configuration for a game.
func DefaultTOML(game string) ([]byte, error) {
	b, ok := defaultTOMLs[game]
	if !ok {
		return nil, fmt.Errorf("unknown game %q (valid: %s)", game, strings.Join(Games(), ", "))
	}
	return b, nil
}

// Settings holds calculator-wide tunables.
type Settings struct {
	Margin       float64 `mapstructure:"margin"`        // target thrust-to-weight ratio
	CargoDensity float64 `mapstructure:"cargo_density"` // kg/L; required by volumetric containers
}

// Container is one storage block; the map key in Config.Containers is its
// CLI shortcut.
type Container struct {
	Name      string  `mapstructure:"name"`
	Mass      float64 `mapstructure:"mass"`       // kg, empty
	Capacity  float64 `mapstructure:"capacity"`   // kg of cargo when full (mass-limited games)
	CapacityL float64 `mapstructure:"capacity_l"` // liters of cargo when full (volumetric games)
}

// FullCargoKg is the cargo mass a full container adds: capacity kg when
// set, otherwise capacity_l converted through the given density.
func (c Container) FullCargoKg(density float64) float64 {
	if c.Capacity > 0 {
		return c.Capacity
	}
	return c.CapacityL * density
}

// ThrusterSize is one size variant within a thruster family.
type ThrusterSize struct {
	Name   string  `mapstructure:"name"`
	Thrust float64 `mapstructure:"thrust"` // N
	Mass   float64 `mapstructure:"mass"`   // kg
	Power  float64 `mapstructure:"power"`  // in the family's PowerUnit
}

// ThrusterFamily groups size variants that share a consumption unit.
type ThrusterFamily struct {
	Name      string                  `mapstructure:"name"`
	PowerUnit string                  `mapstructure:"power_unit"`
	Sizes     map[string]ThrusterSize `mapstructure:"sizes"`
}

// Config is the fully merged, validated configuration.
type Config struct {
	Settings   Settings                  `mapstructure:"settings"`
	Containers map[string]Container      `mapstructure:"containers"`
	Thrusters  map[string]ThrusterFamily `mapstructure:"thrusters"`
	Gravity    map[string]float64        `mapstructure:"gravity"` // named -g presets, in multiples of 1 g
}

// Load returns the embedded defaults for game merged with the user config
// file (if it exists) and then overridePath (if non-empty). Merging is
// per-key, so override files need only the values that differ. A non-empty
// overridePath that cannot be read is an error.
func Load(game, overridePath string) (*Config, error) {
	defaultTOML, err := DefaultTOML(game)
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigType("toml")
	if err := v.ReadConfig(bytes.NewReader(defaultTOML)); err != nil {
		return nil, fmt.Errorf("loading embedded defaults: %w", err)
	}

	if userPath, err := UserConfigPath(game); err == nil {
		if _, statErr := os.Stat(userPath); statErr == nil {
			v.SetConfigFile(userPath)
			if err := v.MergeInConfig(); err != nil {
				return nil, fmt.Errorf("merging %s: %w", userPath, err)
			}
		}
	}

	if overridePath != "" {
		v.SetConfigFile(overridePath)
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("merging %s: %w", overridePath, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UserConfigPath returns the per-game user config file location,
// e.g. ~/.config/secalc/config-se2.toml (os.UserConfigDir respects
// XDG_CONFIG_HOME).
func UserConfigPath(game string) (string, error) {
	if _, err := DefaultTOML(game); err != nil {
		return "", err
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating user config dir: %w", err)
	}
	return filepath.Join(dir, "secalc", "config-"+game+".toml"), nil
}

// ShortcutKeys returns the container shortcut keys, sorted. Viper
// lowercases TOML keys, so these are already lowercase.
func (c *Config) ShortcutKeys() []string {
	keys := make([]string, 0, len(c.Containers))
	for k := range c.Containers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (c *Config) validate() error {
	if c.Settings.Margin <= 0 {
		return fmt.Errorf("config: settings.margin must be > 0 (got %g)", c.Settings.Margin)
	}
	if len(c.Containers) == 0 {
		return fmt.Errorf("config: no containers defined")
	}
	for key, ct := range c.Containers {
		if !containerKeyRe.MatchString(key) {
			return fmt.Errorf("config: containers.%s: key must start with a letter and contain only letters, digits, '_' or '-' (the expression grammar reads digit-leading tokens as masses — try \"s%s\")", key, key)
		}
		if ct.Name == "" {
			return fmt.Errorf("config: containers.%s: name is required", key)
		}
		if ct.Mass < 0 || ct.Capacity < 0 {
			return fmt.Errorf("config: containers.%s: mass and capacity must be >= 0", key)
		}
		if ct.Capacity > 0 && ct.CapacityL > 0 {
			return fmt.Errorf("config: containers.%s: set exactly one of capacity (kg) or capacity_l (liters), not both", key)
		}
		if ct.CapacityL < 0 {
			return fmt.Errorf("config: containers.%s: capacity_l must be >= 0", key)
		}
		if ct.CapacityL > 0 && c.Settings.CargoDensity <= 0 {
			return fmt.Errorf("config: containers.%s uses capacity_l but settings.cargo_density is not set (> 0 kg/L required)", key)
		}
	}
	// Gravity preset keys follow the container-key shape: a numeric-looking
	// name could never win against -g's number parsing.
	for key, mult := range c.Gravity {
		if !containerKeyRe.MatchString(key) {
			return fmt.Errorf("config: gravity.%s: preset name must start with a letter and contain only letters, digits, '_' or '-'", key)
		}
		if mult < 0 {
			return fmt.Errorf("config: gravity.%s: multiplier must be >= 0 (got %g)", key, mult)
		}
	}
	if len(c.Thrusters) == 0 {
		return fmt.Errorf("config: no thruster families defined")
	}
	// Duplicate display names are almost always the copied-config rename
	// trap: user files merge per-key OVER the embedded defaults, so a
	// renamed key yields both the original and the copy. Iterate sorted
	// keys so the reported pair is deterministic.
	familyKeys := make([]string, 0, len(c.Thrusters))
	for fk := range c.Thrusters {
		familyKeys = append(familyKeys, fk)
	}
	sort.Strings(familyKeys)
	familyByName := make(map[string]string, len(familyKeys))
	for _, fk := range familyKeys {
		fam := c.Thrusters[fk]
		if fam.Name == "" || fam.PowerUnit == "" {
			return fmt.Errorf("config: thrusters.%s: name and power_unit are required", fk)
		}
		if prev, dup := familyByName[fam.Name]; dup {
			return fmt.Errorf("config: thruster families %s and %s share the name %q — if you renamed family keys in a copied config, the embedded defaults still provide the originals; keep the original keys and edit values instead", prev, fk, fam.Name)
		}
		familyByName[fam.Name] = fk
		if len(fam.Sizes) == 0 {
			return fmt.Errorf("config: thrusters.%s: at least one size is required", fk)
		}
		sizeKeys := make([]string, 0, len(fam.Sizes))
		for sk := range fam.Sizes {
			sizeKeys = append(sizeKeys, sk)
		}
		sort.Strings(sizeKeys)
		sizeByName := make(map[string]string, len(sizeKeys))
		for _, sk := range sizeKeys {
			s := fam.Sizes[sk]
			if s.Name == "" {
				return fmt.Errorf("config: thrusters.%s.sizes.%s: name is required", fk, sk)
			}
			if prev, dup := sizeByName[s.Name]; dup {
				return fmt.Errorf("config: thrusters.%s: sizes %s and %s share the name %q — if you renamed size keys in a copied config, the embedded defaults still provide the originals; edit values under the original keys instead (size keys are internal and never typed on the CLI)", fk, prev, sk, s.Name)
			}
			sizeByName[s.Name] = sk
			if s.Thrust <= 0 {
				return fmt.Errorf("config: thrusters.%s.sizes.%s: thrust must be > 0", fk, sk)
			}
			if s.Mass < 0 || s.Power < 0 {
				return fmt.Errorf("config: thrusters.%s.sizes.%s: mass and power must be >= 0", fk, sk)
			}
		}
	}
	return nil
}
