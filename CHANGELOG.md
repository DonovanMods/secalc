# Changelog

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](https://semver.org).

## [Unreleased]

### Added

- Makefile wrapping common development tasks (`make help` lists them);
  `make check` runs the formatting/vet/test gate.

### Changed

- Running `se2calc` with no arguments now prints the full help (with a
  grammar summary and examples) and exits 0, instead of the bare
  "requires at least 1 arg(s)" error.

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
