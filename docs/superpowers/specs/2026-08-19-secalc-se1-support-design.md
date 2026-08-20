# secalc v0.2.0 — SE1 Support & Rename: Design

**Date:** 2026-08-19
**Status:** Approved design, pre-implementation
**Baseline:** se2calc v0.1.0 + post-release additions (Makefile, no-arg help,
gravity presets), main @ 74aae1a.

## Purpose

The calculation engine is game-agnostic; only the data is game-specific.
v0.2.0 makes that explicit: ship Space Engineers 1 defaults alongside SE2,
add the one physics model SE1 needs that SE2 lacks (volume-based cargo),
select games with a flag, and rename the project to `secalc` to match its
widened scope. Backwards compatibility with v0.1.x is explicitly NOT
maintained (pre-release tool, sole user consent given).

## Key Decisions

| Decision | Choice |
|---|---|
| Game differences live in | **Config only.** No `-V1`/game-mode flags in calculation code; the binary never knows which game it computes. The only game-aware code is default-config selection. |
| Volumetric cargo | Optional per-container `capacity_l` (liters) × `settings.cargo_density` (kg/L). SE2 configs keep `capacity` (kg). |
| `capacity` + `capacity_l` both set | Load-time validation error naming the container. Exactly one may be set; neither = holds nothing (existing behavior). |
| `capacity_l` without `cargo_density` | Load-time error naming the container and the missing setting. |
| Game selection | `--game se1\|se2` flag, default `se2`. Selects embedded defaults AND the user override file. Unknown id errors listing valid ids. |
| User config files | Symmetric per-game: `~/.config/secalc/config-se2.toml`, `config-se1.toml`. v0.1.x `config.toml` is ignored (no migration). |
| `init` | Writes BOTH files. Per-file protection: existing files reported and skipped; `--force` overwrites. |
| SE1 data scope | Vanilla only, both grids (small + large), from spaceengineers.wiki.gg with per-block source URLs. No DLC blocks. |
| Rename | `se2calc` → `secalc` everywhere current-facing: module path, binary, config dir, report header, README/CHANGELOG. Historical docs untouched. |
| Version | v0.2.0 (new features + breaking renames, pre-1.0). |

## Config Schema (delta from v0.1.x)

```toml
[settings]
margin = 1.5
cargo_density = 2.7   # kg/L — REQUIRED if any container uses capacity_l.
                      # SE1 file ships it (ore, wiki-verified); SE2 file
                      # omits it entirely.

[containers.lc]                    # SE1-style (volumetric)
name = "Large Cargo Container (large grid)"
mass = 2593                        # kg, empty (illustrative until research)
capacity_l = 421875                # liters

[containers.s15m]                  # SE2-style (mass-limited) — unchanged
name = "Cargo Container 1.5 m"
mass = 245.17
capacity = 16800                   # kg
```

Validation (all load-time, fail-fast, errors name file keys):
- exactly one of `capacity` / `capacity_l` per container (neither = 0 cargo);
- `capacity_l >= 0`;
- any `capacity_l` user requires `settings.cargo_density > 0`;
- existing rules (key shape, margin, gravity presets, thruster bounds) stand.

## Calculation Delta

Full-container cargo mass resolves per container:
`capacity` (kg) or `capacity_l × cargo_density`. Implemented as
`(Container).FullCargoKg(density float64) float64` in `config`, so `calc`'s
change is minimal (it needs the resolved density — `calc.Input` gains
`CargoDensity float64`, passed from `cfg.Settings.CargoDensity` by cmd).
Empty-mass path and everything downstream (thrust, power, output) unchanged.
No `--density` CLI flag (config-only knob).

## CLI Surface (delta)

```
secalc [flags] <expression>...
      --game string   which game's config stack to use: se2 (default) | se1
```

| | `--game se2` (default) | `--game se1` |
|---|---|---|
| Embedded defaults | `se2.toml` | `se1.toml` |
| User override file | `~/.config/secalc/config-se2.toml` | `config-se1.toml` |

`--config <path>` merges last, on top of the selected stack. Root
`Long`/`Example` gain an SE1 line (`secalc --game se1 -g moon 1t + 2*lc`).

`secalc init [--force]` writes both per-game files; output names each file
written or skipped.

## Config Package (delta)

- Embedded files: `se2.toml` (today's `default.toml`, renamed) + new
  `se1.toml`; exposed via `config.DefaultTOML(game string) ([]byte, error)`.
- `config.Load(game, overridePath string)`; `config.UserConfigPath(game
  string)`. Layering logic otherwise identical.
- `Config` gains nothing game-specific; `Container` gains `CapacityL`;
  `Settings` gains `CargoDensity`.

## Output (delta)

Report header becomes game-aware: `secalc — SE2` / `secalc — SE1`
(`calc.Plan` gains `Game string`, set by cmd from the flag). Everything else
renders identically.

## SE1 Default Data (research task)

Same discipline as the SE2 config: current live values from the
bot-maintained SE1 wiki (spaceengineers.wiki.gg), per-block source URLs
cited inline, nothing invented; values in this spec are illustrative until
research lands.

- **Thrusters:** atmospheric, hydrogen, ion; two families per type per grid
  (e.g. `atmospheric_lg` → "Atmospheric (large grid)"), small/large size
  variants within each (~6 families, ~12 entries). Thrust N, mass kg; power
  MW for electric families; hydrogen fuel in SE1's wiki-documented unit.
- **Containers:** small/medium/large × both grids where they exist,
  `capacity_l` at the x1 inventory multiplier (comment tells x3/x10 players
  to scale).
- **`cargo_density`:** ore density, wiki-verified (≈2.7 kg/L expected),
  with a comment explaining the "hauling ore" assumption.
- **Gravity presets:** all vanilla planets/moons (earthlike, mars, moon,
  europa, titan, alien, pertam, triton), wiki-verified.

## Rename Sweep

- Module path `github.com/DonovanMods/se2calc` → `github.com/DonovanMods/secalc`
  (go.mod + every import).
- Command/binary/version output → `secalc`; Makefile `BINARY`; `.gitignore`.
- Config dir → `~/.config/secalc/`.
- README/CHANGELOG renamed and updated; historical specs/plans/spikes stay
  untouched (they are records).
- **Last step, after everything else is green:** repo directory
  `~/Projects/apps/se2calc` → `~/Projects/apps/secalc`, Orca repo
  re-registration (add new path, remove stale record), fresh verification
  run from the new path. Sequenced last because the move breaks live
  session/tooling paths.

## Error Handling

Unchanged patterns: typed/contextual errors from config validation and the
new game-id check; cmd is the boundary; exit 1 via main.

## Testing

Existing suites cover the engine; new coverage targets the seams:
- **config:** both embedded defaults load + validate; capacity/capacity_l
  exclusivity error; missing-density error; per-game user file selection
  (se1 file must not affect se2 loads and vice versa); unknown game id.
- **calc:** volumetric full-mass table row (`capacity_l × density`).
- **output:** game-aware header row.
- **e2e:** SE1 run end-to-end (`--game se1 --full -g moon …`) with
  hand-derived expectations against a temp config; init writing both files
  incl. per-file `--force`; rename assertions (usage/version say `secalc`).
- TDD throughout; `make check` green at every commit.

## Out of Scope

- DLC blocks, SE1 modded blocks (user override territory).
- `--density` flag; per-cargo-type density tables.
- Migration of v0.1.x `config.toml`.
- Blueprint validation (tabled; see `docs/spikes/2026-08-19-blueprint-format.md`).
