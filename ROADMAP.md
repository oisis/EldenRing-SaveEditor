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

### 🔲 Event Flags Parser 🟡
Parse the EventFlags bitfield (~14.7 million flags at `EventFlagsOffset`, size `0x1BF99F` bytes). Implement read/write for individual flags by ID.

**Technical details (from audit):**
- Flag addressing: `byteIdx = flagID / 8`, `bitIdx = 7 - (flagID % 8)`
- `flag_set = data[EventFlagsOffset + byteIdx] & (1 << bitIdx)`
- EventFlags offset is computed via dynamic offset chain (see §10 of audit):
  ```
  gesturesOff    = StorageBoxOffset + 0x100
  unlockedRegSz  = read_u32(data[gesturesOff])     // DYNAMIC
  unlockedRegion = gesturesOff + unlockedRegSz*4 + 4
  horse          = unlockedRegion + 0x29
  bloodStain     = horse + 0x4C
  menuProfile    = bloodStain + 0x103C
  gaItemsOther   = menuProfile + 0x1B588
  tutorialData   = gaItemsOther + 0x40B
  IngameTimer    = tutorialData + 0x1A
  EventFlags     = IngameTimer + 0x1C0000
  ```
- PS4 caveat: `unlockedRegSz` can contain garbage — must be bounded (max 1024) and result validated within `0x280000`
- Terminated by single `0x00` byte
- Reference: [soulsmods.github.io/elden-ring-eventparam](https://soulsmods.github.io/elden-ring-eventparam/)

### 🔲 NPC Quest State Editor 🟡
Human-readable quest progression UI built on top of event flags. **Single most requested missing feature** across all Elden Ring editor communities.

**Technical details:**
- Map known event flag IDs to NPC questline steps (source: soulsmods.github.io/elden-ring-eventparam/)
- Show each NPC questline as step-by-step progression
- Allow advancing/reverting quest steps
- Support NPC revival (reset death flags)
- Cover both base game and DLC (Shadow of the Erdtree) questlines
- Community request: "Can I revive NPCs?" / "Can I reset quest progress?" — extremely common

### 🔲 Boss Kill / Respawn Manager 🟡
Dedicated UI for toggling boss defeat states via event flags.

**Technical details:**
- Boss defeat flags: IDs 9100-9135 (synchronized), 61100-61268 (per-boss)
- Each boss has 2 flags: defeat flag + reward flag
- Show boss names with toggle switches
- Include remembrance/reward status

---

## Phase 3 — Sites of Grace & World State

### 🔲 Sites of Grace Toggle 🟡
Unlock/lock individual Sites of Grace. Especially valuable on PS4 where no other tool offers this.

**Technical details:**
- Grace data stored in event flags (flag addressing as in Phase 2)
- Need mapping of grace names → flag IDs (from soulsmods reference)

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
