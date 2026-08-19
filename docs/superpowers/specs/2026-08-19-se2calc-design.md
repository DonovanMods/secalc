# se2calc — Space Engineers 2 Thruster/Mass Calculator: Design

**Date:** 2026-08-19
**Status:** Approved design, pre-implementation

## Purpose

A CLI calculator for Space Engineers 2 ship builders. Given a starting hull
mass, a gravity environment, and a list of storage containers, it reports how
many thrusters of each type/size are needed to lift the fully loaded ship, and
what power (or fuel) those thrusters consume.

## Key Decisions

| Decision | Choice |
|---|---|
| Language | Go (latest stable) |
| CLI/config stack | Cobra + Viper |
| Config format | TOML, embedded defaults via `go:embed`, user override merged per-key |
| Thruster output semantics | Alternatives per size — each family×size row is an independent, complete solution |
| Thrust margin | Target TWR, configurable, default **1.5** (`-m/--margin` overrides config) |
| Thruster self-mass | Accounted for, closed-form solution |
| Full-container mass | `capacity (L) × cargo_density (kg/L)`, density configurable |
| Power output | Per-family `power_unit` (MW for electric, L/s H2 for hydrogen) |
| Game data | **SE2 early-access values only — never SE1.** Researched from current wiki/community sources during implementation; sources cited as comments in `default.toml` |

## CLI Surface

```
se2calc [flags] <expression...>

  -g, --gravity float   gravity multiplier relative to Earth (1 g), default 1.0
  -f, --full            use loaded container masses (capacity × cargo density)
  -m, --margin float    target thrust-to-weight ratio, overrides config
      --config path     alternate config file
      --version

se2calc init [--force]  write embedded default config to
                        ~/.config/se2calc/config.toml (respects $XDG_CONFIG_HOME);
                        refuses to overwrite an existing file without --force
```

## Input Grammar

All positional args are joined with spaces, then tokenized — so `2*S1`,
`2 * S1`, and `2 *S1` parse identically.

```
expression := term ("+" term)*
term       := [count multiplier] (mass | shortcut)
multiplier := "*" | "x"
mass       := number unit?        unit: kg (default) | t
count      := positive integer
shortcut   := config-defined container key, matched case-insensitively
```

- Numbers accept comma thousands-separators and decimals: `1,230kg`, `1.23t`.
- A bare number means kilograms.
- Terms may appear in any order: `se2calc -g 0.5 1.23t + 2*S1 + S2 + 500`.
- Unknown shortcut → error listing the shortcuts the active config defines.
- `x` is accepted as a multiplier alongside `*` as shell-globbing insurance.

## Configuration

**Layering (Viper):** embedded defaults → `~/.config/se2calc/config.toml`
(if present) → `--config` file (if given). Merge is per-key: an override file
may contain only the values that differ.

**Structure** (values illustrative until SE2 research lands):

```toml
[settings]
margin = 1.5          # default target thrust-to-weight ratio
cargo_density = 2.7   # kg/L, used by --full

# Storage containers: the table key IS the CLI shortcut (case-insensitive).
[containers.S1]
name = "Small Cargo Container"
mass = 150            # kg, empty
capacity = 1000       # L

# Thruster families are open-ended: adding [thrusters.ion] later is a pure
# config change. Sizes within a family are open-ended too.
[thrusters.atmospheric]
name = "Atmospheric"
power_unit = "MW"     # unit for this family's consumption values

[thrusters.atmospheric.small]
name = "Small Atmospheric Thruster"
thrust = 96           # kN
mass = 700            # kg
power = 0.6           # in power_unit, per thruster
```

Fixed unit conventions, documented as comments in the generated default
config: mass kg, capacity L, thrust kN, gravity in multiples of 9.81 m/s².

## Calculation Model

1. **Total ship mass** `M` = sum of expression terms. Container terms use
   `mass` (empty) or `mass + capacity × cargo_density` (`--full`).

2. **Required thruster count** per family×size, accounting for the thrusters'
   own mass, with `g_eff = 9.81 × gravity_multiplier`:

   ```
   n × thrust ≥ (M + n × thruster_mass) × g_eff × margin
   n = ceil( M × g_eff × margin / (thrust − thruster_mass × g_eff × margin) )
   ```

   Edge cases:
   - **Denominator ≤ 0** → that size cannot lift its own weight under the
     given gravity/margin; reported as "not viable", never a huge bogus count.
   - **Zero gravity** → no thrust required; special output message instead of
     zero-count rows.

3. **Power/fuel** per row: `n × power`, in the family's `power_unit`.

**Deliberate simplification:** config `thrust` is treated as fully effective;
the gravity multiplier is the only environmental input. Per-family efficiency
curves (e.g. atmospheric falloff) are a possible future config+calc addition.

## Output

```
SE2 Calculator
──────────────────────────────────────────
Gravity: 0.5 g          Target TWR: 1.5

Mass breakdown:
  base                      1.23 t
  2 × Small Cargo (full)    2.30 t
  1 × Medium Cargo (full)   4.65 t
──────────────────────────────────────────
Total ship mass:            8.18 t (full)

Thrusters needed to overcome gravity
(each line is a complete, independent solution):

  Atmospheric                    power
    Small      12               7.2 MW
    Medium      4               6.4 MW
    Large       1               4.5 MW

  Hydrogen                       fuel
    Small       9              14.1 L/s H2
    Medium      3              11.2 L/s H2
    Large       1               8.7 L/s H2
```

- Mass breakdown echoes every parsed term with its resolved mass; `(full)`
  markers appear only with `--full`.
- Masses print in `t` at ≥ 1 t, otherwise `kg`.
- Not-viable rows: `Small — not viable (cannot lift own weight at 1.5 g)`.
- `-g 0` replaces the thruster section with
  `Zero gravity — no thrust needed to hover.`
- Plain aligned text; no color or table dependencies.

## Architecture

```
se2calc/
├── main.go               thin: calls cmd.Execute()
├── cmd/
│   ├── root.go           flags, arg wiring, orchestration
│   └── init.go           `se2calc init` subcommand
└── internal/
    ├── config/           default.toml (go:embed) + Viper load/merge → typed Config
    ├── parse/            expression → []Term (pure, no I/O)
    ├── calc/             mass totals, thruster counts, power (pure math, no I/O)
    └── output/           renders a report to any io.Writer
```

Data flow: `cmd` → `config.Load` → `parse.Expression(args, shortcuts)` →
`calc.Plan` → `output.Render`. Viper is quarantined inside `config`; every
other package sees only a plain typed struct.

## Error Handling

Domain packages fail fast with specific error types (`ParseError` carrying the
offending token, `UnknownShortcutError` carrying the available shortcuts,
config validation errors naming file and key). `cmd` is the boundary that
renders them as friendly one-line messages with exit code 1. No error is
swallowed.

## Testing

TDD throughout; every feature starts with a failing test.

- **parse:** table-driven — units, comma separators, `*`/`x` multipliers,
  case-insensitive shortcuts, arg joining, all error cases.
- **calc:** table-driven — margin math, thruster self-mass feedback,
  not-viable denominator, zero-g, ceil rounding.
- **config:** embedded defaults load; per-key override merge; `init` writes
  and refuses overwrite without `--force`.
- **output:** rendered-string tests against `io.Writer` — empty vs `--full`,
  not-viable rows, zero-g message.
- **e2e:** root-command runs with a temp config, including the canonical
  example `se2calc -g 0.5 1.23t + 2*S1 + S2`.

## Project Conventions

- Semver from v0.1.0; `README.md` and `CHANGELOG.md` maintained as functional
  components change.
- Follows `~/.claude/DEV.md` and `GO.md` directives.

## Out of Scope (future ideas, not built now)

- Ion thrusters as shipped defaults (user can add via config today).
- Per-family environmental efficiency curves.
- Optimal mixed-thruster loadout suggestion.
- Named gravity presets (e.g. `-g moon`).
