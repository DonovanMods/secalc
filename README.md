# secalc

A Space Engineers thruster calculator — SE2 and SE1: given your ship's
mass, gravity, and storage, it reports how many thrusters of each type
lift the ship and what they consume.

## Install

    go install github.com/DonovanMods/secalc@latest

## Usage

    secalc [flags] <expression>

The expression is a sum of masses and storage shortcuts:

    secalc -g 0.5 1.23t + 2*s15m + s25m          # Space Engineers 2 (default)
    secalc --game se1 -g moon --full 1t + 2*lgl  # Space Engineers 1

Masses accept `kg` (default) and `t`, with optional comma separators
(`1,230kg`); `x` works as a multiplier (`2xs15m`). Storage shortcuts
come from the selected game's config: footprint-named for SE2 (`s15m`,
`s25m`, `s75m`), grid+size for SE1 (`sgs`/`sgm`/`sgl` small grid,
`lgs`/`lgl` large grid). Run `secalc shortcuts` (optionally with
`--game se1`) to list every storage code and gravity preset your config
defines.

Flags:

| Flag | Meaning |
|------|---------|
| `--game`        | config stack to use: `se2` (default) or `se1` |
| `-g, --gravity` | gravity multiplier relative to Earth 1 g (default 1), or a named preset like `-g palatine` (SE2) / `-g moon` (SE1) |
| `-f, --full`    | count containers as loaded |
| `-m, --margin`  | target thrust-to-weight ratio (default 1.5, from config) |
| `--config`      | extra config file merged on top of the selected stack |

Each output line is an independent solution: "Atmospheric (large grid)
Small: 3" means three of those thrusters alone would lift the ship
(with your margin), accounting for the thrusters' own mass. Thrust is treated as fully effective everywhere: SE1 atmospheric rows mean nothing on airless bodies, and SE1 ion thrusters actually run reduced in atmosphere — pick the rows that match your environment.

## Configuration

    secalc init

writes both games' defaults to `~/.config/secalc/` as
`config-se2.toml` and `config-se1.toml` (existing files are skipped;
`--force` overwrites). Your file only needs the keys you change.
Container keys are the CLI shortcuts (they must start with a letter);
the `[gravity]` table defines the named `-g` presets.

Games differ only by config. SE2 inventories are mass-limited
(`capacity` in kg); SE1 inventories are volumetric (`capacity_l` in
liters × `settings.cargo_density` kg/L, shipped as ore density 2.7027).
A container sets exactly one of the two.

## Game data

SE2 defaults: **VS2.3** values (July 2026) from
[spaceengineers2.wiki.gg](https://spaceengineers2.wiki.gg); SE1
defaults: current post-v1.203 vanilla values (both grids, no DLC) from
[spaceengineers.wiki.gg](https://spaceengineers.wiki.gg). Source URLs
are cited inline in each config. Known caveat: the SE1 large-grid Large
Cargo capacity is wiki-documented as 421,000 L, while the commonly
cited in-game figure is 421,875 L — override it after verifying in
game. When game patches change stats, override the affected keys.

SE2 model notes: mass-limited inventories, hydrogen in unitless
units/s, 1 g treated as 9.81 m/s² (community-verified). SE1 capacities
are at the x1 inventory multiplier — scale for x3/x10 worlds.

## Development

Common tasks are wrapped in the Makefile — `make` (or `make help`)
lists them. Run `make check` (formatting, vet, tests) before
committing; `make install` builds and installs the binary.
