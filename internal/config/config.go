// Package config loads se2calc settings: embedded defaults merged with the
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

	"github.com/spf13/viper"
)

// containerKeyRe is the lowercase image of parse's shortcut grammar
// (nameRe = ^[A-Za-z][A-Za-z0-9_-]*$); Viper lowercases TOML keys before
// validate() sees them, so this is what a container key must match to ever
// be reachable as a CLI shortcut.
var containerKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// DefaultTOML is the embedded default configuration, written verbatim by
// `se2calc init`.
//
//go:embed default.toml
var DefaultTOML []byte

// Settings holds calculator-wide tunables.
type Settings struct {
	Margin float64 `mapstructure:"margin"` // target thrust-to-weight ratio
}

// Container is one storage block; the map key in Config.Containers is its
// CLI shortcut.
type Container struct {
	Name     string  `mapstructure:"name"`
	Mass     float64 `mapstructure:"mass"`     // kg, empty
	Capacity float64 `mapstructure:"capacity"` // kg of cargo when full
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
}

// Load returns the embedded defaults merged with the user config file (if
// it exists) and then overridePath (if non-empty). Merging is per-key, so
// override files need only the values that differ. A non-empty
// overridePath that cannot be read is an error.
func Load(overridePath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("toml")
	if err := v.ReadConfig(bytes.NewReader(DefaultTOML)); err != nil {
		return nil, fmt.Errorf("loading embedded defaults: %w", err)
	}

	if userPath, err := UserConfigPath(); err == nil {
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

// UserConfigPath returns the user config file location,
// e.g. ~/.config/se2calc/config.toml (os.UserConfigDir respects
// XDG_CONFIG_HOME).
func UserConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating user config dir: %w", err)
	}
	return filepath.Join(dir, "se2calc", "config.toml"), nil
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
	}
	if len(c.Thrusters) == 0 {
		return fmt.Errorf("config: no thruster families defined")
	}
	for fk, fam := range c.Thrusters {
		if fam.Name == "" || fam.PowerUnit == "" {
			return fmt.Errorf("config: thrusters.%s: name and power_unit are required", fk)
		}
		if len(fam.Sizes) == 0 {
			return fmt.Errorf("config: thrusters.%s: at least one size is required", fk)
		}
		for sk, s := range fam.Sizes {
			if s.Name == "" {
				return fmt.Errorf("config: thrusters.%s.sizes.%s: name is required", fk, sk)
			}
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
