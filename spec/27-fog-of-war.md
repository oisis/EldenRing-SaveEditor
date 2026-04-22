# 27 — Fog of War (Map Exploration)

## Overview

Fog of War (FoW) in Elden Ring controls which areas of the map are hidden (greyed out) and which are revealed. It involves **two data structures**:

1. **Unlocked Regions list** — variable-length array of `u32` region IDs (fast travel, area unlock)
2. **Exploration Bitfield** — a dense bitmask where each bit represents a map tile (visual FoW)

### What is required to remove FoW?

| Action | FoW removed? | Notes |
|--------|-------------|-------|
| Region IDs only | NO | Tested: added 6101000, FoW unchanged |
| Bitfield 0xFF only (no region change) | **YES** | Tested: full map revealed with only 6 original regions |
| Region IDs + Bitfield 0xFF | YES | Tested: single region + partial bitfield fill |

**The exploration bitfield alone is sufficient to remove visual FoW.** Region IDs are used by the game engine for fast travel and area state, but are NOT required for the visual map reveal. The game populates both on teleportation, but for editor purposes only the bitfield matters.

---

## 1. Unlocked Regions

### Location in Slot

The region list is a variable-length section in the dynamic offset chain:

```
StorageBoxOffset
  + DynStorageBox (0x6010)        = storageEnd
  + DynStorageToGestures (0x100)  = gesturesOff
  [gesturesOff]                   = unlocked regions (variable length)
  + regCount*4 + 4                = afterRegs (horse section starts here)
```

### Format

```
Offset   Type    Description
0x00     u32     count — number of unlocked region IDs
0x04     u32[]   region_id[0..count-1] — array of region IDs (unsorted)
```

Total section size: `count * 4 + 4` bytes.

### Effect

Each `region_id` corresponds to a geographic area of the game world. Adding a region_id tells the game engine that the player has "discovered" this area. The game uses this to:
- Enable fast travel within the region
- Allow the exploration bitfield to affect FoW rendering in that region

### Region ID Ranges

| Range | Area Type | Examples |
|-------|-----------|---------|
| 1000000–1099xxx | Castles / Legacy Dungeons | Stormveil Castle, Leyndell |
| 1200000–1207xxx | Underground | Ainsel River, Siofra River, Nokron |
| 1300xxx | Crumbling Farum Azula | Dragon Temple, Maliketh |
| 1400xxx | Academy of Raya Lucaria | Debate Parlor, Grand Library |
| 1500xxx | Haligtree / Elphael | Canopy, Prayer Room, Malenia |
| 1600xxx | Volcano Manor | Temple of Eiglay, Rykard |
| 1800xxx | Tutorial / Startup | Stranded Graveyard, Cave of Knowledge |
| 1900xxx | Endgame | Fractured Marika, Elden Beast |
| 3000–3999xxx | Dungeons (Catacombs, Caves, Tunnels) | Stormfoot Catacombs, Murkwater Cave |
| 6100xxx | Limgrave overworld | The First Step, Stormhill, Seaside Ruins |
| 6102xxx | Weeping Peninsula | Castle Morne, Weeping Peninsula East |
| 6200xxx | Liurnia of the Lakes | Liurnia South, Caria Manor |
| 6300xxx | Altus Plateau | Altus Highway Junction, Mt. Gelmir |
| 6400xxx | Caelid / Dragonbarrow | Central Caelid, Bestial Sanctum |
| 6500xxx | Mountaintops / Snowfield | Zamor Ruins, Forbidden Lands |

A fresh character (starting area only) has 6 regions:
```
1001000, 1001001, 1001002   — startup regions (undocumented)
1800001                      — Stranded Graveyard
1800090                      — Cave of Knowledge
6100000                      — The First Step, Church of Elleh
```

Full region list: see `tmp/fow.json` (211 entries).

### Editing — Adding a Region

**CRITICAL**: This is a variable-length section. Inserting bytes shifts ALL subsequent data in the slot (horse, bloodstain, menu profile, GaItemData, tutorial, ingame timer, event flags). The slot size is fixed at `0x280000` — the last bytes before the hash region (last `0x80`) are pushed out.

Algorithm:

```
1. Read regCount at gesturesOff
2. Calculate insertOff = gesturesOff + 4 + regCount * 4
3. Shift slot.Data[insertOff .. SlotSize-0x80-4] forward by 4 bytes
4. Write new region_id (little-endian u32) at insertOff
5. Increment regCount at gesturesOff
```

After insertion, all dynamic offsets (GaItemDataOffset, IngameTimerOffset, EventFlagsOffset) increase by 4. The save loader recalculates these on next load.

---

## 2. Exploration Bitfield

### Location in Slot

The bitfield sits inside the section between BloodStain and MenuProfile in the dynamic chain:

```
afterRegs                                        (end of unlocked regions)
  + DynHorse (0x29)          = horse
  + DynBloodStain (0x4C)     = bloodStain       (afterRegs + 0x0075)
  + DynMenuProfile (0x103C)  = menuProfile       (afterRegs + 0x10B1)
```

The exploration bitfield occupies bytes within the BloodStain→MenuProfile gap:

```
Bitfield start:  afterRegs + 0x087E   (= bloodStain + 0x0809)
Bitfield end:    afterRegs + 0x10B0   (last byte before menuProfile)
Section size:    0x103C bytes (BloodStain→MenuProfile)
Usable range:    2099 bytes (0x087E .. 0x10B0)
```

**CRITICAL**: Do NOT write past `afterRegs + 0x10B0`. The menuProfile section starts at `afterRegs + 0x10B1` and overwriting it **crashes the game**.

### Format

The bitfield is a dense bitmask. Each bit represents one map tile:
- `1` = tile revealed (FoW removed)
- `0` = tile hidden (FoW active)

The bitfield is NOT aligned to region boundaries — it is a flat array covering the entire game map. Byte order is little-endian within each byte (bit 0 = LSB).

### Tile Density

From empirical measurement (single teleportation to Stormhill / Warmaster's Shack):
- **356 new bits** were set (356 tiles revealed)
- Changed bytes span from `+0x087E` to `+0x091B` (157 bytes)
- This corresponds to a small area around one Grace

A fresh character (starting area only) has ~1073 bits set. After one teleportation, ~1429 bits.

### Remove All FoW (Tested & Confirmed)

To remove FoW from the entire map, fill the exploration bitfield with `0xFF`:

```
Bitfield range:  afterRegs + 0x087E  to  afterRegs + 0x10B0
Total:           2099 bytes → set to 0xFF
```

This is an **in-place overwrite** — no byte shifting, no size change, no risk of data corruption.

**Do NOT overwrite:**
- Bytes before `+0x087E` — contain structured data (horse/bloodstain headers with patterns like `00 00 01 80 BF FF FF FF FF 00...`)
- Bytes after `+0x10B0` — menuProfile section starts at `+0x10B1`; overwriting it **crashes the game**

Region IDs do NOT need to be modified for visual FoW removal.

### Selective Approach (Remove FoW for One Region)

There is currently **no known mapping** from region_id to specific bitfield offsets/bits. The game engine determines which bits to set based on the player's position and the region geometry at runtime.

Selective reveal would require further reverse-engineering (comparing saves with different single-region reveals to identify which bits map to which tiles).

---

## 3. Implementation Algorithm

### Remove All FoW

```
func RemoveAllFoW(slot):
    // Step 1: Calculate offsets
    storageEnd  = slot.StorageBoxOffset + 0x6010
    gesturesOff = storageEnd + 0x100
    regCount    = readU32(slot.Data, gesturesOff)
    afterRegs   = gesturesOff + 4 + regCount * 4

    // Step 2: Fill exploration bitfield (in-place, no shifting)
    fowStart = afterRegs + 0x087E
    fowEnd   = afterRegs + 0x10B0
    memset(slot.Data[fowStart .. fowEnd], 0xFF)
```

No region insertion needed. No byte shifting. Safe in-place operation.

### Adding Region IDs (Optional — for fast travel)

If region IDs need to be added (e.g. to enable fast travel to unvisited areas), this requires byte insertion which is **dangerous for large counts**:

```
func AddRegion(slot, region_id):
    // Insert 4 bytes — shifts ALL data after insertOff
    insertOff = gesturesOff + 4 + regCount * 4
    shift(slot.Data[insertOff .. SlotSize-0x80], +4)
    writeU32(slot.Data, insertOff, region_id)
    writeU32(slot.Data, gesturesOff, regCount + 1)
```

**WARNING**: Each insertion pushes 4 bytes out of the slot end (before hash region). Inserting many regions (e.g. 205 × 4 = 820 bytes) **truncates critical data and crashes the game**. Safe limit: ~10-20 regions per edit session. For bulk region unlock, implement proper slot reserialization.

### Constants

| Name | Value | Description |
|------|-------|-------------|
| `DynStorageBox` | `0x6010` | Offset from StorageBoxOffset to storageEnd |
| `DynStorageToGestures` | `0x100` | Offset from storageEnd to gesturesOff |
| `DynHorse` | `0x29` | Offset from afterRegs to horse section |
| `DynBloodStain` | `0x4C` | Offset from horse to bloodStain section |
| `DynMenuProfile` | `0x103C` | Offset from bloodStain to menuProfile |
| `FoW Bitfield Start` | `afterRegs + 0x087E` | First byte of the exploration bitmask |
| `FoW Bitfield End` | `afterRegs + 0x10B0` | Last safe byte (before menuProfile) |
| `FoW Fill Size` | `2099 bytes` | Total bytes to fill with 0xFF |
| `Hash Region` | `SlotSize - 0x80` | Must not be modified (last 128 bytes) |

---

## 4. Verification

### Confirmed by Testing

| # | Test | Result |
|---|------|--------|
| 1 | Adding region_id only (no bitfield) | FoW NOT removed |
| 2 | Bitfield 0xFF (range +0x087E..+0x0960) + 1 region_id | FoW removed around Stormhill ✓ |
| 3 | Bitfield 0xFF (range +0x087E..+0x1FFA) — past menuProfile | **Game crash** (overwrites menuProfile/gaItemsOther) |
| 4 | Bitfield 0xFF (range +0x087E..+0x10B0) + map/grace flags, NO region change | **Full map revealed** ✓ |
| 5 | Insert 205 region IDs (820 byte shift) + bitfield | **Game crash** (truncated slot end data) |
| 6 | Game teleportation (Warmaster's Shack) | Adds region_id 6101000 + sets 356 bitfield bits |

**Key finding**: Test 4 proves that **only the bitfield is needed** for visual FoW removal. Region IDs are NOT required.

### Test Files

| File | Description |
|------|-------------|
| `tmp/save/ER0000.sl2` | Original save, full FoW, 6 regions |
| `tmp/save/ER0000-fow-before.sl2` | After editor (maps+graces added), FoW unchanged |
| `tmp/save/ER0000-from-deck.sl2` | After playing on Deck (1 teleportation), FoW removed around Stormhill |
| `tmp/save/ER0000-no-fow-test.sl2` | Test 2: region + partial bitfield, FoW removed ✓ |
| `tmp/save/ER0000-no-fow.sl2` | Test 4: full bitfield fill, all FoW removed ✓ |

### Diagnostic Scripts

| Script | Purpose |
|--------|---------|
| `tmp/diag/diag_compare_flags.go` | Compare event flags between two saves |
| `tmp/diag/diag_fow_hunt.go` | Full slot diff analysis for FoW hunting |
| `tmp/diag/diag_add_fow_region.go` | Add region_id + fill bitfield (standalone tool) |

---

## 5. Open Questions

1. **Bitfield-to-tile mapping**: Which specific bits correspond to which map tiles? Currently unknown — would need systematic testing (reveal one area at a time, diff the bitfield).
2. **DLC regions**: The region list from Rust source (211 entries) does not include Shadow of the Erdtree regions. DLC FoW may use additional region IDs and/or a separate bitfield range.
3. **Region_id 1001000–1001002**: Present in fresh saves but not in any reference editor database. Purpose unknown (possibly internal startup regions).
4. **Structured data at +0x0800..+0x087D**: Repeating pattern `00 00 01 80 BF FF FF FF FF 00...` — purpose unknown. Possibly coordinate/header data for FoW rendering. Do not overwrite.
5. **Safe bulk region insertion**: Inserting 200+ regions crashes the game (820 bytes truncated from slot end). A proper implementation would need full slot reserialization with size adjustment.
