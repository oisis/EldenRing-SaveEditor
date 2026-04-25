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

### ✅ CSPlayerGameDataHash — Preserved (NOT recomputed) 🔴
The last 0x80 bytes of each save slot are preserved verbatim from the original file. **Do not recompute.**

**Investigation:** `backend/core/hash.go`, `backend/core/structures.go`
- All reference editors (ER-Save-Editor/Rust, er-save-manager/Python, Final.py) treat this region as opaque — they copy it unchanged
- The game uses these bytes at runtime for equipment data, NOT just as an integrity hash
- Our `ComputeHashedValue()` / `bytesHash()` algorithm was wrong: produced 32-bit values, original values are 16-bit (e.g. 0x36DF). 11/12 entries mismatched on real save → EXCEPTION_ACCESS_VIOLATION crash
- Hash functions remain in `hash.go` for reference; `RecalculateSlotHash()` is NOT called from `Write()`
- Round-trip tests exclude hash region (bytes are unchanged by design)

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

### ✅ NPC Quest State Editor 🟡
Human-readable quest progression UI built on top of event flags. **Single most requested missing feature** across all Elden Ring editor communities.

**Implementation:** `backend/db/data/quests.go`, `backend/db/db.go`, `app.go`, `frontend/src/components/WorldProgressTab.tsx`
- 36 NPCs with step-by-step event flag mappings (5-40+ steps each)
- `GetQuestNPCs()` — sorted list of NPC names
- `GetQuestProgress(slotIndex, npcName)` — returns quest steps with current flag state
- `SetQuestStep(slotIndex, npcName, stepIndex)` — atomically sets all flags for a step
- `QuestNPC`, `QuestStep`, `QuestFlagState` types exported via Wails bindings
- UI: "NPC Quests" sub-tab in World Progress with NPC dropdown, step list with completion status (green/yellow/grey), optional location, expandable flag details (current vs target), "Set" button per step
- Undo supported via standard pushUndo mechanism

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

**Known issue — multi-flag boss defeat (needs rework):**
- Currently only sets 1 event flag (9xxx defeat flag) per boss — grants runes but boss remains alive in-game
- Proper kill/respawn requires setting multiple flags per boss (arena state, defeat, quest progression, grace activation, item drops, etc.)
- Reference data available in `tmp/repos/er-save-manager/src/er_save_manager/data/boss_data.py` (208 bosses with complete flag lists)
- Requires: new `EventFlags []uint32` field in `BossData`, re-keying map to arena state flag, iterating all flags in `SetBossDefeated()`, testing per-boss in-game

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

**Known issue:** After toggling a grace via event flag, it appears on the map but is NOT fully activated — the player must physically touch it in-game to activate fast travel. The event flag controls map visibility, not full activation. May require additional flags or a different mechanism.

**Known issue:** DLC (Shadow of the Erdtree) Sites of Grace and map regions are not yet supported. See DLC Progress Manager.

### 🐛 Summoning Pools Toggle — UI works, in-game effect missing 🟡
Enable/disable summoning pool (Martyr Effigy) activation via event flags.

**Implementation:** `backend/db/data/summoning_pools.go`, `backend/db/db.go`, `app.go`, `frontend/src/components/WorldTab.tsx`
- 165 summoning pools (base game + Shadow of the Erdtree DLC) mapped to event flag IDs
- Legacy dungeon pools (10000040+) use precomputed lookup table entries in `event_flags.go`
- Open-world pools (1037530040+) also covered via lookup table (275 entries)
- `SummoningPoolEntry` type with: id, name, region, activated state
- `GetSummoningPools(slotIndex)` / `SetSummoningPoolActivated(slotIndex, poolID, activated)` in `app.go`
- UI: region-grouped with expand/collapse, Activate All per region, global Activate All
- Integrated as "Summoning Pools" section in World → Exploration sub-tab

**Bug status (2026-04-25):** UI toggles correctly, no errors, but **toggled pools are not active in-game** (tested offline to avoid bans). All pools affected, not specific ones.

**Diagnostic checklist:**
- [x] Database covers all pool IDs (165 pools, more than `ClayAmore/ER-Save-Editor` reference of 162)
- [x] Lookup table `event_flags.go` includes pool IDs with byte/bit offsets bit-for-bit identical to `ER-Save-Editor`
- [x] BST resolver produces identical offsets (verified `1037530040`, `1051570840`, `1060440040`)
- [x] `SetEventFlag` flips the correct bit in `slot.Data[EventFlagsOffset:]` slice (backing array — modifications propagate)
- [x] `SaveSlot.Write()` does NOT overwrite event flag region (only writes level/stats/name/runes)
- [x] `SaveFile()` serializes `slot.Data` directly without rebuild from parsed structs

**Remaining hypotheses (next steps when revisiting):**
1. **Persistence test missing** — write integration test: `LoadSave → Set → SaveFile → LoadSave → Get` to verify the bit survives the round-trip. If it doesn't survive, look at `core/writer.go` or encryption pipeline.
2. **Game requires secondary state** — bit may be set in event_flags but game might also check:
   - `unlocked_regions` for the pool's map area (suggests dependency on Invasion Regions feature)
   - Trophy data section (`trophy_data` 52 bytes)
   - `world_area` / `gaitem_game` cross-references
3. **Hash region (`CSPlayerGameDataHash`, last 0x80 bytes of slot)** — currently preserved verbatim. Game may validate it against runtime state when DLC is installed (DLC-specific check?).
4. **PS4-specific** — PS4 saves are unencrypted, but PC SteamID-bound encryption may interact with our flag write.

**Action plan after Invasion Regions (`feature/invasion-regions`) merges:**
- Write `tests/event_flag_persistence_test.go` covering Set → Save → Load → Get round-trip
- If round-trip persists → investigate game-side requirements (compare with reference save where pools are activated)
- If round-trip fails → trace where the bit gets lost in the writer/encryption pipeline

### ✅ Invasion Regions Toggle 🟡
Unlock invasion / blue summon regions (per-slot `unlocked_regions` struct, 78 region IDs).

**Stage 1 — read-only ✅** (commit `90db34b`):
- 78 region entries ported from `er-save-manager/data/regions.py`
- Parser populates `SaveSlot.UnlockedRegions []uint32`
- UI: "Invasion Regions" accordion in World → Unlocks

**Stage 2 — write ✅** (R-1 full slot rebuild, see `PLAN-R1.md` and `CHANGELOG.md`):
- First attempt (shift-based in-place patching) corrupted saves; rolled back.
- Final implementation: `core.RebuildSlot` re-serializes every section after `unlocked_regions` from typed structs, zero-pads the tail to `SlotSize`. Each slot has 408–432 KB of zero tail padding that absorbs the size delta on both PS4 and PC.
- 19 new section types parsed/serialized; round-trip + mutation tests for each.
- `SetUnlockedRegions` / `BulkSetUnlockedRegions` Wails methods drive UI per-region toggle, per-area `+`/`−`, and global Unlock All / Lock All.
- Steam Deck verification: PS4 save with 380 regions added loaded correctly, characters intact.

### ✅ Colosseum Toggle 🟢
Unlock colosseums via their respective event flags.

**Implementation:** `backend/db/data/summoning_pools.go` (Colosseums map), `backend/db/db.go`, `app.go`, `frontend/src/components/WorldProgressTab.tsx`
- 3 colosseums: Limgrave (60360), Caelid (60350), Royal/Leyndell (60370)
- Flag IDs already in `event_flags.go` lookup table
- `ColosseumEntry` type with: id, name, region, unlocked state
- `GetColosseums(slotIndex)` / `SetColosseumUnlocked(slotIndex, colosseumID, unlocked)` in `app.go`
- UI: card grid with large toggles, global Unlock All button
- Integrated as "Colosseums" sub-tab in World Progress tab

**Known issue:** Toggling colosseum flags in the GUI has no visible effect in-game (tested offline). May only work in online mode, or may require additional flags. Needs verification in online multiplayer session.

### ✅ Map Exploration & Fog of War 🟡
Full map reveal with Fog of War removal. **Fully unique feature** — no existing editor touches FoW.

**Implementation:** `app.go`, `backend/db/data/maps.go`, `backend/db/db.go`, `frontend/src/components/WorldProgressTab.tsx`, `spec/27-fog-of-war.md`
- Map visibility (62xxx) + acquisition (63xxx) combined into single toggle per region
- System flags (62000, 62001, 82001, 82002) as top-level checkboxes
- `RemoveFogOfWar(slotIndex)` — fills FoW exploration bitfield (2099 bytes at `afterRegs+0x087E..+0x10B0`) with 0xFF
- Unsafe sub-region flags (62004-62009, 62053, 62065) separated into `MapUnsafe` — excluded from Reveal All
- FoW automatically removed on any map region toggle or Reveal All
- Brute-force POI ranges replaced with individual named flags
- Tested on Steam Deck: full map reveal + FoW removal confirmed working
- See `spec/27-fog-of-war.md` for full reverse-engineering documentation

### ✅ Grace Unlock with Boss Arena Filter 🟢
Global "Unlock All" for Sites of Grace with option to skip boss arena graces.

**Implementation:** `backend/db/db.go`, `frontend/src/components/WorldProgressTab.tsx`
- `IsBossArena` field on `GraceEntry` — detected from grace name (26 boss keywords)
- "Skip Boss Arenas" checkbox (default: on) filters boss graces from bulk unlock
- Boss arena graces marked with "B" indicator in UI

### ✅ Dungeon Entrance Door Auto-Unlock 🟢
Automatically open/close catacomb and hero's grave sealed entrance doors when toggling their Site of Grace.

**Implementation:** `backend/db/data/graces.go`, `app.go`
- `DoorFlag uint32` field on `GraceData` — overworld ObjAct event flag for the entrance door
- `SetGraceVisited()` automatically sets/clears `DoorFlag` alongside the grace flag
- Door flags reverse-engineered via binary diff of before/after save files (5 confirmed via RE, 14 via bruteforce scan)
- Flag format: `10{col}{row}{ObjAct}` where col/row = overworld tile m60 coordinates
- ObjAct offsets: catacombs use 8540 or 8600, hero's graves use 8620
- 22/25 dungeons have confirmed door flags; 2 reclassified as regular graces (no doors); 4 remain:
  - War-Dead Catacombs (requires Radahn defeat, access mechanism unclear)
  - Fog Rift / Scorpion River / Darklight Catacombs (DLC, m61 tiles, not yet supported)

### 🔲 Dungeon Door Flags — Missing Entries 🟢
RE remaining dungeon entrance door flags.

**Resolved (v0.4.1):**
- ✅ Giant-Conquering Hero's Grave — door flag `1050538620` (confirmed via binary diff of before/after saves)
- ✅ Giant's Mountaintop Catacombs — door flag `1050538600` (confirmed via binary diff)
- ✅ Consecrated Snowfield Catacombs — door flag `1050558540` (confirmed via binary diff)
- ✅ Hidden Path to the Haligtree — reclassified as regular grace (doors always open, passage to new area)
- ✅ Leyndell Catacombs — reclassified as regular grace (no sealed doors, accessed via sewers)

**TODO:**
- War-Dead Catacombs (m60_52_41) — requires Radahn boss defeat to access area; standard ObjAct scan inconclusive. May need boss defeat flag + door flag combo, or may not have a standard door at all. Cannot test until boss kill mechanism is fixed.
- DLC catacombs: Fog Rift (m61_47_46), Scorpion River (m61_44_46), Darklight (m61_51_43/52_43) — DLC not yet supported, m61 tile prefix, likely `20{col}{row}{ObjAct}` format

---

## Phase 4 — Inventory & Equipment Enhancements

### ✅ Cookbook / Recipe Checklist 🟢
Visual grid of all cookbooks with unlock status via event flags.

**Implementation:** `backend/db/data/cookbooks.go`, `backend/db/db.go`, `app.go`, `frontend/src/components/WorldProgressTab.tsx`
- ~70 cookbooks (base game + DLC) with event flag IDs for unlock state
- `CookbookEntry` type with: id, name, category, unlocked state
- `GetCookbooks(slotIndex)` / `SetCookbookUnlocked(slotIndex, cookbookID, unlocked)` in `app.go`
- UI: category-grouped with expand/collapse, Unlock All / Lock All buttons
- Integrated as "Cookbooks" sub-tab in World Progress tab

**Known issue:** Toggling a cookbook event flag unlocks crafting recipes, but the physical cookbook item does NOT appear in the player's inventory (Key Items). The game stores cookbook ownership in two places: event flag (recipes) and inventory item (Key Items list). Currently only the event flag is toggled — need to also add/remove the corresponding Key Item (cookbook item IDs TBD).

### ✅ Great Rune Manager 🟢
Equipped Great Rune selector + buff toggle.

**Implementation:** `backend/core/offset_defs.go`, `backend/core/structures.go`, `backend/vm/character_vm.go`, `frontend/src/components/GeneralTab.tsx`
- `GreatRuneOn` (PGD offset 0xF7, u8 bool) — read/write via MagicOffset-184
- `EquippedGreatRune` (EquippedItemsItemIds+0x28, u32) — item IDs: Godrick 0x40000053, Radahn 0x40000054, Morgott 0x40000055, Rykard 0x40000056, Malenia 0x40000057, Mohg 0x40000058
- `EquipItemsIDOffset` stored in SaveSlot for dynamic chain access
- UI: dropdown (None/6 Great Runes) + checkbox (Active/Inactive) in GeneralTab

### ✅ Gesture Unlock Checklist 🟢
Toggle grid for all 64 gestures.

**Implementation:** `backend/db/data/gestures.go`, `backend/db/db.go`, `app.go`, `frontend/src/components/WorldProgressTab.tsx`
- 57 gestures (base game + DLC) with body-type variant detection (even/odd IDs)
- GestureGameData: `0x100` bytes (64 × u32) at `StorageBoxOffset + DynStorageBox`
- Empty sentinel: `0xFFFFFFFE` (not 0)
- `DetectBodyTypeOffset()` — auto-detects body type A (odd) vs B (even) from existing gestures
- `GetGestures(slotIndex)` / `SetGestureUnlocked(slotIndex, gestureID, unlocked)` in `app.go`
- `BulkSetGesturesUnlocked()` — batch operation (single IPC call, single pushUndo)
- UI: flat grid with Unlock All / Lock All buttons
- Integrated as "Gestures" sub-tab in World Progress tab

**Known issue:** Toggling gestures in the binary array changes the save correctly, but gestures may not appear in-game. The game likely requires BOTH the binary array entry AND the corresponding event flag (range 60800–60849) to be set. Event flag mapping per gesture is not yet implemented — see spec/08-spells-gestures.md. Fixing this requires reverse-engineering the exact flag↔gesture mapping.

### 🔲 AoW Acquisition Flag — auto-mark Ash of War as collected 🟡
Adding an Ash of War via Item Database puts the AoW into the player's inventory but does NOT mark it as acquired in the world. The original drop (corpse / enemy / chest) still spawns the same AoW in-game, allowing duplicate pickup and corrupting expected progression flow.

**Investigation needed:**
- Each AoW in the game has a paired "acquired" event flag (similar pattern to map fragments / cookbooks)
- Reverse-engineer the flag↔AoW item ID mapping from `tmp/repos/er-save-manager` or by binary diff (save before/after picking up AoW in-game)
- On `AddItemsToCharacter` for an AoW item type, automatically set the corresponding acquisition flag
- Add reverse logic: removing AoW via UI should clear the flag (optional — may be noisy)

### 🔲 Bell Bearing Merchant Kill Flag — auto-mark merchant as killed 🟡
Adding a Bell Bearing via Item Database adds the item to inventory but the merchant NPC who originally drops it remains alive in-game. Player can re-kill the merchant for a duplicate bell bearing or trigger broken NPC dialogue. Twin Maiden Husks won't recognize the bell bearing as legitimately acquired.

**Investigation needed:**
- Map each bell bearing item ID → merchant NPC kill event flag (Nomadic Merchants, Imp statues, Hermit Merchants, etc.)
- Some bell bearings come from non-merchant sources (boss drops) — distinguish per-item
- On `AddItemsToCharacter` for a bell bearing, automatically set the merchant's death flag
- Twin Maiden Husks may also need the "bell bearing turned in" flag to expand their wares — verify in-game

**Reference data:** `tmp/repos/ER-Save-Editor/src/db/items.rs` may have the mapping; otherwise reverse-engineer via binary diff.

### 🔲 Spirit Ash Upgrade Level Editing 🟢
Edit upgrade levels (+0 to +10) for spirit ashes already in inventory.

### ✅ Talisman Pouch Slots 🟢
Edit number of unlocked talisman slots (0-3 additional, total 1-4).

**Implementation:** `backend/core/offset_defs.go`, `backend/core/structures.go`, `backend/vm/character_vm.go`, `frontend/src/components/GeneralTab.tsx`
- `AdditionalTalismanSlotsCount` at PGD offset 0xBE → MagicOffset-241 (u8, clamped 0-3)
- UI: number input with arrows in GeneralTab profile row

---

## Phase 5 — Character & World

### ✅ NG+ Cycle Editor 🟢
Edit New Game+ cycle (0-7) with automatic event flag synchronization.

**Implementation:** `backend/core/offset_defs.go`, `backend/core/structures.go`, `backend/vm/character_vm.go`, `app.go`, `frontend/src/components/GeneralTab.tsx`
- ClearCount (u32) at dynamic chain offset `horse + 0x44` (`DynClearCount = 0x44`)
- Event flags 50-57 synced on save: flag N = NG+N (clear all, set target)
- `ClearCountOffset` stored in SaveSlot for read/write
- UI: number input 0-7 in GeneralTab profile row

### ✅ Character Appearance Presets 🟢
Apply community-created character appearance presets (face, body, skin, cosmetics, gender, voice).
Write presets to in-game Mirror Favorites (CSMenuSystemSaveLoad in UserData10).

**Implementation:** `backend/core/offset_defs.go`, `backend/db/data/presets.go`, `backend/db/data/presets_generated.go`, `backend/db/data/hair_mapping.go`, `scripts/parse_presets.go`, `app.go`, `frontend/src/components/AppearanceTab.tsx`
- FaceData blob layout fully mapped (303 bytes): header, 8 model IDs, 64 face shape params, 7 body proportions, 91 skin/cosmetics bytes
- Hair model IDs use non-sequential lookup table (`hair_mapping.go`) — UI position ≠ PartsId
- Other male model IDs use PartsId = UI - 1 (bone structure, beard, eyebrow, eyelash, tattoo, eyepatch)
- Female model IDs skipped entirely (non-sequential mapping unknown for face/hair/eyebrow)
- VoiceType added to PlayerGameData (offset -245 from MagicOffset)
- 20 presets from eldensliders.com (parsed by `scripts/parse_presets.go` from `tmp/characters/characters.md`)
- Mirror Favorites: writes to CSMenuSystemSaveLoad safe slots (0, 10-14) to avoid ProfileSummary collision
- Favorites header: 0xFACE marker, 0x11D0 constant, body_type inverted (0=male, 1=female)
- FaceModel forced to 0 in Favorites (non-zero causes invisible body in Mirror preview)
- UI: checkbox selection, image zoom modal, Apply (1 preset) / Add to Mirror (N presets), Remove from Favorites
- Undo supported via standard pushUndo mechanism

**Known limitations / TODO:**
- ✅ Male hair mapping complete: all 37 styles (UI 1-37) confirmed via save slot analysis + Mirror Favorites preset extraction
- ✅ DLC hair positions (UI 32-37) confirmed
- Female model IDs (face, hair, eyebrow) not written — mapping is non-sequential, no lookup table yet
- FaceModel (bone structure) forced to 0 in Favorites — non-zero causes invisible body in Mirror
- Other model categories (beard, eyebrow, etc.) may also need lookup tables like hair — not yet verified beyond UI-1

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
Comprehensive slot diagnostics with corruption detection.

**Backend done:** `backend/core/diagnostics.go`, `app.go`
- `DiagnoseSaveCorruption(slot, index)` — runs all diagnostic checks on a slot
- `DiagnoseSlot(slotIndex)` / `DiagnoseAllSlots()` in `app.go` — exposed via Wails bindings
- `SlotDiagnostics` / `DiagnosticIssue` types with severity levels (info/warning/error)

**TODO — Phase 2 (requires more investigation):**
- Validate dynamic offset chain integrity (offsets monotonically increasing, within slot bounds)
- Bounds-check `projSize` (max 256) and `unlockedRegSz` (max 1024) — especially PS4
- Verify BND4 entry table consistency (PC): entry sizes, data offsets, name table
- Detect common corruption patterns (zeroed magic, broken GaItem handles)
- Frontend UI for diagnostics results display
- Auto-repair for recoverable issues

### 🔲 Save File Merging 🔵
Combine data from two different saves into one. **Fully unique** — no editor does this.
- Merge inventory from save A into save B
- Copy quest progress between saves
- Selective slot-level merge

### 🔲 Multiplayer Group Passwords 🔵
Edit the 5 group password slots stored in PlayerGameData (offset 0x124-0x17B, 5 × wchar[8]).

### 🔲 Achievement / Trophy Progress Viewer 🔵
Show which achievements are completable given current save state (e.g., "5/7 legendary armaments collected").

### 🔲 Full Slot Rebuild (R-1) 🟡
Implement full slot serialization instead of in-place patching, matching reference editors (ER-Save-Editor/Rust, er-save-manager/Python).

**Technical details:**
- Parse entire slot into structured data, modify structures, serialize ALL sections from scratch to 0x280000 buffer
- Eliminates risk of data shift/misalignment bugs (BUG-1 class issues)
- Requires full parsing and serialization of ~25 sequential sections
- Currently mitigated by GaItems region clean-fill (writeGaItem) and version-based scan limits
- Reference: ER-Save-Editor `save_slot.rs` write(), er-save-manager `slot_rebuild.py`

---

## Phase 7 — Remote Deploy & Game Control

### ✅ SSH Deploy Target Management 🟡
Configure remote machines (Steam Deck, gaming PC) as deploy targets. Settings stored in app config directory (`os.UserConfigDir()/EldenRing-SaveEditor/targets.json`), not in a config file.

**Technical details:**
- New package `backend/deploy/` — `Target` struct, `LoadTargets()`, `SaveTargets()`, `DeleteTarget()`
- Config path: macOS `~/Library/Application Support/`, Linux `~/.config/`, Windows `%APPDATA%`
- Target fields: `name`, `host`, `port`, `user`, `keyPath`, `savePath`, `gameStartCmd`, `gameStopCmd`
- Default `savePath`: `/home/deck/.local/share/Steam/steamapps/compatdata/1245620/pfx/drive_c/users/steamuser/AppData/Roaming/EldenRing/{STEAM_ID}/ER0000.sl2`
- Default `gameStartCmd`: `steam steam://rungameid/1245620`
- Default `gameStopCmd`: `pkill -TERM -f eldenring.exe`
- UI: new "Deploy" section in SettingsTab with target list, add/edit/delete forms
- App methods: `GetDeployTargets()`, `SaveDeployTarget(target)`, `DeleteDeployTarget(name)`

### ✅ SSH Connection & File Transfer 🟡
Upload and download save files to/from remote machines via SSH/SFTP.

**Technical details:**
- New package `backend/deploy/ssh.go` — `SSHManager` using `golang.org/x/crypto/ssh`
- Authentication: SSH key (preferred) or password fallback
- File transfer via SFTP (`github.com/pkg/sftp`) — more reliable than SCP
- `TestConnection(targetName)` — verify SSH connectivity, return host info
- `UploadSave(targetName)` — backup remote file (timestamped `.bkp`), upload current save, verify file size
- `DownloadSave(targetName)` — download remote save file, load into editor
- Verification: compare local/remote file sizes after transfer
- Remote backup: `cp "{remote}" "{remote}.{YYYYMMDD_HHMMSS}.bkp"` before overwrite
- App methods: `TestSSHConnection(name)`, `DeploySave(name)`, `DownloadRemoteSave(name)`
- UI: buttons in SettingsTab per target — Test, Upload, Download with status feedback
- Elden Ring Steam App ID: `1245620`

### ✅ Remote Game Launch & Stop 🟢
Start and stop Elden Ring on remote machines via SSH.

**Technical details:**
- `LaunchGame(targetName)` — execute `game_start_cmd` via SSH (default: `steam steam://rungameid/1245620`)
- `CloseGame(targetName)` — execute `game_stop_cmd` via SSH (default: `pkill -TERM -f eldenring.exe`)
- Per-target command overrides (stored in target config)
- SIGTERM for graceful shutdown (game saves before exit)
- App methods: `LaunchRemoteGame(name)`, `CloseRemoteGame(name)`
- UI: Launch / Close buttons per target with status indicators

### ✅ Deploy Workflow Integration 🟢
One-click workflow: Close Game → Upload Save → Launch Game.

**Technical details:**
- `DeployAndLaunch(targetName)` — sequential: close game (if running) → wait 3s → upload save → launch game
- Progress feedback via Wails events (step-by-step status updates to frontend)
- Error handling: abort on upload failure, show detailed error
- UI: single "Deploy & Launch" button combining all steps

---

## Completed

### ✅ Phase 1 — Safety & Integrity
- CSPlayerGameDataHash recalculation (hash.go, hash_test.go)
- Stat consistency validation (validation.go, classes.go)

### ✅ Phase 2 — Event Flags & World State
- Event Flags Parser (db.go, event_flags.go, structures.go)
- Boss Kill / Respawn Manager (bosses.go, WorldProgressTab.tsx)
- NPC Quest State Editor — 36 NPCs, step-by-step progression UI (quests.go, db.go, app.go, WorldProgressTab.tsx)

### ✅ Phase 3 — Sites of Grace & World State
- Sites of Grace Toggle (graces.go, WorldProgressTab.tsx)
- Grace Unlock All with Boss Arena filter (db.go, WorldProgressTab.tsx)
- Summoning Pools Toggle (summoning_pools.go, WorldProgressTab.tsx)
- Colosseum Toggle (summoning_pools.go, WorldProgressTab.tsx)
- Map Exploration & Fog of War removal (maps.go, app.go, spec/27-fog-of-war.md)
- Combined Map Visible + Acquired toggle, System flags as top-level checkboxes

### ✅ Phase 4 — Character Progression & Inventory
- Talisman Pouch Slots (offset_defs.go, structures.go, character_vm.go, GeneralTab.tsx)
- NG+ Cycle Editor with event flag sync (offset_defs.go, structures.go, app.go, GeneralTab.tsx)
- Great Rune Manager — equipped rune + buff toggle (offset_defs.go, structures.go, character_vm.go, GeneralTab.tsx)
- Cookbook / Recipe Checklist — event flag unlock toggle (cookbooks.go, db.go, app.go, WorldProgressTab.tsx)
- Gesture Unlock Checklist — 64-slot gesture toggle (gestures.go, db.go, app.go, WorldProgressTab.tsx)

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

---

## Phase 8 — UI/UX Redesign ("Elden Ring SaveForge")

Comprehensive UI/UX overhaul based on interactive mockup (`tmp/mockups/mockup-v2.html`).
Rebranding from "ER Save Editor" to **Elden Ring SaveForge** (logo "ER", title "SaveForge by OiSiSk").

### ✅ Theme System 🟡
3 color themes (Dark, Light, Elden Ring) via CSS Custom Properties (`data-theme` attribute switching).

**Implementation:** `frontend/src/style.css`
- CSS variables defined on `[data-theme="dark"|"light"|"golden"]` selectors
- Theme switcher in Settings (inline with SteamID)
- All components use `var(--bg)`, `var(--accent)`, etc. — no hardcoded colors
- `accent-color` for native range sliders
- Console contrast colors per theme (`--sf-console-bg`, `--sf-console-text`)

### ✅ Tab Consolidation (7 → 5) 🟡
Reduced tab count for better UX.

**Implementation:** `frontend/src/App.tsx`, new components
| New Tab | Merges | Components |
|---|---|---|
| **Character** | GeneralTab + AppearanceTab | `CharacterTab.tsx` — Profile (collapsible), Attributes (2-col sliders), Appearance Presets |
| **Inventory** | InventoryTab + DatabaseTab | Toggle between Inventory view and Item Database view with split detail panel (60/40) |
| **World** | WorldProgressTab (reorganized) | `WorldTab.tsx` — 3 sub-tabs: Exploration / Progress / Unlocks |
| **Tools** | CharacterImporter + placeholders | `ToolsTab.tsx` — Importer, Save Comparison, Diagnostics, Backup Manager |
| **Settings** | SettingsTab (simplified) | SteamID \| Theme inline, UI toggles, Deploy Targets |

### ✅ Reusable AccordionSection Component 🟢
**Implementation:** `frontend/src/components/AccordionSection.tsx`
- Arrow + title + progress bar + count + action buttons in header
- Collapsed summary (attribute pills, progress percentage)
- `headerRight` prop for always-visible content (e.g. RL level)
- `id` prop for localStorage persistence of open/closed state
- Nested accordion support with `column-count: 2` masonry layout
- All sections default collapsed

### ✅ World Tab Sub-tabs 🟡
**Implementation:** `frontend/src/components/WorldTab.tsx`
- **Exploration**: Map & FoW, Sites of Grace (2-col masonry), Summoning Pools (2-col masonry), Colosseums
- **Progress**: Bosses (2-col masonry), NPC Quests
- **Unlocks**: Gestures, Cookbooks, Bell Bearings, Whetblades (all with progress bars)
- MiniProgress bars on all inner region/category accordions

### ✅ QuakeConsole + ToastBar 🟢
**Implementation:** `frontend/src/components/ToastBar.tsx`, `frontend/src/lib/toast.ts`
- Toast bar: fixed bottom, 30% width, centered, 1 line — hidden when console open
- Quake console: toggle via backtick key or click, resizable (drag edges/corners), dimensions persisted
- Click outside to close, contrast backgrounds per theme
- All `toast.success/error/loading` redirected to console via `lib/toast.ts` wrapper — no popup toasts
- Session log with colored severity (INFO/WARN/ERROR)

### 🔲 Console Log Order & Auto-Scroll 🟢
Three related UX issues with the Quake console — log visibility is degraded for long-running operations.

**Bugs:**
1. **No auto-scroll** — when new logs arrive, the console does not scroll to show the latest entry. User has to manually scroll to bottom on every operation.
2. **Click-outside collapses console** — currently clicking anywhere outside the console (e.g. on the main UI to interact with the app) closes it. Should stay open until user explicitly toggles via backtick / close button — user wants the console persistent while working.
3. **Log order should be reversed** — newest log entry should appear at the **top**, not the bottom. Eliminates the need for auto-scroll entirely (latest is always visible) and aligns with how the user reads operation feedback.

**Files:** `frontend/src/components/QuakeConsole.tsx` (or wherever the console renders), `frontend/src/lib/toast.ts` (log buffer ordering).

**Acceptance:**
- New logs prepended to the visible list (newest on top)
- Console open state persists across UI clicks; only backtick / explicit close toggles it
- Verify scroll position is preserved when user has scrolled mid-list (don't jump on new log if user is reading older entries)

### ✅ Character Tab Enhancements 🟢
**Implementation:** `frontend/src/components/CharacterTab.tsx`
- Collapsible **Profile** — summary: `Name | RL XX | NG+X | Runes`, RL in header with primary color
- Collapsible **Attributes** — 2-column sliders, summary: `Vig XX | Min XX | End XX | ...`
- Memory Slots placeholder field in Profile
- Add Settings moved to Inventory tab (Item Database view)

### ✅ Inventory / Item Database Split View 🟢
**Implementation:** `frontend/src/App.tsx`, `frontend/src/components/ItemDetailPanel.tsx`
- Toggle button switches between Inventory view and Item Database view
- Database view: split layout — DB list (60%) + Item Detail panel (40%)
- `ItemDetailPanel` component: weapon/armor/spell stats, description, icon, item info
- Add Settings collapsible above database
- Sidebar: collapsible empty character slots

### ✅ Rebranding 🟢
- Logo: "ER" in primary-colored square
- Title: "SaveForge" with "by OiSiSk" in primary color
- Window title: "Elden Ring SaveForge by OiSiSk"
- Theme label: "Elden Ring" (internal key: `golden`)

### 🔲 Known Bugs (to investigate)
- **Spectral Steed Whistle duplicate**: Two entries visible in database — `0x400000B5` (correct, in `tools.go`) and possibly `0x40000082` (only in `descriptions.go`, no item definition). One has wrong icon. Need to verify which IDs appear in GUI and remove/hide the duplicate.
- **Boss Kill mechanism incomplete**: Toggling boss defeat flag grants runes but the boss still appears alive in-game. Requires multi-flag approach — see Boss Kill / Respawn Manager section above for details and reference data.

### ✅ Bugfixes
- Fix duplicate talismans in database (155 entries with `0xA0` prefix removed from `talismans.go`)
- Fix game crash (EXCEPTION_ACCESS_VIOLATION) after adding weapons to both inventory and storage:
  - `writeGaItem()`: after writing any GaItem record, fill the entire remaining pre-allocated empty slot region (InventoryEnd → gaLimit) with clean `00000000|FFFFFFFF` markers. Root cause: weapon records are 21B (21 mod 8 = 5), creating a 5-byte phase shift between the game's scanner grid and the original 8-byte empty-slot grid — the two grids never converge, so the game reads garbage handles (e.g. 0x00FFFFFF) and crashes. Previous fix (4-byte zero at InventoryEnd) was insufficient: the scanner advances 8B and hits misaligned data at InventoryEnd+8.
  - `scanGaItems()`: validate GaItem handle type prefix — unknown prefix (e.g. `0xFFFF0000`) treated as stop condition, not a valid item
  - `DatabaseTab.tsx`: when adding 1 non-stackable item to both inv and storage, use single `AddItemsToCharacter` call so both locations share the same GaItem handle (prevents duplicate GaItem records)
- Fix game crash (EXCEPTION_ACCESS_VIOLATION) after loading save with added weapons — wrong `Index` (listId) in EquipInventoryData:
  - `addToInventory()`: new inventory items were assigned `Index = emptyIdx` (array position, e.g. 26). The game uses `Index` to look up CSGaItemIns in an internal table — indices 0-432 are reserved for equipment slots, so index=26 returns 0xFFFFFFFF (uninitialized) → r14=0xFFFFFFFF → crash reading from 0xFFFFFFFFFFFFFFFF.
  - Fix: storage item count header at `StorageBoxOffset` (4 bytes) is now updated after each append — previously stayed at 0, so game read 0 storage items
- Fix game crash (EXCEPTION_ACCESS_VIOLATION) — `next_acquisition_sort_id` not parsed from save file (`structures.go`, `writer.go`, `offset_defs.go`):
  - `EquipInventoryData.Read()` was missing: (a) skip of 4-byte `key_count` header between common and key items, (b) read of trailing counters `next_equip_index` + `next_acquisition_sort_id` with their byte offsets for write-back
  - `addToInventory()` was using `maxExistingIndex + 2` (heuristic) instead of `next_acquisition_sort_id` from save — the heuristic produces wrong values when the game's actual stride is 1 (not 2) or when the counter has diverged
  - Fix: `Read()` now skips key_count header, reads and stores both trailing counters; `addToInventory()` uses `slot.Inventory.NextAcquisitionSortId` / `slot.Storage.NextAcquisitionSortId` as Index, then increments and writes the counter back to `slot.Data`
  - New constants: `StorageCommonCount=0x780`, `StorageKeyCount=0x80`, `InvKeyCountHeader=4`, offset constants for all trailing counters
- Fix per-slot SteamID read/write corrupting PlayerGameDataHash region:
  - `SlotSize-8` (0x27FFF8) falls inside the last 0x80 bytes (hash region), NOT the actual SteamID field
  - Per-slot SteamID is at a dynamic offset in the sequential parsing chain (after BaseVersion, before PS5Activity)
  - Fix: removed per-slot SteamID read/write; authoritative SteamID is in UserData10 (managed by `flushMetadata()`)
- Fix arrows/bolts incorrectly registered in GaItemData section:
  - Arrows have weapon-type handles (0x80xxxxxx) but belong in EquipProjectileData, not GaItemData
  - Fix: `upsertGaItemData()` now skips items identified by `db.IsArrowID()` — matches Rust reference (upsert_projectile_list)
- Fix GaItem scanner reading unbounded entries:
  - `scanGaItems()` now limits entries to 5118 (version ≤ 81) or 5120 (version > 81), matching reference editors
  - Added `SaveSlot.Version` field parsed from slot offset 0x00
- Fix undo not preserving inventory counter offsets:
  - Added `EquipInventoryData.Clone()` method that deep-copies unexported `nextEquipIndexOff`/`nextAcqSortIdOff` fields
  - `pushUndo`/`RevertSlot` now preserve `Version` and `GaItemDataOffset`
- Fix `ReadStorage` breaking on first empty handle — now uses `continue` instead of `break`, preventing data loss from sparse storage gaps
- Fix `ComputeSlotHash()` offset chain missing dynamic `projSize` (BUG-4):
  - Hash entries [7]-[10] now use the full dynamic offset chain matching `calculateDynamicOffsets()`
  - `projSize` is read from save data via `ReadDynamicSize()` at the correct position in the chain
- Fix `readQuickItemIDs()` reading from wrong offset (BUG-5):
  - Quick items start at `equipedItemsOff + 0x58` (after ChrAsmEquipment header), not at `equipedItemsOff`
  - Added `ChrAsmEquipmentSize` constant (0x58 = 22 × 4 bytes)
- Fix `upsertGaItemData()` always setting `reinforce_type = 0` (BUG-6):
  - Added `reinforceTypeFromItemID()` that extracts upgrade level from item ID (`itemID % 100`)
  - Weapons +10/+25 now have correct reinforce_type in GaItemData
- Fix `EquipInventoryData.Read()` and `ReadStorage()` silently ignoring errors (BUG-9):
  - Both functions now return `error` and propagate read failures
  - `mapInventory()` returns error, propagated through `SaveSlot.Read()`
- Remove `ComputeSHA256` dead code from `crypto.go` (BUG-10)
- Sanitize DLC entry flag on cross-platform conversion (R-9):
  - DLC byte[1] (Shadow of the Erdtree entry flag) zeroed on PS4↔PC conversion
  - Prevents infinite loading when target platform/account doesn't own DLC
  - Constants: `DlcSectionOffset`, `DlcSectionSize`, `DlcEntryFlagByte` in `offset_defs.go`
