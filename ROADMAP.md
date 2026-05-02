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

### ✅ FlushGaItems → RebuildSlotFull (DLC + Hash overwrite fix) 🔴
**Post-mortem:** user-reported `EXCEPTION_ACCESS_VIOLATION` at `eldenring.exe+0x1EB9989` after adding many weapons in v0.7.0.

**Root cause:** `FlushGaItems` (`backend/core/writer.go`) shifted slot.Data right by `delta` bytes whenever GaItems section grew (replacing 8 B empty slots with 21 B weapons / 16 B armor). The shift overwrote the last `delta` bytes of slot — DLC section (50 B at `SlotSize-0xB2`) and PlayerGameDataHash (128 B at `SlotSize-0x80`). Comment claimed those bytes are "trailing padding" — they aren't. When delta exceeded `0x132 = 306` bytes, DLC + Hash were catastrophically replaced with bytes from preceding sections (often ASCII resource names from EventFlags region — observed `Ride_Enemy_Attack3018_CMSG` written into DLC offset on user's debug save). Game read non-zero DLC entry flag → loaded DLC area state with stale prerequisites → NULL pointer deref → crash.

**Fix:** `AddItemsToSlot` Phase 2 now calls new `RebuildSlotFull` (full from-scratch typed rebuild) instead of `FlushGaItems`. Both `RebuildSlotFull` and existing `RebuildSlot` end with explicit verbatim copy of DLC + Hash regions from `slot.Data` to fixed end-of-slot positions — defense in depth against any future mutation path that trims tailRest. `FlushGaItems` deprecated (kept for backward compat).

**Files:** `backend/core/slot_rebuild.go` (new `RebuildSlotFull` + DLC/Hash patch in both rebuild paths), `backend/core/writer.go` (Phase 2 rewrite + GaMap snapshot/restore), `backend/core/structures.go` (extracted `parseFromData()` from `Read()` for re-parse after rebuild). Tests: `TestRebuildSlotFullIdentityPC/PS4`, `TestAddItemsPreservesDLCAndHash` regress the bug. See CHANGELOG entry for full details.

**Recovery:** saves edited with the broken build cannot be safely repaired (DLC/Hash bytes were overwritten and original values lost). Restore from backup made before heavy edits.

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

**Implementation:** `app.go`, `backend/db/data/maps.go`, `backend/db/db.go`, `frontend/src/components/WorldProgressTab.tsx`, `spec/27-map-reveal.md`
- Map visibility (62xxx) + acquisition (63xxx) combined into single toggle per region
- System flags (62000, 62001, 82001, 82002) as top-level checkboxes
- `RemoveFogOfWar(slotIndex)` — fills FoW exploration bitfield (2099 bytes at `afterRegs+0x087E..+0x10B0`) with 0xFF
- Unsafe sub-region flags (62004-62009, 62053, 62065) separated into `MapUnsafe` — excluded from Reveal All
- FoW automatically removed on any map region toggle or Reveal All
- Brute-force POI ranges replaced with individual named flags
- Tested on Steam Deck: full map reveal + FoW removal confirmed working
- See `spec/27-map-reveal.md` for full reverse-engineering documentation

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

**Inventory sync:** `BulkSetCookbooksUnlocked` adds/removes the matching Key Item alongside the event flag via `CookbookFlagToItemID`. Cookbooks are filtered out of the Item Database (`db.GetItemsByCategory` skips `IsCookbookItemID`) and rendered as `ReadOnly` in the Inventory tab (`vm/character_vm.go`).

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
- 57 gestures (51 base + 6 DLC). All vanilla IDs are odd; the previous "EvenID/OddID body-type variant" theory was wrong (cross-checked with er-save-manager/data/gestures.py).
- GestureGameData: `0x100` bytes (64 × u32) at `StorageBoxOffset + DynStorageBox`. Empty sentinel: `0xFFFFFFFE`.
- `GetGestures(slotIndex)` / `SetGestureUnlocked(slotIndex, gestureID, unlocked)` / `BulkSetGesturesUnlocked()` in `app.go`. No event flags involved (er-save-manager confirms only the binary array matters).
- Read path counts only canonical (odd) IDs as unlocked — legacy "even body-type B" garbage from older builds is intentionally ignored so the UI reflects what the game actually shows.
- Lock single / Lock All also clears the matching `(id-1)` even slot, freeing all 64 sentinel slots so a follow-up Unlock All never runs out of space.
- 6 ban-risk entries (`The Carian Oath`, `Fetal Position`, both pre-order Rings, `?GoodsName?`, the Ring of Miquella alt slot) are tagged with `Flags: ["cut_content"|"pre_order"|"dlc_duplicate", "ban_risk"]`. UI marks them with a ⚠ tooltip; **Unlock All skips anything with `ban_risk`** so a single click cannot add cut/pre-order content. Users can still toggle them individually if they truly own the relevant DLC.
- UI: flat grid with Unlock All / Lock All buttons in World → Unlocks → Gestures and World Progress → Gestures.

### ✅ AoW Acquisition Flag — auto-mark Ash of War as collected 🟡
Adding an Ash of War via Item Database now also sets the duplication event flag so the AoW is recognised as acquired by the world.

**Implementation:** `backend/db/data/ash_of_war_flags.go`, `app.go`
- `AshOfWarFlags` map (flagID → AoW name, 116 entries from `er-save-manager/event_flags_db.py`)
- `AoWItemToFlagID` map (item ID → flag ID, 116/116 coverage matching DB)
- `AddItemsToCharacter` calls `db.SetEventFlag(slot.Data[slot.EventFlagsOffset:], flagID, true)` whenever the added ID is in the map
- Removing an AoW via UI does **not** clear the flag (intentional — flag is "ever acquired", noise-free)

### ✅ Bell Bearing Acquisition + Inventory Sync — single source of truth 🟡
Bell Bearings are managed exclusively from **World → Unlocks → Bell Bearings**. The toggle now adds the matching key item to inventory on unlock and removes it from inventory + storage on lock, so the acquisition flag and the inventory item never drift apart.

**Implementation:** `backend/db/data/bell_bearing_flags.go`, `app.go`, `frontend/src/components/DatabaseTab.tsx`, `frontend/src/components/InventoryTab.tsx`
- `BellBearingItemToFlagID` (62 entries) — generated from name match between `BellBearings` (er-save-manager event_flags_db) and `key_items.go` BB entries; 59 exact matches + 3 manual aliases (Kalé/Kale, Spell-Machinist, String-seller). Cut-content `Nomadic [11]` (ban_risk) excluded.
- `BellBearingFlagToItemID` — auto-derived reverse map for the World tab toggle.
- `SetBellBearingUnlocked` / `BulkSetBellBearings` now call `syncBellBearingItem` (mirrors the Whetblade pattern): unlock → add 1 to inventory if absent; lock → remove from inventory **and** storage.
- `AddItemsToCharacter` keeps the AoW-style flag hook for defense-in-depth, but Bell Bearings are no longer reachable from there.
- `IsBellBearingItemID(id)` helper drives both `db.GetItemsByCategory` (BBs filtered out of the Item Database) and `vm/character_vm.go` `ReadOnly` (Inventory tab disables Remove / Select). Same backend-driven pattern as Cookbooks and Whetblades — no Wails getter or frontend Set required.
- Test: `bell_bearing_flags_test.go` verifies coverage and flag-existence.

### 🔲 Info-tab item ground drop — investigation paused 🟡
**Symptom:** Adding 1/0-cap items from the **Info tab** (Notes, About * tutorials, Letters, Maps, Cookbooks…) via the editor causes the world copy to drop on the ground when the player walks past the trigger location in-game (e.g. Crafting Kit purchase at Kalé spawns "Tworzenie przedmiotów" on the ground because the player already has it). Cosmetic clutter only — **no ban risk** (vanilla NG+ produces the same behaviour when the player carries the item across cycles).

**What we tried (2026-04-29, branch `feat/inventory-game-accurate-categories`):**
1. **`WorldPickupFlagID`** map (308 entries) — extracted `getItemFlagId` from `ItemLotParam_map` (cat=1) and `eventFlag_forStock` from `ShopLineupParam` (equipType=3) in regulation.bin. Hooked into `AddItemsToCharacter` to set the flag for the world copy. Result: **flag set correctly in save, item still drops in-game**. Confirmed by save diff (flag 550130 for About Item Crafting written via `db.SetEventFlag`, no effect on EMEVD spawn check).
2. **`AboutTutorialID`** map (1 entry, expandable) — discovered `TutorialDataChunk` block (0x408 bytes) at `slot.TutorialDataOffset`, between `GaitemGameData` and `event_flags`. Layout: `unk0x0 u16 | unk0x2 u16 | size u32 | count u32 | u32 IDs[count]`. Buying Crafting Kit appended ID `2010` to the list (verified by save diff). Pre-populating via `core.AppendTutorialID` (clean save edit confirmed: count `8 → 9`, surgical 13-byte change). Result: **list correctly modified in save, item still drops in-game**. Tutorial ID 2010 controls the popup text appearance, not the item-give EMEVD action.

**Conclusion:** The give/spawn action for Info-tab tutorial items is gated by a check we have not yet identified. Likely candidates:
- Hardcoded EMEVD instruction (`event/m??_??_??_??.emevd.dcx` inside `Data0.bdt`) that bypasses both `getItemFlagId` and `TutorialDataChunk`.
- A separate region-state bitset somewhere in slot.Data we have not located.
- A flag in the EMEVD-emitted "tutorialFlagId" range (710xxx, 720xxx) that we have not exhaustively tried.

**Next investigation steps (when resumed):**
- Extract `Data0.bdt` from Steam Deck (`~/.local/share/Steam/steamapps/common/ELDEN RING/Game/`), decrypt BHD with public RSA key, decompile `event/common.emevd.dcx` (and area-specific `event/m11_*.emevd.dcx` for Stranded Graveyard / Limgrave) using community tools.
- Search EMEVD for `give_item(9113, 1)` / `give_item(9135, 1)` patterns and identify the gating flag.
- Alternative: empirical save-diff matrix — for each About item, take BEFORE save → trigger EXACTLY one item in-game → diff and find the unique byte change outside `cs_net_data_chunks` noise (need a save where player is fully stationary to minimize noise).

**Files retained for resumption:**
- `backend/core/tutorial_data.go` (parser/writer for `TutorialDataChunk`) — works as designed, just doesn't gate the right thing.
- `backend/db/data/tutorial_ids.go` (`AboutTutorialID` map) — populate as future findings come in.
- `backend/db/data/world_pickup_flags.go` (`WorldPickupFlagID` map) — 308 entries; useful for items where the flag *does* gate the spawn (Notes that drop from world chests/lots, not from EMEVD scripts), so kept in the codebase.
- `app.go` `AddItemsToCharacter` hooks for both maps — harmless if the gating mechanism isn't triggered, beneficial when we identify it.

**Ban-risk assessment:** None. Setting a flag the game doesn't read is a no-op. Appending a tutorial ID is what the game itself does on first play. Items dropping on the ground is a vanilla pickup-cap interaction.

### 🔲 Bell Bearing Merchant Kill Flag — auto-mark merchant as killed 🟡
The acquisition flag (above) is enough for Twin Maiden Husks to expand wares, but the **merchant NPC who originally drops the BB remains alive in-game**. Player can re-kill the merchant for a duplicate, or trigger broken NPC dialogue.

**Investigation needed (future work):**
- Map each merchant BB item ID → merchant NPC kill event flag (Nomadic, Hermit, Imp, etc.)
- Some BBs come from non-merchant sources (boss drops, dead bodies) — distinguish per-item
- Likely overlaps with quest_flags_db.py "Picking up X's Bell Bearing" flag groups in `tmp/repos/er-save-manager`
- Once mapped: extend `AddItemsToCharacter` to set both acquisition flag and kill flag

### ✅ Spirit Ash Upgrade Level Editing 🟢
Pick the upgrade level (+0 to +10) when adding a spirit ash from the Item Database.

**Implementation:** `frontend/src/App.tsx`, `frontend/src/components/GeneralTab.tsx`, `frontend/src/components/DatabaseTab.tsx`, `app.go`
- Each upgrade tier is a distinct item ID in `backend/db/data/ashes.go` (`baseID + N` for `+N`); editing in-place would still rewrite the inventory record, so the simpler “pick on add” flow is what we ship
- Add Settings exposes an `upgradeAsh` slider (0-10), persisted per-character
- `AddItemsToCharacter` resolves `finalID = id + uint32(upgradeAsh)` for items with `category == "ashes"`

### ✅ Talisman Pouch Slots 🟢
Edit number of unlocked talisman slots (0-3 additional, total 1-4).

**Implementation:** `backend/core/offset_defs.go`, `backend/core/structures.go`, `backend/vm/character_vm.go`, `frontend/src/components/GeneralTab.tsx`
- `AdditionalTalismanSlotsCount` at PGD offset 0xBE → MagicOffset-241 (u8, clamped 0-3)
- UI: number input with arrows in GeneralTab profile row

### 🔲 Inventory Custom Order — Sort Modes + Drag & Drop Grid 🟡
Pozwolić graczowi wybrać sposób sortowania ekwipunku (Acquisition / Alphabetical / Item Type / Weight / Attack Power / Upgrade) i ułożyć itemy w **dowolnej kolejności drag & drop** w siatce w stylu in-game UI gry. Scope startowy: **bronie + zbroja + talizmany** (niestackable). Tarcze + AoW + Ashes — opcjonalnie w follow-upie.

**Mechanika (zweryfikowana w research, BEFORE coding wymaga Phase 0 in-game test):**
- W save jest pole `acquisition_index` (offset `0x08` w 12-bajtowym `InventoryItem`, u nas `core.InventoryItem.Index` w `structures.go:90`). To **globalny licznik** inkrementowany przy każdym pickup (`next_acquisition_sort_id` przy końcu sekcji — `structures.go:97`)
- **To pole kontroluje sortowanie "Acquisition Order"** w grze. Custom porządek można ustawić TYLKO przez manipulację `acquisition_index`
- Inne sortowania w grze (Item Type / Weight / Attack Power / Alphabetical) są **liczone runtime z `regulation.bin` params** (`EquipParamWeapon.sortGroupId`/`sortId`, `EquipParamProtector.sortGroupId`, `EquipParamAccessory.sortId`) — **NIE są w save'ie**, edytora ich nie zmieni
- Konsekwencja: nasz custom order jest widoczny tylko gdy gracz w grze ma sortowanie ustawione na "Acquisition Order" (default po fresh load — verified w `er-save-manager/parser/equipment.py:196,229`). Przełączenie w grze na inne sortowanie ignoruje nasze indeksy (gra liczy z params); powrót na Acquisition Order odzyska nasz porządek

**Pułapki:**
- **Reserved index range 0-432** (`InvEquipReservedMax`, `core/diagnostics.go:158`) — zarezerwowane dla equipment slotów (broń aktywna L/R, zbroja active, talizmany active, gestures, quick items). Custom order MUSI używać `Index >= 433` — najlepiej `base = max(NextAcquisitionSortId, 1000)` jako bufor
- **Reorder per kategoria** — gra pokazuje sub-zakładki (Tools / Melee / Shields / Head / Chest / Talismans...), sort jest per-tab. `sortGroupId` definiuje grupę. Ułożenie kolejności broni nie wpływa na talizmany — każda kategoria osobny zakres `acquisition_index`
- **Stackables** (consumables, materiały, AoW) — gra może je grupować inaczej (po `goodsType`); scope startowy zawęża się do niestackable equipment, gdzie mechanika `acquisition_index` jest zweryfikowana
- **Brak danych do innych sort modes** — `ItemData` nie ma obecnie pól `Weight` / `AttackPower` / `SortGroupId`. Trzeba zaimportować z `tmp/erdb/1.10.0/EquipParam*.csv` przez `scripts/import_erdb.go`

**Reference editors**: ani `er-save-manager` ani `ER-Save-Editor/Rust` nie mają takiego feature'u — edytują `acquisition_index` per row w spreadsheet view, brak drag&drop / grid view. Byłby to **unique feature** (jak map reveal / FoW / SSH deploy)

---

#### Phase 0 — Verify in-game (1-2h, KRYTYCZNE przed kodowaniem)

Test:
1. Save z 5 broniami w slotach
2. Hex edit `acquisition_index` 5 broni → 1000, 1001, 1002, 1003, 1004
3. Steam Deck deploy (memory feedback `reference_steam_deck_deploy.md`) → wczytaj save
4. W grze: przełącz sortowanie na Acquisition Order
5. Sprawdź czy bronie są w kolejności [1000..1004] ascending
6. Pickup nowego itemu w grze → sprawdź czy nasze 5 broni zachowuje porządek (nowy item na końcu jako `next_acquisition_sort_id`)

Jeśli nie potwierdza → **abort** całego feature'u (nasza hipoteza o `acquisition_index` jest błędna). Jeśli OK → kontynuuj.

---

#### Phase 1 — Backend reorder API + 2 sort modes (3-5h)

**Pliki**: `app.go`, nowy `tests/inventory_reorder_test.go`

```go
// Returns ordered handle list per category for current state.
func (a *App) GetInventoryOrder(charIdx int, category string) ([]uint32, error)

// Sets new acquisition_index per handle in given order.
// Indices: base, base+1, base+2... where base = max(NextAcquisitionSortId, 1000)
// Updates next_acquisition_sort_id counter on completion.
func (a *App) ReorderInventory(charIdx int, category string, orderedHandles []uint32) error

// Bulk sort by mode. Phase 1 supports: "acquisition" | "alphabetical".
// Phase 2 adds: "weight" | "attackPower" | "sortGroupId" | "upgradeLevel".
func (a *App) SortInventory(charIdx int, category string, sortMode string) error
```

**Kategorie (Phase 1 scope)**: `"melee_armaments"`, `"head"`, `"chest"`, `"arms"`, `"legs"`, `"talismans"` (mapping na `db.GetItemDataFuzzy(itemID).Category`).

**Implementacja `ReorderInventory`:**
1. `pushUndo(charIdx)`
2. Walidacja: każdy handle istnieje w `slot.Inventory.CommonItems`, należy do podanej kategorii
3. Zarezerwuj zakres: `base = max(slot.Inventory.NextAcquisitionSortId, 1000)` (bufor żeby nie kolidować z reserved range 0-432)
4. Per handle w `orderedHandles`: znajdź slot w `CommonItems`, ustaw `Index = base + i`, write-back przez `SlotAccessor.WriteU32` na `commonStart + slotIdx*12 + 8`
5. Update `slot.Inventory.NextAcquisitionSortId = base + len(orderedHandles)`, write-back na `nextAcqSortIdOff`

**Tests:**
- Reorder 5 broni → reload save → kolejność zgodna
- Reorder respektuje reserved range (`Index >= 433` zawsze)
- Mixing categories: reorder broni nie zmienia indeksów talizmanów
- `next_acquisition_sort_id` poprawnie zaktualizowany
- Round-trip preservation: `Save → Reorder → Write → Read → indices match`

---

#### Phase 2 — Import danych do pozostałych sort modes (2-3h)

**Pliki**: `backend/db/data/types.go` (rozszerzyć `ItemData`), `scripts/import_erdb.go` (rozszerzyć), `backend/db/data/{melee_armaments,head,chest,arms,legs,talismans}.go` (regenerate)

Pobrać z `tmp/erdb/1.10.0/EquipParam*.csv`:
- `weight` (broń + zbroja) → `ItemData.Weight float32`
- `attackBasePhysics` (broń) → `ItemData.AttackPower uint32` (aproksymacja)
- `sortGroupId` (broń + zbroja) → `ItemData.SortGroupId uint32` — kolejność by Item Type w grze

Test: `SortInventory(weight)` → top 10 broni musi się zgadzać z manualnym in-game sortem.

---

#### Phase 3 — In-game-style grid UI + drag&drop (8-12h)

**Nowa zakładka**: w `InventoryTab.tsx` toggle **List view / Grid view** (analog do istniejącego Inventory/Database toggle).

**Nowe komponenty:**
- `frontend/src/components/InventoryGrid.tsx` — siatka 6×N w stylu gry: ciemne tło, golden border na hover, ikona + qty + upgrade w prawym dolnym rogu
- `frontend/src/components/InventoryGridCell.tsx` — pojedyncza komórka, draggable
- Sub-tabs w grid view: tylko Phase 1 scope (Melee Armaments / Head / Chest / Arms / Legs / Talismans). Inne tabki disable z toastem "Reordering supported only for equipment items"

**Biblioteka DnD:**
- `@dnd-kit/core` + `@dnd-kit/sortable` + `@dnd-kit/utilities` (~30KB combined, modern + accessible)
- W projekcie obecnie nie ma żadnej DnD libki — **będzie to pierwsza** (`npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities`)

**State flow:**
```ts
const [sortMode, setSortMode] = useState<SortMode>('acquisition');
const [items, setItems] = useState<ItemViewModel[]>([]);

// On sortMode change → SortInventory(charIdx, cat, sortMode), refresh
// On drag end (DndContext.onDragEnd) → optimistic local reorder + ReorderInventory(charIdx, cat, newHandles)
// "Reset to default" button → SortInventory(charIdx, cat, 'sortGroupId')
```

**Wizualne detale (in-game look):**
- Tło: `bg-zinc-900` z lekkim grain pattern (CSS `radial-gradient`)
- Komórka: `64x64px`, golden border `1px solid var(--primary)` na selected/hovered
- Ikona: 56x56 centered (reuse `IconPath` z `ItemViewModel`)
- Quantity badge: prawy dolny róg, `text-xs font-black`
- Upgrade badge: lewy dolny róg, `text-[10px] text-amber-400`
- Drag preview: półprzezroczysta kopia (`opacity-50`)
- Drop indicator: golden vertical line między komórkami
- Empty cells na końcu siatki dla wizualnej spójności (jak w grze)

**Obsługa:**
- Click → preview w `ItemDetailPanel` (już istnieje)
- Drag → reorder
- Right-click / long-press → context menu (Remove / Set Quantity / Upgrade — reuse z `InventoryTab`)

---

#### Phase 4 (opcjonalna) — Per-character preset persistence (2-3h)

Integracja z **Character Preset Export/Import** (Phase 6 — Character Preset Export/Import (JSON profile)):
- Dodaj `inventoryOrder: { weapons: [handle1, ...], armor: [...], talismans: [...] }` do `CharacterPreset`
- Apply: po `AddItemsToCharacter` → wywołaj `ReorderInventory` per kategoria

---

### Phase summary

| Phase | Czas |
|---|---|
| **0** Verify in-game (krytyczne!) | 1-2h |
| **1** Backend API + 2 sort modes (acquisition/alphabetical) | 3-5h |
| **2** Import erdb + 4 sort modes (weight/attack/type/upgrade) | 2-3h |
| **3** Grid UI + drag&drop + in-game look | 8-12h |
| 4 (opcj.) Preset persistence | 2-3h |
| **Total minimum (0+1+3)** | **12-19h** |
| **Total full (0-3)** | **14-22h** |
| **Z presetami (0-4)** | **16-25h** |

---

### Open questions before Phase 0

1. **Phase 0 weryfikacja**: user przygotowuje save z hex-editem sam, czy edytor ma wyprodukować test save (możliwe ale wymaga ad-hoc Go scriptu)?
2. **Scope kategorii**: bronie + zbroja + talizmany only (jak user prosił), czy też shieldy / Ashes of War / Ashes / consumables?
3. **Sort modes** dostępne w UI:
   - `Acquisition Order` (default, no extra data) — Phase 1
   - `Alphabetical` — Phase 1
   - `Item Type` (`sortGroupId`) — Phase 2 (wymaga erdb import)
   - `Weight` — Phase 2
   - `Attack Power` (broń) / `Defense` (zbroja) — Phase 2
   - `Upgrade Level` — Phase 1 (mamy już `currentUpgrade` w VM)
   - `Custom (drag&drop)` — Phase 3 (zawsze dostępny)
4. **Reset button** behavior: wraca do `sortGroupId` (in-game default), `acquisition` (oryginalny pickup order), czy disable po użyciu drag&drop?
5. **Persistence after `WriteSave`**: zachowujemy custom porządek na zawsze (aż user kliknie Reset), czy invalid'ujemy po jakiejś akcji (np. Add Items)?
6. **Sloty 0-432 reserved**: nasz reorder pomija je (sugerowane) czy też przeindexowuje? Sugeruję NIE ruszać `Index <= 432` — to fizycznie equipped items, gra je kontroluje przez osobne offsety

**Status**: 🔲 Planned — paused; przed startem wymagana decyzja Phase 0 (in-game weryfikacja `acquisition_index` semantyki).

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

**Implementation:** `backend/core/offset_defs.go`, `backend/db/data/presets.go`, `backend/db/data/presets_generated.go`, `backend/db/data/hair_mapping.go`, `scripts/parse_presets.go`, `app.go`, `frontend/src/components/CharacterTab.tsx`, `frontend/src/components/AppearanceTab.tsx`, `spec/31-appearance-presets.md`
- FaceData blob layout fully mapped (303 bytes): header, 8 model IDs, 64 face shape params, 7 body proportions, 91 skin/cosmetics bytes
- **Mirror Favorites preset slot layout reverse-engineered** (`spec/31`) — 0x130 bytes per slot, 15 slots in CSMenuSystemSaveLoad, 5 segments map directly to FaceData
- Apply onto character via `ApplyMirrorFavoriteToCharacter(charIdx, mirrorIdx)` — copies bytes from Mirror slot to FaceData (works cross-gender M↔F because Mirror slots hold real PartsIds when preset is created in-game), flips `Gender`, zeros trailing flags `0x125..0x126`, preserves `unk0x6c`. Tested byte-for-byte against in-game apply on `tmp/re-character/`.
- Hair model IDs use non-sequential lookup table (`hair_mapping.go`) — UI position ≠ PartsId
- Other male model IDs use PartsId = UI - 1 (bone structure, beard, eyebrow, eyelash, tattoo, eyepatch)
- VoiceType added to PlayerGameData (offset -245 from MagicOffset)
- 20+ presets from eldensliders.com (parsed by `scripts/parse_presets.go` from `tmp/characters/characters.md`)
- Mirror Favorites: writes to all 15 slots (after ProfileSummary offset fix removed band-aid `FavSafeSlots`)
- Favorites header: 0xFACE marker, 0x11D0 constant, body_type inverted (0=male, 1=female)
- UI (CharacterTab): checkbox selection, image zoom modal, **Add to Mirror** (writes Type A presets) → ✓ on Mirror slot to apply. ✗ to remove from Mirror.
- UI guard: blocks Add to Mirror for Type B presets (would corrupt slot — see below)
- Undo supported via standard pushUndo mechanism

**Known limitations / TODO:**
- ✅ Male hair mapping complete: all 37 styles (UI 1-37) confirmed via save slot analysis + Mirror Favorites preset extraction
- ✅ DLC hair positions (UI 32-37) confirmed
- ✅ Cross-gender apply works via Mirror Favorites Apply (real PartsIds from in-game presets)
- ❌ `WriteSelectedToFavorites` skips Model IDs for Type B (no female PartsId mapping in `presets.go`) — UI guard blocks this path until fixed
- 🔜 Re-source `presets.go` Type B as raw 0x130 B blobs sourced from real save files — would unblock Add to Mirror for female presets
- Equipment clearing (game zeroes gender-specific gear on apply) intentionally NOT replicated — preserves player's gear at cost of possibly invisible armor for cross-gender slots

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

### ✅ Ban-Risk Awareness System 🟡
3-tier UI system that educates the user about which edits are likely to trigger Easy Anti-Cheat detection during online sync, instead of silently allowing or hard-blocking them.

**Implementation:** `frontend/src/data/riskInfo.ts`, `frontend/src/state/safetyMode.tsx`, `frontend/src/components/Risk{InfoIcon,Badge,ActionButton,SectionBanner}.tsx`, `frontend/src/components/SafetyModeBanner.tsx`, `spec/32-ban-risk-system.md`
- **Tier 0 / 1 / 2**: cosmetic / caution (modal-confirm with per-action opt-out) / high-risk (modal + field outline + clamping under Online Safety Mode)
- **`RISK_INFO` dictionary** — 24 entries: 4 per-flag (`cut_content`, `pre_order`, `dlc_duplicate`, `ban_risk`), 7 per-field (Tier 2 — `runes_above_999m`, `stat_above_99`, …), 13 per-bulk-action (Tier 1 — `bulk_grace_unlock`, `map_reveal_full`, `quest_step_skip`, `character_import`, …). Each entry has `whyBan / reports / mitigation / sources` framed as community-reported, not officially confirmed
- **Online Safety Mode** — global toggle in Settings → Safety; when enabled: top-level amber banner, Tier 1 forces confirmation modal regardless of "Don't ask again", Tier 2 inputs auto-clamp to legal max (e.g. Runes ≤ 999,999,999) with toast
- **Components**: `<RiskInfoIcon>` (clickable ⚠ + popover via `createPortal`), `<RiskBadge>` (CUT / ⚠ BAN inline tag), `<RiskActionButton>` (button + confirm modal + per-action `localStorage` dismissal), `<RiskSectionBanner>` (warning bar above whole sections)
- **Coverage**: 11 bulk actions in `WorldTab` (Unlock All / Activate All / Kill All / Reveal All / Set quest step), `CharacterImporter` confirm, Runes input outline + clamp, Database/Inventory ban-risk badges, Gestures ⚠ icons. Section banners on Map and Quests
- **Cleanup during this work**: removed dead `GeneralTab.tsx`, `StatsTab.tsx`, `WorldProgressTab.tsx` (legacy components no longer routed from `App.tsx`)

### ✅ Information Tab + DB Categorization Audit 🟡
**Implementation:** `backend/db/data/info.go` (new), `backend/db/data/key_items.go`, `backend/db/data/tools.go`, `backend/db/data/crafting_materials.go`, `backend/db/db.go`, `frontend/src/components/CategorySelect.tsx`, `frontend/src/components/DatabaseTab.tsx`, `spec/33-db-categorization-audit.md`
- Created `Information` category matching the in-game Informacje/Information tab — 114 entries spanning About tutorials, Notes, Letters, Maps, Paintings, Cross/Diary messages
- Migrated 105 misclassified items across `tools.go` ↔ `key_items.go` ↔ `crafting_materials.go`: Multiplayer Items (13) + Remembrances (25) → tools, Crystal Tears (11) + 7 keys/scrolls → key_items, 5 crafting materials → crafting_materials
- Cut/ban-risk flag audit: dropped `cut_content` from `0x400023A7 About Monument Icon` (was reachable on disc v1.0, removed in patch 1.06 — kept `ban_risk` since EAC doesn't whitelist by version), preserved on confirmed cut items (About Multiplayer, Erdtree Codex, Burial Crow's Letter, Keep Wall Key)
- Source of truth: **Fextralife per-item breadcrumb categories** + in-game user verification. er-save-manager's `KeyItems.txt`/`NotesPaintings.txt` proved unreliable (community taxonomy ≠ in-game UI placement)
- Known issue: `Prayer Room Key` icon is a binary copy of `gestures/prayer.png` (identical bytes); needs manual artwork replacement (TODO comment in code)

### ✅ Item Caps Enforcement + NG+ Scaling + Full Chaos Mode 🟡
**Implementation:** `backend/db/data/types.go`, `backend/db/data/info.go`, `backend/db/data/key_items.go`, `backend/db/data/bolstering_materials.go`, `frontend/src/components/DatabaseTab.tsx`, `frontend/src/components/SettingsTab.tsx`, `spec/34-item-caps.md`
- Tightened MaxInventory/MaxStorage caps for ~155 items so they reflect Fextralife single-playthrough obtainable counts: 29 paintings/notes 99/600 → 1/0, 11 Crystal Tears 99/600 → 1/0, 109 cookbooks (varied) → 1/0
- New `scales_with_ng` flag for 7 items (Stonesword Key 55, Dragon Heart 22, Larval Tear 24, Golden Seed 30, Sacred Tear 12, Scadutree Fragment 50, Revered Spirit Ash 25) — effective cap = base × (ClearCount + 1) so NG+1 doubles the cap, NG+7 multiplies by 8
- `Mohg's Great Rune` relocated `bolstering_materials.go` → `key_items.go` (correct in-game tab)
- New **Full Chaos Mode** Settings toggle (red-bordered, in Safety section) bypasses all caps to 999 with explicit ban-risk copy. Cross-component sync via `'fullChaosModeChanged'` window CustomEvent
- Modal banner shows live "Vanilla NG: X · NG+Y: Z · Adding more increases EAC ban risk" when any selected item has the `scales_with_ng` flag
- Clamp enforced **only at user-add** (DatabaseTab modal min/max + handleAdd) — load/save paths untouched, so legacy saves with high item counts are not retroactively clipped
- TODO: verify `ClearCount` semantic on real save (when does ClearCount increment — at Elden Beast kill or after entering NG+ menu? Current implementation assumes 0 = pre-Elden-Beast first cycle)

### ✅ Inventory & Item Database — 1:1 Game-Aligned Layout 🟡
**Implementation:** `backend/db/data/subcategories.go` (new), `backend/db/data/melee_subcat.go` (new), `backend/db/data/key_items_subcat.go` (new), `backend/db/data/info_subcat.go` (new), `backend/db/data/ranged_and_catalysts_subcat.go` (new), `backend/db/data/shields_subcat.go` (new), `backend/db/data/types.go`, `backend/db/db.go`, `app.go`, `backend/db/data/tools.go` / `key_items.go` / `info.go` / `bolstering_materials.go` / `arrows_and_bolts.go` / `shields.go` / `melee_armaments.go` (rename z `weapons.go`) / `ashes_of_war.go` (rename z `aows.go`) / `head.go` (rename z `helms.go`), `frontend/src/components/CategorySelect.tsx` / `InventoryTab.tsx` / `DatabaseTab.tsx`, `frontend/src/App.tsx`, `frontend/public/items/tools/*` (flatten 11 sub-folderów), `frontend/public/items/info/` (new), `spec/36-inventory-categories-game-order.md`
- Aligned `Inventory` and `Item Database` tabs 1:1 with the in-game inventory layout — 18 categories in canonical game order (`Tools → Ashes → Crafting Materials → Bolstering Materials → Key Items → Sorceries → Incantations → Ashes of War → Melee Armaments → Ranged Weapons / Catalysts → Arrows / Bolts → Shields → Head → Chest → Arms → Legs → Talismans → Info`). Display labels match game UI: `Ashes` (not "Spirit Ashes"), `Info` (not "Information"), `/` separator (not `&`)
- New `SubCategory` field on `ItemData` + `ItemEntry` propagating to frontend. `subcategories.go` is single source of truth for ~70 sub-cat constants across 8 tabs that have sub-grouping (Tools 12, Bolstering 6, Key Items 9, Melee 30, Ranged 7, Arrows/Bolts 4, Shields 4, Info 3)
- Reclassifications: **Larval Tears** moved to Key Items / Larval Tears + Deathroot + Lost AoW; **Torches** (9) moved from Melee → Shields (top); **Region Maps** (24) consolidated to Key Items / World Maps (zero duplication, removed from `info.go`); **Golden Runes** (33) moved Bolstering → Tools; **Whetblades + Cookbooks** un-filtered from `key_items` so they're visible in Item Database (still managed exclusively from World UI). **Bell Bearings** stay filtered (single source of truth = World → Bell Bearings UI per user decision 2026-04-28). Bastard Sword/Bolt of Gransax/Bloody Helice corrected to Greatswords/Great Spears/Heavy Thrusting Swords
- New `app.GetItemListChunk(category)` enables progressive 18× chunked load in `Item Database`'s "All Categories" view — first items visible <100 ms, scroll non-blocking during load (thin progress strip above table with `pointer-events: none`)
- Top bar restructure on both tabs: `[Cat dropdown][Owned/Total badge][Search]`. Search debounced 200 ms via `useDeferredValue`. Sub-Category column shown for tabs with sub-cats, hidden otherwise; in `'all'` view shows main category as fallback
- Header consolidation in App.tsx: `[Inventory|Item Database toggle pills][global capacity bar]` (Inventory view) / `[toggle pills][▶ Add Settings accordion 4-param summary]` (Database view) on a single line. Add Settings summary now shows 4 params (`+25 · +10 · Standard · Ash +10`) including Infuse
- Icon directory cleanup: 11 `items/tools/<sub>/` sub-folders flattened to root + IconPaths inlined in `tools.go` via Go generator pass; new `items/info/` with 49 info-tab icons relocated from `key_items/`
- File renames (Phase 0 — git mv, var symbols intact): `weapons.go` → `melee_armaments.go`, `aows.go` → `ashes_of_war.go`, `helms.go` → `head.go`. Stray Torchpole (`0x00F55C80`) moved to `shields.go`
- Source of truth: Eldenpedia inventory tab list + Fextralife per-item breadcrumb + in-game observation (PC make dev + Steam Deck verification). Extends spec/33 (Information tab + Multiplayer/Remembrances audit)
- Known limitations: Active vs Inactive Great Runes not split (DB doesn't distinguish; sub-group "Active Great Runes" empty); Quest Tools sub-group empty (quest items in info.go/key_items.go); 2 DLC info icons missing on disk (`message_from_leda.png`, `tower_of_shadow_message.png` — pre-existing absent); Melee classifier is best-effort (curated lookup + suffix fallback over 427 weapons; user reports drive `melee_subcat.go` patches)

### ✅ World Tab Collapsed Actions & Per-Session State 🟢
**Implementation:** `frontend/src/components/AccordionSection.tsx`, `frontend/src/components/WorldTab.tsx`, `frontend/src/App.tsx`, `frontend/src/components/RiskActionButton.tsx`
- All 11 World sections (map / graces / pools / colosseums / bosses / quests / gestures / cookbooks / bells / whetblades / regions) start collapsed on every save load and only persist their open/closed state for the current session
- Bulk action buttons (Unlock All / Lock All / Reveal All / Reset / Activate All / Deactivate All / Kill All / Respawn All) sit on the collapsed header next to the progress bar — single-click bulk edits without an extra expand step
- New `resetSignal` prop on `AccordionSection` — when defined, state lives in `sessionStorage` and resets to `defaultOpen` whenever the value changes; equality-guarded ref protects against React 18 StrictMode double-invoked effects
- Pools: added "Deactivate All". Colosseums: added "Lock All". "Map & Fog of War" → "Map". Bosses "Respawn" → "Respawn All"
- `btnSm` style updated to `border-foreground/30 bg-foreground/5` so action button borders are readable in both light and dark themes
- Online Safety Mode contract simplified: confirmation modals appear only when Safety Mode is on; off-mode click runs the action immediately. Dismissal plumbing (`localStorage.setItem('setting:dismissedRisk:…')`, `Don't ask again` checkbox, `allowDismiss` prop) removed. ⚠ info icon next to each action stays as the always-on educational affordance

### 🔲 DB Cleanup, Cut-Content Registry & Multiplayer Dedup 🟡
Comprehensive cleanup of in-app Item Database based on user-reported in-game evidence (2026-04-28). Many items appear with `[ERROR]` prefix in-game (missing FMG entries), in wrong inventory section (ban risk), or as visual duplicates (filled+empty flask variants, multiplayer active/inactive states). Branch: `fix/db-error-items-and-duplicates`.

**Source of truth:** user in-game screenshots:
- `tmp/Zrzut ekranu 2026-04-28 o 00.08.14.png` — character wearing unidentified light "Altered" hood + chest set, showing in-game without crash → cut content suspect
- `tmp/Zrzut ekranu 2026-04-28 o 00.11.57.png` — Crimson Tears Flask appears multiple times in Tools menu (filled+empty variants treated as separate items)
- `tmp/Zrzut ekranu 2026-04-28 o 00.13.11.png` — multiplayer items (Wizened Finger, Furled Finger, Effigies) appear with active+inactive icons of same item
- `tmp/Zrzut ekranu 2026-04-28 o 00.14.24.png` — Database tab shows "ICON" placeholders for Notes (missing icon files)

**User decisions (2026-04-28):**
- Filled+empty flask variants ARE duplicates — verify save section/offset routing
- Multiplayer active/inactive: dedup before adding (skip if any variant present)
- Items showing `[ERROR]` in-game are either cut_content OR saved in wrong section (ban risk) — flag liberally
- Notes auto-given at game start (Great Coffins, Revenants, Gateway) — remove from DB entirely
- Unknown helmet/armor from screenshot → mark cut_content
- All uncertain items → mark cut_content + list in spec/ for later verification

---

#### Phase A — Empty Flask variants cleanup
**File:** `backend/db/data/tools.go`

1. Verify in `tmp/erdb/1.10.0/EquipParamGoods.csv` whether `(Empty)` IDs (e.g. 0x400003E8) have distinct game params or are placeholder duplicates of `(Filled)` IDs.
2. If verified as redundant: remove ~27 `(Empty)` entries from `tools.go`:
   - Crimson Tears Flask Empty: 0x400003E8 + 12 upgrade variants
   - Cerulean Tears Flask Empty: base + 12 upgrade variants
   - Wondrous Physick Flask Empty: 0x400000FA
3. Cross-check `goodsType` column (NormalItem=0 vs ReinforceMaterial vs other) — writer routing may depend on type.

#### Phase B — Multiplayer active/inactive dedup
**Files:** `backend/core/writer.go`, `backend/db/db.go`

**Hypothesis:** save stores separate handles for "inactive" (held) vs "active" (deployed/used) state of multiplayer items. Game auto-rewrites handle on activation. Our writer adds inactive handle even when active variant exists → user sees 2 in-game.

1. Forensic on `tmp/crash/ER0000.sl2` — search CommonItems for known multiplayer pairs to identify active-state IDs (probably handles near 0x4000006A, 0x400000B3, etc.).
2. Build `MultiplayerStatePairs map[uint32]uint32` (active→inactive) in `db/db.go`.
3. In `addToInventory`: before insert, check if active OR inactive variant already present. Skip if yes.
4. Affected items (per user report): Tarnished's Wizened Finger (0x4000006A), Tarnished's Furled Finger (0x400000AA), Small Golden Effigy (0x400000B3), Small Red Effigy (0x400000B4) + likely all 11 multiplayer items in `tools.go:7-19`.

#### Phase C — Cut-content registry + flag uncertain items
**Files:** `backend/db/data/info.go`, `backend/db/data/tools.go`, `backend/db/data/key_items.go`, `spec/36-cut-content-registry.md` (new)

Items to flag `cut_content, ban_risk` (user reported `[ERROR]` icon in-game OR wrong save section):

**Notes (info.go) — flag, keep in DB:**
| ID | Name |
|---|---|
| 0x4000222E | Note: Hidden Cave |
| 0x4000222F | Note: Imp Shades |
| 0x40002230 | Note: Flask of Wondrous Physick |
| 0x40002231 | Note: Stonedigger Trolls |
| 0x40002232 | Note: Walking Mausoleum |
| 0x40002233 | Note: Unseen Assassins |
| 0x40002235 | Note: Flame Chariots |
| 0x40002236 | Note: Demi-human Mobs (Half-Wolves PL) |
| 0x40002237 | Note: Land Squirts |
| 0x40002238 | Note: Gravity's Advantage |
| 0x4000223A | Note: Waypoint Ruins |
| 0x4000223D | Note: Frenzied Flame Village |

**Tools (tools.go) — flag:**
- 0x40000BCC Miranda's Prayer (user reported `[Error]Modlitwa Mirandy` in-game)
- ✅ Spectral Steed Whistle — fixed: hex `0x400000B5` was the duplicate entry from the Multiplayer block; canonical ID is `0x40000082` (item 130 per er-save-manager / ER-Save-Editor reference). Updated in `tools.go` + `descriptions.go`.
- Scorpion Stew (DLC) — user reports 3 visible in-game; we have 2 (`0x401E8932`, `0x401E8933`). er-save-manager DLCConsumables.txt lists 4 IDs (2001200..2001203) with duplicate names — likely base vs reward/NG+ variants. Missing IDs to verify in-game: `0x401E8934` (2001202 Scorpion Stew), `0x401E8935` (2001203 Gourmet Scorpion Stew). TODO comment added at the entry in `tools.go`.
- ✅ Innard Meat `0x401E8B24` (2001700) — fixed: the item is a throwable bait (DLC), present in-game in Tools/Throwables sub-tab alongside Bone Darts. Initially mis-classified as Consumables based on the name; user verified in-game position. Reclassified to `SubcatToolsThrowables` in `tools.go`. (Note: er-save-manager DLCConsumables.txt mixes throwables with edibles — file name is misleading.)

**Key Items (key_items.go) — flag:**
- 0x4000229E Golden Order Principia (candidate for `[ERROR]Zasady Złotego Porządku` reported by user)

**Helms/Chest — unidentified set from screenshot 00.08.14:**
- Mark as cut_content after identifying via Fextralife cross-check + EquipParamProtector.csv comparison
- Suspects: dark cloth/leather "Altered" sets that exist in DB but aren't standard armor pieces

**Spec doc `spec/36-cut-content-registry.md`:**
- Table: `ID | Name | Section (info/tools/key/helm/chest) | DB file:line | Source verification (Fextralife / Unobtainable list / user in-game) | Status (confirmed/suspected)`
- Per-item Fextralife links
- Cross-references to existing flag locations

#### Phase D — Remove auto-given notes
**File:** `backend/db/data/info.go`

Player receives these automatically at game start (user-confirmed) and cannot remove → no value in DB:
- 0x40002234 Note: Great Coffins (icon file missing anyway)
- 0x40002239 Note: Revenants (icon file missing)
- 0x4000223B Note: Gateway (icon file missing)

Also check `presets/`, `audit/`, tests for references before removing.

#### Phase E — Memory of Grace duplicate investigation
**File:** `backend/core/writer.go`

User reports Memory of Grace (0x40000073) appears 2× in in-game inventory after add, despite single DB entry and existing dedup logic in `addToInventory:505-515`. Hypothesis: legacy duplicate from pre-fix writer version persists in user's save. Options:
1. Add load-time deduplicator: scan CommonItems for duplicate handles, merge stacks
2. Add save-time validator: warn if duplicate handles detected
3. Document as "fix on next clean save" if too risky

#### Phase F — Tests + build verification
1. `go test -v ./backend/db/...`
2. `go test -v ./backend/core/...` (multiplayer dedup unit tests)
3. `cd frontend && npx tsc --noEmit && npm run lint`
4. `make build`

#### Phase G — Docs
1. `CHANGELOG.md` — entry per phase
2. This ROADMAP entry → mark phases ✅ as completed
3. `spec/36-cut-content-registry.md` linked from spec/README

---

**Open questions before implementation:**
- Helmet/chest item from screenshot 00.08.14.png — exact EN name needed (user to provide via game language switch or inventory screenshot with cursor-on item)
- Whether Phase B (multiplayer dedup) requires separate "active state" IDs in DB or just runtime save inspection
- Whether to keep flagged-but-not-removed notes in DB (current proposal) or move to a separate `cut_content_archive.go` file

**Effort estimate:** 4-6 iterations (Phase A trivial, B+C+D+E investigation-heavy)

---

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

### 🔲 Character Preset Export / Import (JSON profile) 🟡
Human-readable JSON dump of a character profile (stats + inventory + storage + opcjonalnie wygląd / world flags) z możliwością re-importu na inny slot, oraz edycja presetu offline (bez ładowania save'a).

**Why:**
- Share builds — gracze wymieniają się buildami w community bez kopiowania całych `.sl2`
- Backup przed eksperymentami — szybki snapshot postaci do JSON, restore w 1 kliku
- Replace dla aktualnie wyłączonego `App.ImportCharacter` (`app.go:1719`, "temporarily disabled during architecture refactor") — nowa ścieżka jest cleaner (po BaseID, nie po surowych bajtach)
- Standalone editing — power-user planuje build w aplikacji bez load'owania save'a, później aplikuje jednym kliknięciem

**Source of truth:** istniejący `vm.CharacterViewModel` (już ma `json:` tagi) + spec/31 (FaceData layout) + spec/34 (item caps + NG+ scaling, walidacja przy apply).

**Format pliku:** JSON z `formatVersion: 1` i `appVersion` w nagłówku — versioned dla backward-compat. **Nie YAML** (nowa zależność `gopkg.in/yaml.v3`), **nie TXT** (nieparsowalny w drugą stronę). Items identyfikowane po `BaseID + upgrade + infuse + quantity` (NIE po runtime handle — handles są re-generowane przy apply).

```json
{
  "formatVersion": 1,
  "exportedAt": "2026-04-29T20:14:00Z",
  "appVersion": "0.7.0",
  "character": {
    "name": "OiSiSk", "class": 0, "className": "Vagabond",
    "level": 150, "souls": 999999999,
    "vigor": 60, "mind": 25, "endurance": 40,
    "strength": 50, "dexterity": 30, "intelligence": 9, "faith": 7, "arcane": 9,
    "talismanSlots": 3, "clearCount": 0,
    "greatRuneOn": true, "equippedGreatRune": 1073741909,
    "scadutreeBlessing": 20, "shadowRealmBlessing": 13
  },
  "inventory": [
    { "baseId": 134218848, "name": "Uchigatana", "quantity": 1, "upgrade": 25, "infuse": 800 },
    { "baseId": 1073741857, "name": "Crimson Tears Flask", "quantity": 14, "upgrade": 12 }
  ],
  "storage": [ /* ... */ ]
}
```

**Spec doc:** `spec/37-character-presets.md` (TBD — utworzyć w Phase 1).

---

#### Phase 1 — Export MVP (stats + inventory + storage) — ~4-6h

**Backend** (~80 linii, nowy plik `backend/vm/preset.go`):
```go
type CharacterPreset struct {
    FormatVersion int                 `json:"formatVersion"`  // 1
    ExportedAt    string              `json:"exportedAt"`     // RFC3339 UTC
    AppVersion    string              `json:"appVersion"`     // "0.7.0"
    Character     CharacterPresetCore `json:"character"`
    Inventory     []PresetItem        `json:"inventory"`
    Storage       []PresetItem        `json:"storage"`
}

type CharacterPresetCore struct {
    Name string `json:"name"`
    Class uint8 `json:"class"`
    ClassName string `json:"className"`
    Level uint32 `json:"level"`
    Souls uint32 `json:"souls"`
    Vigor uint32 `json:"vigor"`
    Mind uint32 `json:"mind"`
    Endurance uint32 `json:"endurance"`
    Strength uint32 `json:"strength"`
    Dexterity uint32 `json:"dexterity"`
    Intelligence uint32 `json:"intelligence"`
    Faith uint32 `json:"faith"`
    Arcane uint32 `json:"arcane"`
    TalismanSlots uint8 `json:"talismanSlots"`
    ClearCount uint32 `json:"clearCount"`
    GreatRuneOn bool `json:"greatRuneOn"`
    EquippedGreatRune uint32 `json:"equippedGreatRune"`
    ScadutreeBlessing uint8 `json:"scadutreeBlessing"`
    ShadowRealmBlessing uint8 `json:"shadowRealmBlessing"`
}

type PresetItem struct {
    BaseID         uint32 `json:"baseId"`
    Name           string `json:"name"`
    Quantity       uint32 `json:"quantity"`
    CurrentUpgrade uint32 `json:"upgrade"`
    InfuseOffset   uint32 `json:"infuse,omitempty"`
}

// VMToPreset / PresetToVM / NewEmptyPreset(class uint8) — pure functions, testowalne
```

**App methods** (`app.go`):
- `ExportCharacterPreset(charIdx int) (*vm.CharacterPreset, error)` — zwraca strukturę (Wails serializuje do JS auto)
- `ExportCharacterPresetToFile(charIdx int) (string, error)` — `runtime.SaveFileDialog` + `os.WriteFile` z `json.MarshalIndent`. Default filename: `<CharacterName>_<level>_<className>.preset.json`

**Frontend** (~30 linii, `CharacterTab.tsx`):
- Przycisk "Export Preset" w sekcji Profile (obok Add to Mirror)
- Toast: "Preset exported to: {path}"

**Tests** (`backend/vm/preset_test.go`):
- VMToPreset → PresetToVM round-trip preserves all fields
- JSON serialization stable (golden file)
- Item identity: BaseID extraction strips upgrade/infuse correctly

---

#### Phase 2 — Import / Apply do slotu — ~6-8h

**Backend** (`backend/vm/preset.go` + `app.go`, ~150 linii):

```go
type ApplyOptions struct {
    ReplaceStats     bool `json:"replaceStats"`
    ReplaceInventory bool `json:"replaceInventory"`
    ReplaceStorage   bool `json:"replaceStorage"`
    KeepName         bool `json:"keepName"`         // jeśli true, nie nadpisuj imienia w slocie
    KeepClass        bool `json:"keepClass"`        // domyślnie true — class wpływa na stat floor
}

type ApplyResult struct {
    StatsApplied      bool     `json:"statsApplied"`
    ItemsAdded        int      `json:"itemsAdded"`
    ItemsRemoved      int      `json:"itemsRemoved"`
    Warnings          []string `json:"warnings"`    // unknown item IDs, qty cap clamps, class mismatch, ...
}
```

**App methods:**
- `LoadCharacterPresetFromFile() (*vm.CharacterPreset, error)` — `runtime.OpenFileDialog` (filtr `*.preset.json,*.json`) + `json.Unmarshal` + walidacja `FormatVersion == 1`
- `ValidateCharacterPreset(preset vm.CharacterPreset) []string` — pre-flight check przed apply: unknown BaseIDs (`db.GetItemDataFuzzy`), qty > MaxInventory × (ClearCount+1) per spec/34, stat floor klasy mismatch
- `ApplyCharacterPreset(charIdx int, preset vm.CharacterPreset, opts ApplyOptions) (*ApplyResult, error)`:
  1. `pushUndo(charIdx)`
  2. Jeśli `opts.ReplaceStats` → `ApplyVMToParsedSlot` z core fields presetu (skip Name jeśli `KeepName`, skip Class jeśli `KeepClass`)
  3. Jeśli `opts.ReplaceInventory` → zbierz wszystkie handles z `slot.Inventory.CommonItems` + `KeyItems` → `RemoveItemsFromCharacter(charIdx, handles, true, false)`
  4. Loop po `preset.Inventory` → `core.AddItemsToSlot(slot, finalID, qty, 0, forceStackable)` z `finalID = baseID + upgrade + infuse`
  5. Analogicznie dla `preset.Storage` (z `actualInv=0, actualStorage=qty`)
  6. Reuse logiki AoW flag / world pickup flag / container key item / tutorial ID z istniejącego `AddItemsToCharacter` — wyciągnąć do `addOneItemToSlotWithFlags()` helper'a, używanego przez oba code paths

**Frontend** (~120 linii, nowy `frontend/src/components/PresetImporter.tsx` w `ToolsTab`):
- "Import Preset" button → `LoadCharacterPresetFromFile()` → preview card (postać X, level Y, items: Z inv + W storage)
- Checkboxes dla `ApplyOptions` (default: `ReplaceStats=true, ReplaceInventory=true, ReplaceStorage=false, KeepName=false, KeepClass=true`)
- Dropdown wyboru slotu docelowego
- Pre-flight warnings z `ValidateCharacterPreset` — lista "⚠ Unknown item ID 0xXXXX (skipped)", "⚠ Qty 999 capped to 600 (NG cap)"
- `RiskActionButton` z `riskKey="character_import"` (Tier 1 — bulk inventory replace, już istnieje w `riskInfo.ts`)
- Toast podsumowujący `ApplyResult`: "Applied: 240 items added, 12 stats, 3 warnings"

**Cleanup:**
- Usunąć dead `App.ImportCharacter` stub (`app.go:1719`) — preset import zastępuje
- Usunąć dead `CharacterImporter.tsx` route z `ToolsTab` jeśli nie używamy do niczego innego (lub przepisać żeby używał preset flow)

**Tests** (`tests/preset_apply_test.go`):
- Apply preset na czysty slot — stats + items się zgadzają
- ReplaceInventory clear+add — końcowa lista handles = preset items (po nowych handles)
- Round-trip Export → Apply na drugi slot → Export → diff zerowy (oprócz Name/Class jeśli KeepName/KeepClass)
- Walidacja qty cap: preset z `qty: 999` na itemie z MaxInventory=10 → po apply qty=10, warning w ApplyResult

---

#### Phase 3 — Standalone preset editor (offline, bez save'a) — ~10h

**Frontend-heavy refactor.** Backend już wspiera (`GetItemList`, `GetItemListChunk`, `GetInfuseTypes`, `GetClassStats` nie wymagają `save != nil`).

**State management** (`App.tsx`):
```ts
type EditorMode = 'save' | 'preset';
const [editorMode, setEditorMode] = useState<EditorMode>('save');
const [editingPreset, setEditingPreset] = useState<CharacterPreset | null>(null);
```

Gdy `editorMode === 'preset'`: nie wywołuj `GetCharacter(slot)` / `GetActiveSlots()`; selectory `currentChar`/`currentInventory` zwracają shimmed VM z `presetToVM(editingPreset)`. Większość komponentów (`CharacterTab`, `InventoryTab`, `DatabaseTab`) już bierze VM przez propsy a nie z globalnego state — refactor głównie w `App.tsx` selectorach.

**Backend helpers** (`app.go`, ~30 linii):
- `NewBlankPreset(class uint8) vm.CharacterPreset` — domyślny preset dla danej klasy startowej (stats z `db.GetClassStats(class)`, level computed z formuły, empty inventory/storage)
- `SavePresetToFile(preset vm.CharacterPreset) (string, error)` — file dialog + write
- `ApplyToPresetItem(preset *vm.CharacterPreset, itemIDs []uint32, qty int, ...)` — analog `AddItemsToCharacter` ale modyfikujący `CharacterPreset` zamiast `SaveSlot`. Wraps `AddItemsToCharacter` semantykę (container caps, AoW flag tracking — tylko stat-tracking, bez writes do save data).

**UI changes:**
- Top bar: nowy toggle "Save / Preset Workspace" obok Open Save button
- W trybie Preset: sidebar pokazuje "Active Preset: <name>" + buttony [New Preset] [Load Preset] [Save Preset] [Apply to current Save Slot →] (ostatni disabled jeśli `save === null`)
- "New Preset" → modal wyboru klasy (10 klas) → `NewBlankPreset` → wpadamy w edycję
- Tabki Character + Inventory + Item Database działają normalnie; World/Tools/Settings — disabled lub ukryte (nie mają sensu bez save'a)
- "Save Preset" zapisuje stan `editingPreset` do JSON
- "Apply to current Save Slot" — gdy save loaded, wywołuje `ApplyCharacterPreset` (Phase 2 path) z aktualnym `editingPreset` → ten sam dialog z opcjami co Phase 2

**Refactor risk:** `selectedChar` indeks slotu i `editingPreset` to dwa różne źródła truth. Trzeba zrobić abstrakcję "currentVM" w `App.tsx`. Sugerowany pattern: `useCurrentVM()` hook zwracający `{ vm, source: 'slot'|'preset', refresh, applyChange }` — wszystkie tabki przepinamy na ten hook.

**Tests** (frontend — `vitest`):
- `presetToVM` / `vmToPreset` round-trip (lib)
- New blank preset for each class has correct base stats
- Apply preset to save slot triggers `ApplyCharacterPreset` z prawidłowymi argumentami

---

#### Phase 4 (opcjonalna) — Appearance (FaceData blob) — ~3-4h

**Co dochodzi do presetu:**
```go
type CharacterPresetCore struct {
    // ... istniejące pola
    Gender      uint8  `json:"gender,omitempty"`       // 0=female, 1=male
    VoiceType   uint8  `json:"voiceType,omitempty"`    // 0-5
    FaceDataB64 string `json:"faceDataB64,omitempty"`  // 303 B → base64 (404 chars)
}
```

**Implementation:**
- Export: `slot.Data[FaceDataStart():FaceDataStart()+core.FaceDataBlobSize]` → `base64.StdEncoding.EncodeToString`
- Apply: precedens 1:1 z `ApplyMirrorFavoriteToCharacter` (`app.go:2634`) — copy 303 B w te same 5 segmentów (model IDs 32B, face shape 64B, body 7B, skin 91B + flag zeros + Gender flip)
- UI: nowy checkbox `ReplaceAppearance` w `ApplyOptions`
- Preset filename suffix `.face.json` jeśli zawiera FaceData (pure-stats vs full-character)

**Caveat:** cross-gender Type B preset wymaga PartsId mapping (jest `hair_mapping.go`); dla zwykłego raw blob copy gra obsługuje bez problemów (tak działa ApplyMirrorFavorite). UI ostrzeżenie gdy `preset.gender != slot.gender` (precedens: `WriteSelectedToFavorites` guard).

---

#### Phase 5 (opcjonalna) — World flags (graces / bosses / quests / maps / cookbooks / bell bearings / whetblades / AoW / gestures / regions) — ~12-16h

**Co dochodzi do presetu:**
```go
type CharacterPresetWorld struct {
    Graces          []uint32 `json:"graces,omitempty"`           // unlocked grace IDs
    Bosses          []uint32 `json:"bosses,omitempty"`           // defeated boss IDs
    Quests          map[string]int `json:"quests,omitempty"`     // npcName → stepIndex
    MapRegions      []uint32 `json:"mapRegions,omitempty"`       // visible region flag IDs
    Cookbooks       []uint32 `json:"cookbooks,omitempty"`        // unlocked cookbook IDs
    BellBearings    []uint32 `json:"bellBearings,omitempty"`     // unlocked BB flag IDs
    Whetblades      []uint32 `json:"whetblades,omitempty"`       // unlocked whetblade flag IDs
    AshOfWarFlags   []uint32 `json:"ashOfWarFlags,omitempty"`    // acquired AoW flags
    Gestures        []uint32 `json:"gestures,omitempty"`         // unlocked gesture IDs
    UnlockedRegions []uint32 `json:"unlockedRegions,omitempty"`  // invasion region IDs
    FogOfWarRemoved bool     `json:"fogOfWarRemoved,omitempty"`
}
```

Ekstrakcja (export): per-kategoria `Get*` → filtruj `unlocked/defeated/visible: true` → zapisz IDs.
Apply: per-kategoria `BulkSet*` (już istnieją). Diff vs current state, by nie touchować nieznanych IDs.

**Why opcjonalne:** to robi się "full save clone via JSON" co jest sporym scope creep — najczęściej user chce share'ować builda (stats + items), nie cały world progress. Decision deferred do user feedback po Phase 1+2.

---

### Phase summary

| Phase | Czas | Wartość |
|---|---|---|
| **1** Export MVP | 4-6h | Snapshot postaci → JSON |
| **2** Import / Apply | 6-8h | Bidir. preset workflow, replace dla wyłączonego ImportCharacter |
| **1+2 (rekomendowany start)** | **10-14h** | 80% wartości feature'u |
| 3 Standalone editor | +10h | Edycja offline bez save'a |
| 4 Appearance blob | +3-4h | Pełen visual transfer |
| 5 World flags | +12-16h | Full character clone |

### Open questions / decisions to confirm before Phase 1

1. **Equipped items w MVP**: pominąć (gracz re-equipuje z inwentarza po apply) czy dodać sekcję `equipped: { weapon1, helm, ring1, ... }` z item handles? — sugeruję pominąć w v1, dodać w Phase 4 razem z appearance jeśli okaże się potrzebne.
2. **Class change przy apply**: domyślnie `KeepClass=true` (precedens: spec/34 mówi że class wpływa na stat floor walidację). Override checkbox dostępny.
3. **NG+ scaling przy walidacji**: efektywny cap dla item w preset = `MaxInventory × (preset.ClearCount + 1)` czy `MaxInventory × (slot.ClearCount + 1)`? Sugeruję slot's NG+ cycle (gracz aplikujący na NG+5 dostaje NG+5 capy mimo że preset zrobiony na NG).
4. **Fail-fast vs best-effort przy apply**: jeśli 5/240 item IDs nieznanych → kontynuuj z warningami czy abort? Sugeruję best-effort z warningami w `ApplyResult` (precedens: `AddItemsToCharacter` zwraca `[]SkippedAdd`).
5. **Backup przed apply**: zwykły `pushUndo(charIdx)` (5-deep stack) czy też auto-export presetu obecnej postaci do `<savePath>.preset.bak.json` przed nadpisaniem? Sugeruję ten drugi — szybki rollback poza limit undo.

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
- Map Exploration & Fog of War removal (maps.go, app.go, spec/27-map-reveal.md)
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

### ✅ Console Log Order & Auto-Scroll 🟢
Three related UX issues with the Quake console resolved.

**Implementation:** `frontend/src/components/ToastBar.tsx`
- Logs rendered with newest entry on top (`logs.slice().reverse()`) — no auto-scroll needed, latest is always visible
- Removed click-outside `useEffect` — console stays open until user explicitly toggles via backtick or X button
- Scroll position preserved when reading older entries (scrollable inner container untouched on new log)

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

---

## Phase 9 — Transactional Item Adding (Crash Prevention) 🔴

> **Problem:** `AddItemsToCharacter` modyfikuje slot bez walidacji capacity i bez rollbacku. Partial failure (pełny inventory, pełna tablica GaItems, pełny GaItemData) zostawia slot w niespójnym stanie: orphaned GaItems, handle bez inventory entry, uszkodzony counter. Gra crashuje przy ładowaniu (`EXCEPTION_ACCESS_VIOLATION`).

> **Design principle:** ALL-OR-NOTHING — albo wszystkie żądane itemy zostają dodane, albo żaden. Partial write = corrupted save = niedopuszczalny.

### Architektura rozwiązania

```
     ┌─────────────────────────────────────────────────────────┐
     │               AddItemsToCharacter (app.go)              │
     │                                                         │
     │  1. PRE-COMPUTE: finalIDs, quantities, container caps   │
     │  2. PRE-FLIGHT: CheckSlotCapacity() — all fit?          │
     │     └─ NO  → return AddResult{CapHit, 0 added}         │
     │     └─ YES → continue                                   │
     │  3. SNAPSHOT: deep copy slot state                       │
     │  4. MUTATE: AddItemsToSlotBatch() — one rebuild         │
     │  5. POST-FLAGS: event flags, tutorials, containers      │
     │  6. VALIDATE: ValidateSlotIntegrity() — invariants OK?  │
     │     └─ NO  → ROLLBACK to snapshot, return error         │
     │     └─ YES → commit, return AddResult{success}          │
     └─────────────────────────────────────────────────────────┘
```

### Step 1 — Fix `upsertGaItemData` silent overflow 🔴

**Problem:** `writer.go:31` — when `count >= GaItemDataMaxCount (7000)`, returns `nil` instead of error. GaItem gets created but never registered in GaItemData → orphaned metadata → game crash.

**Fix:** Return `fmt.Errorf(...)` instead of `nil`. Error propagates through `AddItemsToSlot` → caller handles it.

**Files:** `backend/core/writer.go`
**Tests:** Unit test: call `upsertGaItemData` on slot with count=6999 (success) and count=7000 (error).

---

### Step 2 — Pre-flight capacity check 🔴

**New function:** `CheckAddCapacity(slot, items []ItemToAdd) (canFitAll bool, details CapacityReport)`

**Logic:**
- Count free slots: inventory CommonItems (2688 - used), storage CommonItems (1920 - used), GaItems array (5120 - used), GaItemData (7000 - count header)
- For each item in the request: classify as stackable vs non-stackable, compute how many GaItem entries needed (non-stackable: 1 per unit, stackable: 0 if handle exists, 1 if new), compute inventory/storage slots needed (stackable existing: 0, stackable new: 1, non-stackable: 1 per unit)
- Return: whether ALL items fit + which limit would be exceeded first

**Reuses:** existing `GetSlotCapacity()` (app.go:2280) for counting used slots — extract counting logic into `core.CountSlotUsage()` shared by both.

**Files:** `backend/core/capacity.go` (new), `app.go` (integration)
**Tests:** Unit test with mock slot at various fill levels (empty, 50%, 99%, full). Test with mixed stackable/non-stackable items.

---

### Step 3 — `AddResult` return type + all-or-nothing semantics 🔴

**Replace** `([]SkippedAdd, error)` with:

```go
type AddResult struct {
    Added     int          `json:"added"`
    Requested int          `json:"requested"`
    Skipped   []SkippedAdd `json:"skipped"`   // container cap trims (game mechanic, not capacity)
    CapHit    string       `json:"capHit"`     // "" | "inventory_full" | "storage_full" | "gaitem_full" | "gaitemdata_full"
    FreeInv   int          `json:"freeInv"`    // free slots after operation
    FreeStore int          `json:"freeStore"`  // free slots after operation
}
```

**Semantics:**
- If pre-flight says not all fit → `AddResult{Added: 0, Requested: N, CapHit: "..."}`, no mutation
- Container cap trims (pots/aromatics) are reported in `Skipped` but don't trigger rollback — they're game rules, not capacity failures
- `CapHit = ""` means all items added successfully

**Files:** `app.go` (type change + logic), `frontend/src/wailsjs/` (auto-regenerated)
**Tests:** Integration test: add items to near-full slot, verify AddResult fields.

---

### Step 4 — Snapshot + rollback in `AddItemsToCharacter` 🔴

**Before mutation:**
```go
snapshot := snapshotSlot(slot)  // deep copy: Data, GaItems, GaMap, Inventory, Storage, all indices
```

**After mutation (if any error in steps 4–6):**
```go
restoreSlot(slot, snapshot)     // restore all fields from snapshot
return AddResult{...}, fmt.Errorf("rollback: %w", err)
```

**Snapshot scope:** `slot.Data` ([]byte), `slot.GaItems` ([]GaItemFull), `slot.GaMap` (map), `slot.Inventory` (deep clone via existing `Clone()`), `slot.Storage` (deep clone), `slot.NextAoWIndex`, `slot.NextArmamentIndex`, `slot.NextGaItemHandle`, `slot.PartGaItemHandle`, `slot.GaItemDataOffset`, `slot.MagicOffset`, `slot.StorageBoxOffset`, `slot.EventFlagsOffset`.

**Note:** `pushUndo()` already does a similar deep copy for UI undo — snapshot/rollback is a separate, internal safety mechanism that doesn't touch the undo stack.

**Files:** `backend/core/snapshot.go` (new — `SnapshotSlot()` / `RestoreSlot()`), `app.go` (integration)
**Tests:** Unit test: corrupt mid-add, verify slot.Data matches pre-add snapshot byte-for-byte.

---

### Step 5 — Batch rebuild (`AddItemsToSlotBatch`) 🟡

**Problem:** Current flow calls `AddItemsToSlot` per-item (app.go:373). Each non-stackable item triggers `RebuildSlotFull` (~50-100ms). 50 weapons = 50 rebuilds = 2.5-5s. Batching = 1 rebuild = ~100ms.

**New function:**

```go
type ItemToAdd struct {
    ItemID         uint32
    InvQty         int
    StorageQty     int
    ForceStackable bool
}

func AddItemsToSlotBatch(slot *SaveSlot, items []ItemToAdd) error {
    // Phase 1: allocate ALL GaItems + GaItemData entries
    // Phase 2: ONE RebuildSlotFull (if any non-stackable)
    // Phase 3: add ALL to inventory/storage
}
```

**Refactor in `app.go`:**
1. Pre-compute all `finalID`, `actualInv`, `actualStorage`, `forceStackable` in a loop (lines 340-368 current logic) → collect into `[]ItemToAdd`
2. Call `AddItemsToSlotBatch(slot, items)` once
3. Post-batch: set event flags, tutorial IDs, container key items in separate loops (safe to defer — no dependency on add order)

**Container cap logic stays per-item** (stateful FCFS distribution) but runs BEFORE the batch call, not inside it.

**Files:** `backend/core/writer.go` (new `AddItemsToSlotBatch`, keep old `AddItemsToSlot` for single-item callers), `app.go` (refactor loop)
**Tests:** Benchmark test: batch 50 weapons vs per-item, verify <200ms. Roundtrip test: batch add 50 mixed items, verify all survive reload.

---

### Step 6 — Post-write validation (`ValidateSlotIntegrity`) 🔴

**Extend existing `diagnostics.go`** with a fast, focused invariant check run AFTER every mutation (not full diagnostic — only crash-causing invariants):

```go
func ValidatePostMutation(slot *SaveSlot) []IntegrityError {
    // 1. Every non-empty inventory handle exists in GaMap
    // 2. Every non-stackable GaMap entry has a GaItem record
    // 3. No duplicate Index values across inventory CommonItems + KeyItems
    // 4. NextEquipIndex > max(all item indices) — for both inventory and storage
    // 5. GaItemData count header matches actual non-zero entries
    // 6. Storage count header matches actual non-empty storage items
    // 7. NextAoWIndex <= NextArmamentIndex <= len(GaItems)
    // 8. No handle in GaMap references itemID=0
}
```

**If any check fails → trigger rollback (step 4).** This is the safety net for bugs we haven't found yet.

**Performance target:** <10ms (only iterate in-memory structures, no I/O).

**Files:** `backend/core/diagnostics.go` (new `ValidatePostMutation`), `app.go` (call after mutation, before commit)
**Tests:** Unit test: manually corrupt each invariant, verify detection. Benchmark: verify <10ms on full slot.

---

### Step 7 — Event flag error classification 🟡

**Audit all `_ = db.SetEventFlag(...)` sites** in `app.go` (6 locations):
- **Non-critical** (log to console, don't block): AoW duplication flag (line 380), world pickup flag (line 391), tutorial ID (line 401), container pickup flags (line 448), vendor purchase flags (line 457)
- **Critical** (block add + rollback): none currently — event flags are convenience features, not structural integrity

**Implementation:** Replace `_ =` with `if err := ...; err != nil { runtime.LogWarningf(ctx, "event flag %d: %v", flagID, err) }`. Frontend console already receives Wails runtime logs.

**Files:** `app.go`
**Tests:** No new tests needed — existing event flag tests cover SetEventFlag correctness.

---

### Step 8 — Frontend `AddResult` handling 🟡

**DatabaseTab.tsx changes:**

1. **Type update:** `AddItemsToCharacter` returns `AddResult` instead of `SkippedAdd[]`
2. **Capacity failure (capHit non-empty):**
   ```typescript
   if (result.capHit) {
       toast.error(`Cannot add items: ${result.capHit}. 0/${result.requested} added.`);
   }
   ```
3. **Container cap trims (skipped non-empty):**
   ```typescript
   if (result.skipped.length > 0) {
       console.log(`Trimmed ${totalCut} units due to container caps`);
   }
   ```
4. **Success path:**
   ```typescript
   console.log(`Added ${result.added}/${result.requested} items`);
   toast.success(`Added ${result.added} items`);
   ```
5. **Modal always closes, `onItemsAdded?.()` always fires** (refresh UI regardless of outcome)
6. **`isSaving` guard** prevents double-click (already exists — verify it covers all paths)

**Files:** `frontend/src/components/DatabaseTab.tsx`
**Tests:** Manual test: add items to near-full inventory, verify toast shows correct message.

---

### Step 9 — Storage count header reconciliation 🟡

**Problem:** `addToInventory` (writer.go:636) blindly increments `currentCount + 1`. If header is already wrong (e.g. from external editor), each add worsens the mismatch.

**Fix:** After batch add, reconcile storage count header with actual non-empty item count (same scan as `checkStorageHeader` in diagnostics.go). Write correct count. This runs as part of `ValidatePostMutation` (step 6) auto-repair path.

**Files:** `backend/core/writer.go` (reconcile function), `backend/core/diagnostics.go` (extend validation)
**Tests:** Unit test: set header to wrong value, add item, verify header corrected.

---

### Step 10 — Tests 🔴

New test file: `tests/capacity_test.go`

| Test | What it verifies |
|------|-----------------|
| `TestPreFlightCapacity_Empty` | Pre-flight on empty slot, all items fit |
| `TestPreFlightCapacity_NearFull` | Pre-flight on 99% full slot, reports correct free count |
| `TestPreFlightCapacity_Full` | Pre-flight on full slot, reports 0 capacity, capHit set |
| `TestPreFlightCapacity_MixedStackable` | Stackable items reuse handles, non-stackable need new GaItems |
| `TestAllOrNothing_CapacityExceeded` | Request 50, capacity for 30 → 0 added, slot unchanged |
| `TestAllOrNothing_MidAddError` | Force error mid-batch → full rollback, slot matches snapshot |
| `TestBatchRebuild_SingleVsMultiple` | Batch 20 items produces identical slot.Data as 20 single adds |
| `TestBatchRebuild_Performance` | Batch 50 weapons: <200ms (vs ~2500ms per-item) |
| `TestPostValidation_OrphanHandle` | Inject orphan handle → ValidatePostMutation detects it |
| `TestPostValidation_DuplicateIndex` | Inject duplicate index → detected |
| `TestPostValidation_CounterMismatch` | Inject wrong NextEquipIndex → detected |
| `TestStorageHeaderReconcile` | Wrong header count → corrected after add |
| `TestGaItemDataFull_ErrorNotSilent` | upsertGaItemData at count=7000 → returns error |
| `TestRoundtrip_FullInventory` | Fill inventory to 100%, save, reload → identical |
| `TestRoundtrip_BatchAdd300Items` | Add 300 mixed items in one batch, roundtrip → all survive |
| `TestAddResult_ContainerCapTrim` | Container cap trims reported in Skipped, not in CapHit |

**Files:** `tests/capacity_test.go` (new), `backend/core/writer_test.go` (extend), `backend/core/diagnostics_test.go` (extend)

---

### Implementation order

| Order | Step | Priority | Est. time | Dependency |
|-------|------|----------|-----------|------------|
| 1 | Step 1 — upsertGaItemData fix | 🔴 | 15 min | none |
| 2 | Step 4 — Snapshot/rollback | 🔴 | 1-2h | none |
| 3 | Step 2 — Pre-flight capacity | 🔴 | 2-3h | none |
| 4 | Step 6 — Post-write validation | 🔴 | 2-3h | none |
| 5 | Step 3 — AddResult type | 🔴 | 1-2h | steps 2, 3 |
| 6 | Step 7 — Event flag logging | 🟡 | 30 min | none |
| 7 | Step 9 — Storage header reconcile | 🟡 | 1h | step 6 |
| 8 | Step 5 — Batch rebuild | 🟡 | 3-4h | steps 2, 3, 4, 5 |
| 9 | Step 8 — Frontend AddResult | 🟡 | 1-2h | step 5 |
| 10 | Step 10 — Full test suite | 🔴 | 3-4h | all above |

**Total estimate:** 15-22h

---

## 🔚 Final Cleanup (do końcu wszystkich pozostałych prac)

> Te zadania robimy dopiero na końcu, po zamknięciu wszystkich feature'ów z roadmapy.

### 🔲 Dead code audit
- Pełny przegląd `frontend/src/` i `backend/` pod kątem nieużywanego kodu (komponenty, eksporty, helpery, stałe).
- Tooling: `npx ts-prune` dla TS + `staticcheck` / `golangci-lint --enable=unused` dla Go.
- Wcześniejsze straty (GeneralTab, StatsTab, WorldProgressTab) sugerują że warto powtarzać ten audyt przed każdym mergem do `main`.

### 🔲 Refactor — przyspieszenie aplikacji
- Profilowanie: cold start, czas otwarcia save'a, czas przełączania zakładek, render dużych list (Item Database, Event Flags).
- Frontend: `React.memo`, virtualization (react-virtual / react-window) dla list >500 elementów, lazy-loading zakładek (`React.lazy`).
- Backend: profilowanie `core/reader.go` + `writer.go` (`go test -bench` / `pprof`), eliminacja zbędnych alokacji w hot path parsing.
- Cel: <1s open save, <100ms tab switch, brak jankov przy filtrowaniu Database.
