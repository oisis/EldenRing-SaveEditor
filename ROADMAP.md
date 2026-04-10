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

### 🔲 CSPlayerGameDataHash Recalculation 🔴
Implement the modified Adler-like checksum that lives in the last 0x80 bytes of each save slot. Currently our editor does NOT recalculate this hash after editing stats/equipment — this is a detection vector.

**Details:**
- 12 hash entries covering: Level, Stats, ArcheType, Souls, SoulMemory, EquippedWeapons, EquippedArmors, EquippedItems, EquippedSpells
- Stats hash has Int and Faith **swapped** in hash order
- Algorithm uses `ComputeHashedValue()` with magic constant `0x80078071`
- Reference: ClayAmore/EldenRingSaveTemplate (010 Editor template)

### 🔲 Stat Consistency Validation 🔴
Validate that edited character data is mathematically consistent to avoid server-side detection:

- `Level = StartingLevel + sum(current_stats - base_class_stats)`
- No attribute below starting class minimum
- No attribute above 99, max level 713
- SoulMemory (total runes acquired) >= runes required for current level
- MatchmakingWeaponLvl consistent with inventory

---

## Phase 2 — Event Flags & Quest System

### 🔲 Event Flags Parser 🟡
Parse the EventFlags bitfield (~14.7 million flags at `EventFlagsOffset`, size 0x1BF99F bytes). Implement read/write for individual flags by ID.

**Flag addressing:**
```
byteIdx = flagID / 8
bitIdx  = 7 - (flagID % 8)
```

### 🔲 NPC Quest State Editor 🟡
Human-readable quest progression UI built on top of event flags. Most requested missing feature across all Elden Ring editor communities.

- Map known event flag IDs to NPC questline steps (source: soulsmods.github.io/elden-ring-eventparam/)
- Show each NPC questline as step-by-step progression
- Allow advancing/reverting quest steps
- Support NPC revival (reset death flags)
- Cover both base game and DLC (Shadow of the Erdtree) questlines

### 🔲 Boss Kill / Respawn Manager 🟡
Dedicated UI for toggling boss defeat states via event flags.

- Boss defeat flags: IDs 9100-9135 (synchronized), 61100-61268 (per-boss)
- Each boss has 2 flags: defeat flag + reward flag
- Show boss names with toggle switches
- Include remembrance/reward status

---

## Phase 3 — Sites of Grace & World State

### 🔲 Sites of Grace Toggle 🟡
Unlock/lock individual Sites of Grace. Especially valuable on PS4 where no other tool offers this.

### 🔲 Summoning Pools Toggle 🟢
Enable/disable summoning pool activation via event flags.

### 🔲 Colosseum Toggle 🟢
Unlock colosseums via their respective event flags.

### 🔲 Map Exploration Data Editing 🔵
Edit fog-of-war / map reveal data. Fully unique feature — no existing editor touches this.

- Reset map exploration for fresh discovery feel on existing characters
- Full map reveal without walking everywhere
- Requires reverse engineering the map discovery flag range (62000-63065)

---

## Phase 4 — Inventory & Equipment Enhancements

### 🔲 Cookbook / Recipe Checklist 🟢
Visual grid of all cookbooks with unlock status. Cookbooks are inventory items with known IDs — straightforward to implement.

### 🔲 Great Rune Manager 🟢
Dedicated UI showing:
- Which Great Runes are obtained (inventory check)
- Which is currently equipped
- Rune Arc buff status (GreatRuneOn field at PlayerGameData+0xF7)

### 🔲 Gesture Unlock Checklist 🟢
Toggle grid for all 64 gestures. Data at GestureGameData offset (0x100 bytes, 64 x u32 gesture IDs).

### 🔲 Spirit Ash Upgrade Level Editing 🟢
Edit upgrade levels (+0 to +10) for spirit ashes already in inventory.

### 🔲 Talisman Pouch Slots 🟢
Directly set the number of unlocked talisman slots (1-4). Field: `AdditionalTalismanSlotsCount` at PlayerGameData+0xBE.

---

## Phase 5 — Character & World

### 🔲 NG+ Cycle Editor 🟢
Edit the current New Game+ cycle. alfizari's editor has this — we should reach parity.

### 🔲 Player Coordinates / Teleportation 🔵
Edit CSPlayerCoords section (0x3D bytes) — position, mapID, angle. Teleportation without the in-game map.

### 🔲 Weather & Time of Day 🔵
Edit CSWorldAreaWeather (AreaId, WeatherType, Timer) and CSWorldAreaTime (Hour, Minute, Seconds).

### 🔲 DLC Progress Manager 🔵
Shadow of the Erdtree specific data:
- Scadutree Fragment count / blessing level
- Revered Spirit Ash upgrades
- DLC-specific grace points
- Miquella's Cross states
- CSDlc section (0x32 bytes at slot tail - 0xB2)

---

## Phase 6 — Save Management & Safety

### 🔲 Save Corruption Detection / Repair 🟢
Expand beyond MD5 checksum recalculation:
- Detect common corruption patterns
- Validate dynamic offset chain integrity
- Bounds-check projSize and unlockedRegSz (especially PS4)
- Verify BND4 entry table consistency

### 🔲 Save File Merging 🔵
Combine data from two different saves into one:
- Merge inventory from save A into save B
- Copy quest progress between saves
- Selective slot-level merge

### 🔲 Multiplayer Group Passwords 🔵
Edit the 5 group password slots stored in PlayerGameData (offset 0x124-0x17B, 5 x wchar[8]).

### 🔲 Achievement / Trophy Progress Viewer 🔵
Show which achievements are completable given current save state (e.g., "5/7 legendary armaments collected").

---

## Completed

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
