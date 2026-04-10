# ROADMAP

## Legend

| Symbol | Meaning |
|--------|---------|
| 🔴 | Critical / Safety |
| 🟡 | High priority |
| 🟢 | Medium priority |
| 🔵 | Low priority / Exploratory |
| ✅ | Done |
| 🔲 | Planned |

---

## Phase 1 — Safety & Integrity (Critical)

### ✅ CSPlayerGameDataHash Recalculation 🔴
Implemented modified Adler-like checksum for the last 0x80 bytes of each save slot. Hash is recalculated automatically on every save.

**Implementation:** `backend/core/hash.go`
- 12 hash entries: Level, Stats (Int/Faith swapped), ArcheType, PGD+0xB8, padding, Souls, SoulMemory, EquippedWeapons (10 IDs), EquippedArmors (9 IDs), EquippedItems (16 IDs, lower 28 bits), EquippedSpells (14 IDs), padding
- Algorithm: `ComputeHashedValue()` with magic constant `0x80078071`
- `RecalculateSlotHash()` called in `SaveSlot.Write()` before return
- Tests: `backend/core/hash_test.go` (9 tests), round-trip tests updated to exclude hash regions

### ✅ Stat Consistency Validation 🔴
Class-aware validation of character stats with automatic level recalculation.

**Implementation:** `backend/vm/validation.go`, `backend/db/classes.go`
- Starting class data for all 10 classes (ID 0-9)
- Validates: attribute ≥ class base, bounds [1,99], level formula `sum(attrs) - 79`, level bounds [1,713]
- `ValidateStatsConsistency()` runs automatically on slot load
- `ClampToClassMinimums()` enforces class floors
- Tests: `backend/vm/validation_test.go` (9 tests), `backend/db/classes_test.go`

---

## Phase 2 — Event Flags & Quest System

### ✅ Event Flags Parser 🟡
Parse the EventFlags bitfield (~14.7 million flags at `EventFlagsOffset`, size `0x1BF99F` bytes). Implement read/write for individual flags by ID.

**Implementation:** `backend/db/db.go`, `backend/db/data/event_flags.go`, `backend/core/structures.go`
- `GetEventFlag(flags, id)` / `SetEventFlag(flags, id, value)` — lookup table + standard formula fallback
- ~840 flags in precomputed lookup table (`EventFlagInfo{Byte, Bit}`)
- `EventFlagsOffset` computed via full dynamic offset chain in `calculateDynamicOffsets()`
- `EventFlagsAvailable` exposed in `CharacterViewModel`
- PS4 caveat handled: `unlockedRegSz` bounded (max 1024), offset validated within `0x280000`

### 🔲 NPC Quest State Editor 🟡
Human-readable quest progression UI built on top of event flags. **Single most requested missing feature** across all Elden Ring editor communities.

**Technical details:**
- Map known event flag IDs to NPC questline steps (source: soulsmods.github.io/elden-ring-eventparam/)
- Show each NPC questline as step-by-step progression
- Allow advancing/reverting quest steps
- Support NPC revival (reset death flags)
- Cover both base game and DLC (Shadow of the Erdtree) questlines
- Community request: "Can I revive NPCs?" / "Can I reset quest progress?" — extremely common

### ✅ Boss Kill / Respawn Manager 🟡
Dedicated UI for toggling boss defeat states via event flags.

**Implementation:** `backend/db/data/bosses.go`, `backend/db/db.go`, `app.go`, `frontend/src/components/WorldProgressTab.tsx`
- ~120 bosses (base game + Shadow of the Erdtree DLC) with per-map defeat flag IDs
- `BossEntry` type with: id, name, region, type (main/field), remembrance, defeated state
- `GetBosses(slotIndex)` / `SetBossDefeated(slotIndex, bossID, defeated)` in `app.go`
- UI integrated into World Progress tab with sub-tabs (Sites of Grace / Bosses)
- Filter by type: all / main / field
- Kill All / Respawn All per region
- Remembrance boss indicator, main boss indicator
- Boss diff support in save comparison (`diffBosses`)

---

## Phase 3 — Sites of Grace & World State

### ✅ Sites of Grace Toggle 🟡
Unlock/lock individual Sites of Grace. Especially valuable on PS4 where no other tool offers this.

**Implementation:** `backend/db/data/graces.go`, `backend/db/db.go`, `app.go`, `frontend/src/components/WorldProgressTab.tsx`
- ~460 grace entries mapped to flag IDs in `data.Graces`
- `GraceEntry` type with: id, name, region, visited state
- `GetGraces(slotIndex)` / `SetGraceVisited(slotIndex, graceID, visited)` in `app.go`
- UI: region-grouped with expand/collapse, Unlock All per region, region map previews
- Grace diff support in save comparison (`diffGraces`)

### 🔲 Summoning Pools Toggle 🟢
Enable/disable summoning pool activation via event flags.

### 🔲 Colosseum Toggle 🟢
Unlock colosseums via their respective event flags.

### 🔲 Map Exploration Data Editing 🔵
Edit fog-of-war / map reveal data. **Fully unique feature** — no existing editor touches this.

**Technical details:**
- Map discovery flags in range 62000-63065
- Reset map exploration for fresh discovery feel on existing characters
- Full map reveal without walking everywhere

---

## Phase 4 — Inventory & Equipment Enhancements

### 🔲 Cookbook / Recipe Checklist 🟢
Visual grid of all cookbooks with unlock status. Cookbooks are inventory items with known IDs — straightforward to implement.

### 🔲 Great Rune Manager 🟢
Dedicated UI showing:
- Which Great Runes are obtained (inventory check)
- Which is currently equipped
- Rune Arc buff status (`GreatRuneOn` field at PlayerGameData+0xF7)

### 🔲 Gesture Unlock Checklist 🟢
Toggle grid for all 64 gestures.

**Technical details:**
- GestureGameData: `0x100` bytes (64 × u32 gesture IDs)
- Located at `StorageBoxOffset + storageSize` in dynamic offset chain

### 🔲 Spirit Ash Upgrade Level Editing 🟢
Edit upgrade levels (+0 to +10) for spirit ashes already in inventory.

### 🔲 Talisman Pouch Slots 🟢
Directly set the number of unlocked talisman slots (1-4).

**Technical details:**
- Field: `AdditionalTalismanSlotsCount` at PlayerGameData+0xBE (u8)

---

## Phase 5 — Character & World

### 🔲 NG+ Cycle Editor 🟢
Edit the current New Game+ cycle. alfizari's editor has this — we should reach parity.

### 🔲 Player Coordinates / Teleportation 🔵
Edit CSPlayerCoords section (0x3D bytes) — position, mapID, angle.

**Technical details:**
- CSPlayerCoords: coords (float×3), mapID (4 bytes), angle (float), + unknown coords/angle
- Located after CSWorldGeomMan sections in dynamic offset chain

### 🔲 Weather & Time of Day 🔵
Edit CSWorldAreaWeather (`AreaId`, `WeatherType` enum, `Timer` — 0x0C bytes) and CSWorldAreaTime (`Hour`, `Minute`, `Seconds` — 0x0C bytes).

### 🔲 DLC Progress Manager 🔵
Shadow of the Erdtree specific data:
- Scadutree Fragment count / blessing level
- Revered Spirit Ash upgrades
- DLC-specific grace points
- Miquella's Cross states
- CSDlc section: `0x32` bytes at `SlotSize - 0xB2`

---

## Phase 6 — Save Management & Safety

### 🔲 Save Corruption Detection / Repair 🟢
Expand beyond MD5 checksum recalculation.

**Technical details:**
- Validate dynamic offset chain integrity (offsets monotonically increasing, within slot bounds)
- Bounds-check `projSize` (max 256) and `unlockedRegSz` (max 1024) — especially PS4
- Verify BND4 entry table consistency (PC): entry sizes, data offsets, name table
- Detect common corruption patterns (zeroed magic, broken GaItem handles)

### 🔲 Save File Merging 🔵
Combine data from two different saves into one. **Fully unique** — no editor does this.
- Merge inventory from save A into save B
- Copy quest progress between saves
- Selective slot-level merge

### 🔲 Multiplayer Group Passwords 🔵
Edit the 5 group password slots stored in PlayerGameData (offset 0x124-0x17B, 5 × wchar[8]).

### 🔲 Achievement / Trophy Progress Viewer 🔵
Show which achievements are completable given current save state (e.g., "5/7 legendary armaments collected").

---

## Completed

### ✅ Phase 1 — Safety & Integrity
- CSPlayerGameDataHash recalculation (hash.go, hash_test.go)
- Stat consistency validation (validation.go, classes.go)

### ✅ Phase 2 — Event Flags & World State
- Event Flags Parser (db.go, event_flags.go, structures.go)
- Boss Kill / Respawn Manager (bosses.go, WorldProgressTab.tsx)

### ✅ Phase 3 — Sites of Grace
- Sites of Grace Toggle (graces.go, WorldProgressTab.tsx)

### ✅ Phase 22 — Item Descriptions & Stats
Display item flavor text and detailed stats in the item detail modal. Data sourced from ERDB (MIT-licensed, parsed from regulation.bin).

### ✅ Core Features (v0.2.0)
- Save file loading (PC + PS4)
- AES-128-CBC encryption/decryption
- MD5 checksum recalculation
- Character stats editing
- Inventory management (add/remove items)
- Item database browser with icons
- SteamID patching
- Bidirectional PC ↔ PS4 conversion
- Backup manager
- Cross-platform desktop app (Wails)

### ✅ Bugfixes
- Fix duplicate talismans in database (155 entries with `0xA0` prefix removed from `talismans.go`)
