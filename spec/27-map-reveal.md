# 27 — Map Reveal (Visibility, Fast Travel, Cover Layer, Fog of War)

> **Scope:** how the editor makes the world map visible to the player.
> Replaces the older `27-fog-of-war.md` — FoW bitfield is one of four
> independent layers, exposed as a separate user action (`RemoveFogOfWar`)
> rather than as part of `RevealAllMap`.

---

## 1. Three-layer map model

The Elden Ring map is a stack of independent layers. Revealing the map
to the player requires touching layers 1 and 2 at minimum; layer 3 (FoW)
is cosmetic and layer 0 (regions) is for fast travel. Each layer is
controlled by a different mechanism in the save file.

| # | Layer | What it does | Storage | Editor function |
|---|-------|--------------|---------|------------------|
| 0 | Unlocked Regions | Fast-travel + region-state availability | Variable-length list of `u32` IDs at `gesturesOff` | `core.SetUnlockedRegions` (see §3) |
| 1 | Detailed Bitmap | Per-region colored map texture | Event flags `62xxx` (visible) + Map Fragment items in inventory | `App.SetMapRegionFlags`, `App.RevealAllMap` (see §2) |
| 2 | Cover Layer | Black tiles hiding undiscovered DLC areas | 8 floats inside BloodStain section | `revealDLCMap` (see `spec/29-dlc-black-tiles.md`) |
| 3 | Fog of War | Grey overlay above map texture | Dense bitfield in BloodStain→MenuProfile gap | `App.RemoveFogOfWar` (separate user action — see §4) |

**Implication:** `RevealAllMap` (the bulk action) touches layers 1 and 2
only — it sets visible event flags, adds the corresponding Map Fragment
items (so the in-game map UI shows the texture), and — for DLC regions —
overwrites the Cover Layer coordinates. The FoW bitfield is **not** part
of the "Reveal All" path; it is exposed separately as `RemoveFogOfWar` so
the user can opt in to cosmetic fog removal independently of map texture
reveal. See §4.

---

## 2. Layer 1 — Detailed Bitmap (event flags + fragments)

This is the **primary** mechanism the editor uses. Reference implementation:
`app.go::revealBaseMap` and `app.go::revealDLCMap`.

### 2.1 System flags

Four event flags control whether the map UI is even visible. Without them,
revealing region textures has no effect — the map screen stays empty.

| Flag ID | Name | When required |
|---------|------|---------------|
| 62000 | Allow Map Display | Base game — always |
| 62001 | Allow Underground Map Display | When revealing 62060–62064 |
| 82001 | Show Underground | When revealing 62060–62064 |
| 62002 | Allow Shadow Realm Map Display | When revealing 62080–62084 (DLC) |
| 82002 | Show Shadow Realm Map | When revealing 62080–62084 (DLC) |

Source: `backend/db/data/maps.go::MapSystem`.

### 2.2 Region visibility flags (62xxx)

Each map region has a single visible flag in the `62xxx` range. Setting
the flag uncovers the region's texture in the in-game map UI.

| Range | Area |
|-------|------|
| 62010–62012 | Limgrave |
| 62020–62022 | Liurnia |
| 62030–62032 | Altus Plateau |
| 62040–62041 | Caelid |
| 62050–62052 | Mountaintops / Snowfield |
| 62060–62064 | Underground |
| 62080–62084 | DLC (Shadow of the Erdtree) |
| 62100–62xxx | Dungeon-specific maps |

Full data and DLC range check: `backend/db/data/maps.go::MapVisible` /
`IsDLCMapFlag`. Sub-region flags that corrupt the cover layer when set
manually live in `MapUnsafe` and are excluded from "Reveal All".

### 2.3 Map Fragment items

Every overworld 62xxx flag has a paired inventory item (Map Fragment).
The flag uncovers the texture; the item is what the player picks up
in normal gameplay. We add both for consistency with the regular game flow.

Mapping: `backend/db/data/maps.go::MapFragmentItems` (24 entries — 19 base
game + 5 DLC). Item IDs span `0x40002198..0x400021AA` (base) and
`0x401EA618..0x401EA61C` (DLC).

### 2.4 Acquired flags (63xxx) — NOT used

`backend/db/data/maps.go::MapAcquired` documents 63xxx flags that match
each visible flag (offset = `visibleID + 1000`). These are **transient
notification flags** the game raises to show the "Map Fragment acquired"
popup, then clears. They have no effect on map visibility and are
deliberately not toggled by the editor. Leave them alone.

### 2.5 Algorithm — `RevealAllMap`

```
revealBaseMap(slot):
    flags = slot.Data[slot.EventFlagsOffset:]

    # Phase 1 — system + visible flags (no slot mutation, slice is safe)
    for id in MapSystem (excluding DLC system flags): SetEventFlag(flags, id, true)
    for id in MapVisible where !IsDLCMapFlag(id):
        SetEventFlag(flags, id, true)
        if id in MapFragmentItems: queue item add

    # Phase 2 — add fragment items (mutates slot length, flags slice now stale)
    for itemID in queue:
        AddItemsToSlot(slot, [itemID], qty=1, durability=0, isWeapon=false)

revealDLCMap(slot):
    # same phases for DLC flags + DLC fragments
    SetEventFlag(flags, 62002, true); SetEventFlag(flags, 82002, true)
    ...

    # Phase 3 — Cover Layer write (see spec/29)
```

**Order matters.** `AddItemsToSlot` shifts bytes inside the slot, which
invalidates any pre-computed slice into `slot.Data`. Set every flag
before adding any item, or recompute the flags slice between calls.

---

## 3. Layer 0 — Unlocked Regions (fast travel)

Variable-length list of `u32` region IDs at `gesturesOff` inside the slot.
Detailed format: `spec/11-regions.md`.

### 3.1 Effect

Each region ID corresponds to a geographic area. The game uses the list to:

- Enable fast travel between Sites of Grace inside the region.
- Track per-region state for invasions and multiplayer matchmaking.

**Region IDs do NOT remove FoW or reveal the map texture.** They are an
independent system — verified empirically (test 1 in §5).

### 3.2 ID ranges (selection)

| Range | Area |
|-------|------|
| 1001000–1001002 | Internal startup regions (purpose unknown) |
| 1800001 / 1800090 | Stranded Graveyard / Cave of Knowledge |
| 6100xxx | Limgrave overworld |
| 6102xxx | Weeping Peninsula |
| 6200xxx | Liurnia |
| 6300xxx | Altus Plateau |
| 6400xxx | Caelid / Dragonbarrow |
| 6500xxx | Mountaintops / Snowfield |
| 1xxxxx | Legacy dungeons (1000–1900 prefixes) |
| 3xxxxxx | Catacombs / caves / tunnels |

Full list: `backend/db/data/regions.go` (exposed via `db.GetAllRegions`).
Fresh save has 6 entries; "Unlock All" sets ~211 base-game region IDs.

### 3.3 Editing — use `core.SetUnlockedRegions`

```go
err := core.SetUnlockedRegions(slot, []uint32{...})
```

The function dedupes + sorts the IDs and rebuilds the affected portion
of the slot via `core.RebuildSlot` (full sequential serializer, see
`spec/30-slot-rebuild-research.md`). Zero risk of slot truncation:
`RebuildSlot` reaches end-of-data around byte ~2.2 MB, leaving 408–432 KB
of tail padding inside the 0x280000-byte slot. Tested up to ~100,000
regions in synthetic stress tests.

> **Historical note.** Earlier iterations of this spec (and the original
> Stage-2 implementation) inserted region IDs in place by shifting the
> rest of the slot, with a "max 10–20 regions" safety limit. That path
> was removed in R-1 Step 14 — `SetUnlockedRegions` is the only
> supported entry point now.

---

## 4. Layer 3 — Fog of War bitfield (`RemoveFogOfWar`)

A dense bitmask between BloodStain and MenuProfile representing per-tile
exploration state. The editor exposes a dedicated user action
(`App.RemoveFogOfWar`) that fills the entire range with `0xFF`. It is
**deliberately decoupled from `RevealAllMap`** — revealing map textures
and removing the fog overlay are conceptually different operations and
the user might want one without the other.

### 4.1 Location

```
afterRegs   = gesturesOff + 4 + regCount * 4
bitfield_start = afterRegs + 0x087E
bitfield_end   = afterRegs + 0x10B0      (inclusive last safe byte)
section_size   = 0x103C bytes total      (BloodStain → MenuProfile)
usable_range   = 2099 bytes (0x087E .. 0x10B0)
```

**Critical:** writing past `+0x10B0` overlaps `menuProfile` and
**crashes the game**. The 0x0000..0x087D prefix contains structured
horse + bloodstain data — also do not touch from this layer.

### 4.2 Format

Flat bitmask, LSB-first within each byte. `1` = tile revealed,
`0` = tile hidden. Tile-to-bit mapping is unknown and not derivable
from region IDs (see open questions in §6). One in-game teleport
flips ~356 bits in a contiguous 157-byte window.

### 4.3 Why it stays separate from `RevealAllMap`

- The Detailed Bitmap layer (§2) gives the player the *useful* signal
  — region textures, dungeon icons, fragment ownership. That is what
  most users mean by "show me the map".
- Filling the FoW bitfield is purely cosmetic — it removes the grey
  overlay the game uses to mark "you have not walked here yet". It does
  not unlock new content; the player can already use the map without it.
- Bundling it into `RevealAllMap` would commit users who only wanted
  fragments to also lose the natural exploration signal. Keeping it
  behind its own action preserves user choice.
- Selective per-region FoW removal would require reverse-engineering
  the bit-to-tile mapping. Not on the roadmap.

### 4.4 `RemoveFogOfWar` algorithm

```go
storageEnd  := slot.StorageBoxOffset + core.DynStorageBox
gesturesOff := storageEnd + core.DynStorageToGestures
regCount    := int(binary.LittleEndian.Uint32(slot.Data[gesturesOff:]))
afterRegs   := gesturesOff + 4 + regCount*4
for i := afterRegs + 0x087E; i <= afterRegs+0x10B0; i++ {
    slot.Data[i] = 0xFF
}
```

In-place overwrite, no byte shifting, no offset recalculation.
Reference: `app.go::RemoveFogOfWar`.

---

## 5. Verification log

Empirical results from the FoW research that informed §1's separation
into independent layers.

| # | Test | Result |
|---|------|--------|
| 1 | Add region_id only (no flag, no item) | Map texture unchanged (regions ≠ visibility) |
| 2 | 0xFF in bitfield (small range) + 1 region | Fog removed locally |
| 3 | 0xFF written past `+0x10B0` (overlaps menuProfile) | **Game crash** |
| 4 | 0xFF in full bitfield range, no region change | All fog removed (cosmetic only) |
| 5 | Insert 205 regions via byte-shift | **Game crash** (slot truncated) — fixed by switching to `RebuildSlot` |
| 6 | In-game teleport (Warmaster's Shack) | Adds region 6101000 + sets 356 bits |
| 7 | Set 62xxx visible flag without fragment item | Map texture revealed, but player has no UI hint |
| 8 | DLC visible flags only, no Cover Layer write | Texture appears but black tiles still cover DLC area |

### Test files (`tmp/save/`)

| File | Description |
|------|-------------|
| `ER0000.sl2` | Original save, full FoW, 6 regions |
| `ER0000-fow-before.sl2` | After editor (maps + graces added), FoW unchanged |
| `ER0000-from-deck.sl2` | After in-game play (1 teleport), local fog removed |
| `ER0000-no-fow-test.sl2` | Region + partial bitfield, fog removed locally |
| `ER0000-no-fow.sl2` | Full bitfield fill, all fog removed |

---

## 6. Open questions

1. **Bit-to-tile mapping** — which specific bits in the FoW bitfield
   correspond to which map tiles. Unknown; would need systematic
   single-area exploration diffs.
2. **Region IDs `1001000–1001002`** — appear in every fresh save but
   absent from every reference editor's region database. Likely
   internal startup markers.
3. **Structured prefix `+0x0800..+0x087D`** — repeating pattern
   `00 00 01 80 BF FF FF FF FF 00...`. Possibly per-tile coordinate
   anchors used by the game's render path. Do not overwrite.

---

## 7. References

- `backend/db/data/maps.go` — `MapSystem`, `MapVisible`, `MapUnsafe`,
  `MapFragmentItems`, `MapAcquired`, `IsDLCMapFlag`.
- `backend/db/data/regions.go` — full unlocked-region database.
- `app.go::SetMapRegionFlags`, `SetMapFlag`, `RevealAllMap`,
  `revealBaseMap`, `revealDLCMap`, `RemoveFogOfWar`,
  `ResetMapExploration` — Wails-exposed map editing API.
- `backend/core/writer.go::SetUnlockedRegions` — region list editor.
- `backend/core/slot_rebuild.go::RebuildSlot` — full-slot serializer.
- `spec/11-regions.md` — region list binary format.
- `spec/29-dlc-black-tiles.md` — Cover Layer (Layer 2) coordinates.
- `spec/30-slot-rebuild-research.md` — slack analysis + the path from
  byte-shift to `RebuildSlot`.
