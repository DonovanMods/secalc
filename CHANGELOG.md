# Changelog

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](https://semver.org).

## [0.2.0] - 2026-08-19

### Added

- Space Engineers 1 support: embedded vanilla SE1 defaults (both grids,
  wiki-sourced) selected with `--game se1`; per-game user config files
  (`config-se2.toml`, `config-se1.toml`); `init` writes both (per-file
  skip, `--force` overwrites).
- Volumetric cargo model: containers may declare `capacity_l` (liters)
  converted through `settings.cargo_density` (kg/L); exactly one of
  `capacity`/`capacity_l` per container, validated at load.
- Game-aware report header (`secalc — SE1` / `secalc — SE2`).
- Makefile wrapping common development tasks (`make help` lists them);
  `make check` runs the formatting/vet/test gate.
- Named gravity presets from the config's `[gravity]` table:
  `-g palatine` resolves to 0.33 g. Ships verdure/palatine/caligo/space;
  numbers keep working; unknown names error listing the presets.

### Changed

- **Breaking:** project renamed `se2calc` → `secalc` (module path,
  binary, config directory `~/.config/secalc/`).
- **Breaking:** user config file split per game; a v0.1.x `config.toml`
  is ignored — rerun `secalc init` and re-apply your tweaks.
- Running with no arguments prints the full help (grammar summary and
  examples) and exits 0, instead of a bare arg-count error.

## [0.1.0] - 2026-08-19

### Added

- Mass expression input: literals (`1.23t`, `1,230kg`, bare kg) and
  config-defined storage shortcuts with multipliers (`2*s15m`, `2xs15m`).
- Thruster requirements per family and size as independent solutions,
  accounting for thruster self-mass, gravity multiplier (`-g`), and
  configurable thrust-to-weight margin (`-m`, default 1.5).
- Power/fuel totals in per-family units (MW, hydrogen units/s).
- `--full` to count containers as loaded (SE2 mass-based capacity).
- TOML config: embedded SE2 VS2.3 defaults, per-key overrides from
  `~/.config/se2calc/config.toml` and `--config`; `se2calc init`
  writes the defaults for editing (ion thrusters included commented
  out).
