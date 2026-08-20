# se2calc

A Space Engineers 2 calculator: given your ship's mass, gravity, and
storage, it reports how many thrusters of each type lift the ship — and
what they consume.

## Install

    go install github.com/DonovanMods/se2calc@latest

## Usage

    se2calc [flags] <expression>

The expression is a sum of masses and storage shortcuts:

    se2calc -g 0.5 1.23t + 2*s15m + s25m

reads as: gravity 0.5 g, ship of 1.23 tonnes, plus two 1.5 m cargo
containers (`s15m`) and one 2.5 m container (`s25m`) — shortcuts are
named after the block footprint and matched case-insensitively. Masses
accept `kg` (default) and `t`, with optional comma separators
(`1,230kg`). `x` works as a multiplier too (`2xs15m`).

Flags:

| Flag | Meaning |
|------|---------|
| `-g, --gravity` | gravity multiplier relative to Earth 1 g (default 1), or a named preset like `-g palatine` |
| `-f, --full`    | count containers as loaded (empty mass + capacity) |
| `-m, --margin`  | target thrust-to-weight ratio (default 1.5, from config) |
| `--config`      | alternate config file |

Each output line is an independent solution: "Atmospheric 2 m: 3" means
three 2 m atmospheric thrusters alone would lift the ship (with your
margin), accounting for the thrusters' own mass.

## Configuration

    se2calc init

writes the built-in defaults to `~/.config/se2calc/config.toml`. Your
file only needs the keys you change — everything else falls back to the
defaults. Add containers (the table key is the CLI shortcut, and must
start with a letter since the expression grammar reads digit-leading
tokens as masses), tweak stats after a game patch, or uncomment the ion
thruster block for vacuum-world hauling. The `[gravity]` table defines
the named `-g` presets (verdure, palatine, caligo, space ship as
defaults; kemik/byblos are stubbed out until the wiki documents their
gravity — fill them in from your in-game map screen).

## Game data

Defaults are Space Engineers 2 **VS2.3** values (July 2026) from
[spaceengineers2.wiki.gg](https://spaceengineers2.wiki.gg) bot-extracted
block pages; source URLs are cited inline in the config. SE2 is in
early access — stats change. When they do, override the affected keys
in your config (and please open an issue).

Notes on SE2's model: inventories are **mass-limited** (capacity is kg,
there is no volume/density), hydrogen consumption is in the game's
unitless hydrogen units per second, and 1 g is treated as 9.81 m/s²
(community-verified).

## Development

Common tasks are wrapped in the Makefile — `make` (or `make help`)
lists them. Run `make check` (formatting, vet, tests) before
committing; `make install` builds and installs the binary.
