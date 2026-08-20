# Spike: SE2 blueprint file format (tabled)

**Date:** 2026-08-19
**Status:** Tabled — findings recorded for a possible future module.
**Question:** Can se2calc read an SE2 blueprint and grade its engineering
(actual thrust vs. mass, power/fuel) from the stats in our config?
**Answer:** Yes, feasibly — but only via a reverse-engineered, undocumented
format. Decision: park it until Keen documents the format or a dedicated
decoding module is worth the maintenance cost.

## Where blueprints live (Linux/Proton)

```
<steam-library>/steamapps/compatdata/1133870/pfx/drive_c/users/steamuser/
  AppData/Roaming/SpaceEngineers2/AppData/Blueprints/<name>/grid.json.vrb
```

`grid.json.vrb` is **not JSON** (legacy filename from a deprecated plain-JSON
era). Sibling files: `snapshot.vrb`, `icon.png`, `screenshots/`.

## The VR3B container format (as reverse-engineered, VS2.3 era)

- Magic `VR3B`, u32 version (=1), then a segment table. Observed entry shape
  (all little-endian): `{u64 offset, u64 hash, u64 compressedSize,
  u64 uncompressedSize, u32 codec}` with **codec 2 = raw brotli** (no frame
  magic — which is why zstd/lz4/gzip probing all failed).
- The sample file (18,149 bytes) held three brotli segments:
  1. **Main payload** (starts ~0xC0 in the sample; 13,928 → 265,639 bytes):
     the binary VRAGE "DCS" serialization of every block instance.
  2. **Assembly manifest** (69 → 108 bytes): "Game2", "VRage",
     "System.Runtime".
  3. **Type table** (3,992 → 22,008 bytes): UTF-16LE type names
     (`…CubeBlocks.Movement.ThrusterObjectBuilder`,
     `…Items.InventoryObjectBuilder`, cockpit/LCD/weapon types, …).
- Segment sizes/offsets in the header sum exactly to the file size.

## What the payload contains (the useful part)

UTF-16LE strings appear **once per placed block instance**, pairing a
composition name with a definition GUID. From the "Space Explorer" sample:

| Composition string | Count | Reading |
|---|---|---|
| `ArmorCubeLight50_ServerComposition` | 149 | 0.5 m light armor cubes |
| `HydrogenThruster50_ServerComposition` | 30 | 0.5 m hydrogen thrusters |
| `AtmosphericThruster250_ServerComposition` | 3 | 2.5 m-class atmo thrusters |
| `Gyroscope50_ServerComposition` | 2 | gyros |
| `Battery150_ServerComposition` | 2 | batteries |
| `HydrogenTank500_ServerComposition` | 1 | hydrogen tank |

GUID occurrence counts corroborate the census (30× the hydro GUID, 3× the
atmo GUID). The grid display name also appears in the payload. Numeric
suffixes are centimeters (50 = 0.5 m, 250 = 2.5 m); note the internal names
do not match the wiki's display sizes one-for-one (wiki lists atmo 1/2/5/10 m
but the internal name here says 250) — a future mapping table must be
verified block by block, not assumed.

## Decode recipe (reference)

```python
import brotli, struct
data = open("grid.json.vrb", "rb").read()
assert data[:4] == b"VR3B"
# Sample-file segment coordinates; a real parser must walk the header table:
main = brotli.decompress(data[0xC0:14088])       # entity graph, 265,639 B
manifest = brotli.decompress(data[14088:14157])  # 108 B
types = brotli.decompress(data[14157:])          # 22,008 B, UTF-16LE names
```

## What a `se2calc validate <blueprint>` would take

**Pragmatic v1 (~2 tasks):**
1. VR3B reader in Go: header walk + brotli (`github.com/andybalholm/brotli`,
   pure Go — no stdlib brotli exists) + UTF-16 composition-string census.
2. Config table mapping composition names → existing thruster/container
   stats (+ per-block masses), then reuse `calc` as-is: TWR achieved at a
   given `-g`, power/fuel budget, verdict.

**v1 limitations (must be stated in output, not hidden):**
- Orientation ignored — lift needs downward-facing thrust; directions live
  in binary quaternions v1 would not decode. Total-thrust grading is an
  upper bound.
- Total ship mass needs per-block masses for every block type; our config
  only carries thrusters/containers. The wiki's bot-maintained block pages
  all list mass — a one-off scripted scrape could build the table. Until
  then: "known mass + N blocks unaccounted".
- Fragile: undocumented early-access format; any game update may break it.
  The parser must fail loudly on unknown header shapes, never guess.

**v2 (full DCS deserialization for per-axis TWR): not recommended** while the
format churns; wait for modding docs or a stabilized format.

## Recommended shape when resumed

A separate module/package (e.g. `internal/blueprint` or its own repo if the
decoder grows) owning: VR3B parsing, the composition→stats mapping data, and
the census. se2calc's `calc`/`output` stay untouched consumers.

## Sources

- Official (SE1→SE2 only, no format docs):
  https://2.spaceengineersgame.com/experienced-players/grid-exporter/
- Plain `grid.json` deprecation:
  https://support.keenswh.com/spaceengineers2/pc/topic/51578-cant-update-blueprints
- Everything else: direct inspection of a local blueprint (read-only), VS2.3
  era, 2026-08-19. No community parser for VR3B was found at spike time.
