# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Branch: feat/dlc-spells-cleanup — DLC sorceries / incantations + miscategorisation cleanup

**Goal:** Add 10 missing DLC spells, move 1 spell to its correct category, and remove 5 historical miscategorisations of DLC spells that lived in `tools.go` and `key_items.go`.

**Verified via Fextralife wiki for every ID before merging.**

**Added — `incantations.go` (+6):**
- `0x401E9D1C` Furious Blade of Ansbach (was wrongly in `sorceries.go`)
- `0x401E9E7A` Aspects of the Crucible: Thorns
- `0x401E9F7E` Dragonbolt of Florissax — Dragon Cult incantation
- `0x401E9FD8` Bayle's Tyranny — Dragon Cult incantation
- `0x401EA0AA` Pest-Thread Spears
- `0x401EA2BC` Divine Bird Feathers

**Added — `sorceries.go` (+4):**
- `0x401E9614` Glintstone Nail (Finger Sorcery, Ymir)
- `0x401E961E` Glintstone Nails (Glintstone Sorcery, Ymir)
- `0x401E96DC` Blades of Stone (Gaius Remembrance Sorcery)
- `0x401EA17C` Cherishing Fingers (Finger Sorcery, Ymir)

**Removed — `sorceries.go` (-1):**
- `0x401E9D1C` Furious Blade of Ansbach (incantation, moved to `incantations.go`)

**Removed — miscategorised duplicates (-5):**
- `key_items.go`: `0x401EA17C` Cherishing Fingers (sorcery, never a key item)
- `tools.go`: `0x401E9614` Glintstone Nail, `0x401E961E` Glintstone Nails, `0x401E96DC` Blades of Stone (all sorceries, not throwables)
- `tools.go`: `0x401E9F7E` Dragonbolt of Florissax (incantation, not grease)

**Tests:** `go build ./backend/... ./` ✅, `go test ./backend/...` ✅, `make build` ✅.

### Branch: fix/console-ux — Gestures: free slots before write + hide ban-risk behind setting

**Follow-up #2 to the gesture bug.** User feedback after the previous build:
1. **First Unlock All did nothing** — save still held 44 legacy even-ID garbage entries plus 13 valid odd, leaving only 7 sentinel slots; backend errored with "no empty gesture slot available" and the frontend silently swallowed it. After Lock All cleared everything, a second Unlock All worked.
2. **Ban-risk gestures still visible** in the Gestures grid — user wants them gated behind the existing `showFlaggedItems` toggle (Tools / Settings).

**Changes:**
- `app.go`:
  - New helper `purgeUnknownGestures(slots)` — replaces any non-canonical slot value (legacy even garbage, unknown IDs) with the empty sentinel.
  - `SetGestureUnlocked` and `BulkSetGesturesUnlocked` now call `purgeUnknownGestures` first, freeing space so a single Unlock All on a save corrupted by older builds succeeds in one click.
  - Lock path simplified back to canonical IDs only (the purge already wiped legacy evens, no `id-1` extension needed).
- `frontend/src/App.tsx`: pipes the existing `showFlaggedItems` setting into `<WorldTab>`.
- `frontend/src/components/WorldTab.tsx`: new `WorldTabProps.showFlaggedItems`; computes `visibleGestures` (filter out `ban_risk` unless toggle is on) and uses it for both rendering and the Unlock All bulk list. Lock All still iterates all known gestures so the save gets wiped clean. Progress counter switched to `visibleGestures.length`.

**Tests:** `tsc --noEmit` ✅, `make build` ✅, `go build ./backend/... ./` ✅.

### Branch: fix/console-ux — Gestures: stop auto-recovering ghost unlocks + ban-risk filter

**Follow-up to the previous gesture fix.** User feedback after that build:
1. **Read still showed all 57 unlocked** — auto-`SanitizeGestureSlots` on every `GetGestures` call was rewriting in-memory legacy even IDs to odd, so the UI claimed the user had every gesture even though only ~13 were really unlocked from gameplay.
2. **Unlock All triggered ban-risk content** — Pre-order Rings, "The Carian Oath", "Fetal Position", "?GoodsName?" appeared in-game with placeholder `ICON` text, indicating cut content / pre-order entries that violate online anti-cheat.

**Changes:**
- `backend/db/data/gestures.go`: added `Flags []string` to `GestureDef`. Tagged 6 entries with `cut_content` / `pre_order` / `dlc_duplicate` plus a shared `ban_risk` flag (IDs 111, 193, 217, 221, 227, 233).
- `backend/db/db.go`: `GestureEntry` now carries `Flags`; `GetAllGestureSlots` propagates them.
- `app.go`:
  - `GetGestures`: removed sanitize-on-read. Only canonical (odd) IDs count as unlocked, matching what the game actually displays.
  - `SetGestureUnlocked` lock path: also clears the `(id-1)` even legacy slot.
  - `BulkSetGesturesUnlocked` lock path: `removeSet` includes both `id` and `(id-1)` for every odd ID, so Lock All wipes the array clean even when previous builds left even garbage behind.
  - Removed sanitize-on-write call (Lock All extension is sufficient and avoids silently re-adding gestures the user didn't ask for).
- `frontend/src/components/WorldTab.tsx`, `WorldProgressTab.tsx`:
  - Unlock All filters `g.flags?.includes('ban_risk')` so ban-risk entries are never bulk-added.
  - Lock All sends every known gesture (not just unlocked ones) to ensure legacy garbage gets cleared.
  - Each gesture row shows a ⚠ next to ban-risk entries with a tooltip explaining the risk.

**User flow after the fix:**
1. Reload save → UI shows only really-unlocked gestures (13 in user's slot 4).
2. Click Lock All → all 64 entries become sentinel, including legacy even garbage.
3. Click Unlock All → 51 safe gestures written; the 6 ban-risk ones must be toggled individually if the user truly owns them.

**Tests:** `go test ./backend/...` ✅, `tsc --noEmit` ✅, `make build` ✅.

### Branch: fix/console-ux — Gestures invisible in-game (root cause + auto-repair)

**Bug reported by user:** "Gesty się nie pojawiają pomimo tego że je dodałem w apce" — gestures unlocked via the editor were silently ignored by the game.

**Root cause:** The previous editor build encoded an "EvenID / OddID body-type variant" theory in `AllGestures`. In practice all vanilla gesture slot IDs are odd (verified against `er-save-manager/data/gestures.py`, which only writes odd IDs and is known to work). When the editor wrote the EvenID, the game silently ignored it, so up to 44/57 gestures became invisible in slots edited by previous builds.

**Diagnosis (`tmp/diag-gesture/main.go`):**
- User's slot 4 contained 13 odd (correct) + 44 even (broken) + 7 sentinel = 64 entries — matches "almost all unlocks invisible in-game" report.
- All 5 active slots had the same pattern.

**Fix:**
- `backend/db/data/gestures.go`: rebuilt `GestureDef` with a single canonical `ID` (always odd) plus the matching `ItemID` from er-save-manager. Removed `EvenID`/`OddID`, removed `DetectBodyTypeOffset`. Added `SanitizeGestureSlots(slots)` which rewrites any even slot whose `(id+1)` is a known gesture to `(id+1)` — silent, idempotent migration.
- `backend/db/db.go`: `GetAllGestureSlots` returns the new canonical ID.
- `app.go`: `GetGestures` runs `SanitizeGestureSlots` on the in-memory copy before computing unlock state, so the UI immediately reflects the repaired state. `SetGestureUnlocked` and `BulkSetGesturesUnlocked` sanitise the slot before any write so the next save commits the repair to disk. Removed `resolveGestureWriteID` and `gestureMatchesCanonical` (no body-type variants exist). Added `writeGestureSlots` helper.
- Cut-content / unknown DLC entries kept under their canonical IDs so saves containing them still display correctly.

**Tests:** `go test ./backend/...` ✅, `make build` ✅. Diag verifies sanitize repairs all 5 user slots from {13–45 known + many broken} → {57 known + 0 unmatched}. Manual in-game test required to confirm gestures now appear (user simply reopens the slot and toggles Unlock All / Lock All to commit the repair).

### Branch: fix/console-ux — BB refactor to backend-driven readOnly + ROADMAP cookbook sync

**Goal:** Drop the BB-specific Wails getter and frontend Set in favour of the same backend pattern already used for Cookbooks and Whetblades. ROADMAP updated to reflect that cookbook inventory sync is in fact already shipped.

**Changes:**
- `backend/db/data/bell_bearing_flags.go`: new `IsBellBearingItemID(id)` helper.
- `backend/db/db.go`: `GetItemsByCategory("key_items")` now also skips BB items.
- `backend/vm/character_vm.go`: `ReadOnly` is now true for BB items as well.
- `app.go`: removed `GetBellBearingItemIDs()` Wails method (no longer needed).
- `frontend/src/components/DatabaseTab.tsx`: reverted BB Set + filter (backend already hides them).
- `frontend/src/components/InventoryTab.tsx`: reverted BB Set + readOnly OR (VM already marks them ReadOnly).
- `ROADMAP.md`: cookbook entry — replaced “Known issue: physical item missing” with the actual implementation note (`CookbookFlagToItemID` + `IsCookbookItemID` + `ReadOnly` in VM).

**Tests:** `tsc --noEmit` ✅, `go test ./backend/db/data/...` ✅, `make build` ✅.

### Branch: fix/console-ux — Bell Bearing single source of truth (World tab)

**Goal:** Make Bell Bearings reachable from exactly one place — World → Unlocks → Bell Bearings — and keep the acquisition flag and the matching key item perfectly in sync.

**Changes:**
- `backend/db/data/bell_bearing_flags.go`: added auto-derived reverse map `BellBearingFlagToItemID` for the World tab toggle.
- `app.go`:
  - `SetBellBearingUnlocked` now calls new helper `syncBellBearingItem`: unlock → add 1 of the matching key item to inventory if absent; lock → remove from inventory and storage. Mirrors the Whetblade pattern.
  - `BulkSetBellBearings` runs the same sync per flag.
  - New Wails method `GetBellBearingItemIDs() []uint32` for the frontend to identify managed BB items.
- `frontend/src/components/DatabaseTab.tsx`: BB items are filtered out of the Item Database list (no Add path). Loaded via `GetBellBearingItemIDs` once on mount.
- `frontend/src/components/InventoryTab.tsx`: BB items appear in the Inventory list as `readOnly` — no Remove button, no selection checkbox — so users can preview but only manage them via World → Unlocks.

**Tests:** `tsc --noEmit` ✅, `make build` ✅, manual round-trip TBD (toggle ON adds BB, toggle OFF removes from inv+storage, no DB add path remains).

### Branch: fix/console-ux — Bell Bearing acquisition flag + ROADMAP sync

**Goal:** Round out the auto-flag-on-add hooks so Bell Bearings behave like Ashes of War (Twin Maiden Husks expand wares); sync ROADMAP with already-shipped Spirit Ash and AoW work.

**Changes:**
- `backend/db/data/bell_bearing_flags.go` (new): `BellBearingItemToFlagID` map (62 entries) — itemID → acquisition event flag, generated from `BellBearings` × `key_items.go` (59 exact name matches + 3 aliases for Kalé/Kale, Spell-Machinist, String-seller). Cut-content `Nomadic [11]` excluded.
- `backend/db/data/bell_bearing_flags_test.go` (new): coverage test verifying every non-cut-content BB key item is mapped and every flag exists in `BellBearings`.
- `app.go`: `AddItemsToCharacter` now also flips `BellBearingItemToFlagID[id]` after the AoW hook.
- `ROADMAP.md`: marked **Spirit Ash Upgrade Level Editing** ✅ (already shipped via `upgradeAsh` slider) and **AoW Acquisition Flag** ✅ (already shipped via `AoWItemToFlagID`). Split the old BB roadmap entry into shipped Acquisition flag (✅) and remaining Merchant Kill flag (🔲, RE-heavy follow-up).

**Tests:** `go test ./backend/...` ✅, `tsc --noEmit` ✅, `make build` ✅. Pre-existing `tests/bulk_add_test.go` failures (unrelated, GaItem array exhaustion) confirmed present on clean main.

### Branch: fix/console-ux — Quake console UX fixes

**Goal:** Eliminate three UX papercuts in the Quake console that hurt visibility during long-running operations.

**Changes:**
- `frontend/src/components/ToastBar.tsx`: render logs reversed (`logs.slice().reverse()`) so newest entry is on top — no auto-scroll needed, latest is always in view.
- Removed click-outside `useEffect` so the console stays open while user interacts with the rest of the UI. Toggle is now strictly via backtick or X button.
- Cleaned up stale `Spectral Steed Whistle duplicate` ROADMAP entry — the duplicate `0x40000082` no longer exists in `descriptions.go`; only `0x400000B5` (correct entry in `tools.go`) remains.

**Tests:** `tsc --noEmit` ✅, `make build` ✅, manual UI verification by user.

### Branch: feature/invasion-regions — Stage 2 (write support via R-1 full slot rebuild)

**Goal:** Implement write support for the per-slot Regions struct so players can unlock/lock invasion regions from the editor. Required a full slot rebuild because shift-based in-place patching corrupted saves (first attempt rolled back).

**Approach (Option B — full struct rebuild, see `PLAN-R1.md` for the 17-step checklist):**
- Replaced shift-based `core.SetUnlockedRegions` with sequential rebuild that re-serializes every section after `unlocked_regions` from typed Go structs, then zero-pads the tail to `SlotSize`.
- 19 new section types parsed and serialized: `RideGameData`, `BloodStain`, `MenuSaveLoad`, `TrophyEquipData`, `GaitemGameData` (7000 entries × 16B), `TutorialData`, `PreEventFlagsScalars`, `EventFlagsBlock`, 5× `SizePrefixedBlob` (`field_area`, `world_area`, `world_geom_man`×2, `rend_man`), `PlayerCoordinates`, `SpawnPointBlock` (version-gated), `NetMan`, `WorldAreaWeather`, `WorldAreaTime`, `BaseVersion`, `PS5Activity`, `DLCSection`, `PlayerGameDataHash`.
- Each section has a per-slot byte-for-byte round-trip test (`backend/core/section_*_test.go`).

**Key insight (`spec/30-slot-rebuild-research.md`):** Initial slack analysis was misleading — it assumed DLC was pinned at `SlotSize - 0xB2`. After full sequential parsing we discovered every slot has 408–432 KB of zero tail padding past the parsed sections, on both PS4 and PC. DLC and hash slide left/right naturally as `unlocked_regions` grows or shrinks; the tail rest absorbs the delta.

**New files:**
- `backend/core/section_io.go` — `SectionWriter` helper (mirrors `Reader`).
- `backend/core/section_types.go` — `FloatVector3`, `FloatVector4`, `MapID` primitives.
- `backend/core/section_world.go` — `RideGameData`, `BloodStain`, `WorldHead`.
- `backend/core/section_menu.go` — `MenuSaveLoad`, `TrophyEquipData`, `GaitemGameData`(+`Entry`), `TutorialData`.
- `backend/core/section_eventflags.go` — `PreEventFlagsScalars`, `EventFlagsBlock`.
- `backend/core/section_world_geom.go` — `SizePrefixedBlob`, `WorldGeomBlock`.
- `backend/core/section_player_coords.go` — `PlayerCoordinates`, `SpawnPointBlock`.
- `backend/core/section_netman.go` — `NetMan`.
- `backend/core/section_trailing.go` — `WorldAreaWeather`, `WorldAreaTime`, `BaseVersion`, `PS5Activity`, `DLCSection`, `TrailingFixedBlock`.
- `backend/core/section_hash.go` — `PlayerGameDataHash`.
- `backend/core/slot_rebuild.go` — `RebuildSlot` (sequential rebuild driver) + `SectionRange` / `buildSectionMap`.
- `spec/30-slot-rebuild-research.md` — slack analysis with 2026-04-26 update.
- `tmp/r1-stagedeck/main.go` — Steam Deck preflight CLI.

**Modified:**
- `backend/core/writer.go`: `SetUnlockedRegions(slot, ids)` now dedupe+sort, call `RebuildSlot`, replace `slot.Data`, refresh dynamic offsets. Rolls back on error.
- `backend/core/structures.go`: `SaveSlot.SectionMap` populated during `Read()` for use by `RebuildSlot`.
- `app.go`: `SetRegionUnlocked(slotIdx, regionID, unlocked)` and `BulkSetUnlockedRegions(slotIdx, regionIDs)` Wails methods.
- `frontend/src/components/WorldTab.tsx`: actionable checkboxes, per-area `+`/`−` quick-toggle buttons, global Unlock All / Lock All.

**Tests:** `go test ./backend/...` ✅ (incl. identity round-trip, mutation +50 regions PC, shrink -5 regions PS4, full Set→Save→Load→Get round-trip on both platforms). Manual Steam Deck verification ✅ (PS4 save loaded in-game, characters intact, grace/map/gestures preserved).

**Steam Deck test save:** `tmp/r1-stagedeck/oisis-r1-test-PS4.sl2` (380 + 81 regions across 2 slots).

### Branch: feature/invasion-regions — Stage 1 (read-only)

**Goal:** Surface the per-slot Regions struct (count + u32 IDs) in the UI so players can see which map areas are unlocked for invasions / blue summons. Stage 1 is read-only; Stage 2 will add write support (variable-size slot rebuild).

**Changes:**
- `backend/core/structures.go`: Added `SaveSlot.UnlockedRegionsOffset` and `UnlockedRegions []uint32`; parser populates the list during `Read()`.
- `backend/db/data/regions.go` (new): Ported 78 region IDs from `er-save-manager/data/regions.py` — Limgrave, Liurnia, Altus/Mt. Gelmir, Caelid, Mountaintops, Underground, Farum Azula, Haligtree, Land of Shadow (DLC), and legacy dungeon aliases. Each entry has `Name` + `Area` for grouping. Helper `IsDLCRegion()`.
- `backend/db/db.go`: New `RegionEntry` type and `GetAllRegions()` returning all known regions sorted by Area then Name.
- `app.go`: `GetUnlockedRegions(slotIdx)` Wails binding — merges the database with the slot's unlocked list.
- `frontend/src/components/WorldTab.tsx`: New "Invasion Regions" accordion in the Unlocks sub-tab. Per-area expand/collapse (matching Summoning Pools/Graces). Read-only badge + tooltip on checkboxes.
- `frontend/wailsjs/go/{main,models}`: Auto-regenerated bindings (added `RegionEntry` + `GetUnlockedRegions`).

**Tests:** `go test ./backend/...` ✅, round-trip PS4/PC/conversion ✅, `tsc --noEmit` ✅, `make build` ✅. Manual verification by user — unlocked regions match in-game progress.

### Branch: feature/database-tab-owned-counts — owned/max counts in Item Database

**Goal:** Show players how many of each item they currently own (in inventory and storage) and the per-slot max, directly in the Item Database tab — without switching to Owned Items.

**Changes:**
- `backend/vm/character_vm.go`: Added `BaseID` field to `ItemViewModel` so the frontend can match upgrade/infusion variants of the same weapon back to its base DB entry.
- `frontend/src/components/DatabaseTab.tsx`:
  - Fetches character via `GetCharacter` (refreshes on `inventoryVersion` bump).
  - Builds a `Map<BaseID, {inv, storage}>` of owned counts (sums stack quantity for stackable, counts copies for non-stackable).
  - New columns **Inventory** and **Storage** rendered as `owned / max` in every view (All Categories + per-category).
  - **Category** column forced visible in "All Categories" regardless of column-visibility setting.
  - Color coding: gray = 0, green = owned, amber = at/over max.
- `frontend/src/App.tsx`: Passes `inventoryVersion` to `<DatabaseTab>` for live refresh after Add/Remove.
- `frontend/wailsjs/go/models.ts`: Auto-regenerated bindings (added `baseId`).

**Tests:** `go test ./backend/...` ✅, round-trip PS4/PC ✅, `tsc --noEmit` ✅, `make build` ✅.

### Branch: fix/dlc-map-reveal-v2 — DLC black tile removal (SOLVED)

**Problem:** DLC "Shadow of the Erdtree" map had persistent black tiles that could not be removed via event flags, FoW bitfield, map items, or any known mechanism.

**Root cause:** The DLC map cover layer is controlled by two position records in the BloodStain section (afterRegs+0x0088..0x0110). These records contain DLC-area coordinates that tell the game the player has physically explored the DLC map. Without them, the game renders black tiles over the DLC map regardless of all other flags.

**Solution:** Write synthetic DLC coordinates into the BloodStain section:
- Record 1 (afterRegs+0x008D): X=9648.0, Y=9124.0, flag=0x01
- Record 2 (afterRegs+0x00C5): X=3037.0, Y=1869.0, Z=7880.0, W=7803.0, flag=0x01

**Changes:**
- `app.go`: `revealDLCMap()` — added Phase 3 (DLC black tile removal via synthetic coordinates)
- `backend/core/offset_defs.go`: Added `DLCTile*` constants for BloodStain position offsets
- `backend/db/data/maps.go`: Added 237 dungeon map flags (62100-62999) to `MapVisible`, updated `IsDLCMapFlag()` range
- `spec/29-dlc-black-tiles.md`: Full research documentation with binary search results

**Testing:** 20 iterative tests on Steam Deck, confirmed working with base game + DLC map fully revealed.

### Branch: fix/dlc-map-reveal (experimental, not merged)

Deep research into DLC (Shadow of the Erdtree) map black tiles removal.

**Research findings (2025-04-25):**
- DLC ownership confirmed — not a runtime entitlement check
- Game resets ALL map event flags on load, then rebuilds from ground truth
- DLC map fragment items survive game reset (persist in inventory)
- FoW bitfield is shared between base game and DLC (same 2099-byte range)
- CsDlc bytes[3-49] are NOT always zero (contradicts earlier spec)
- Event flags 62080-62084 survive game load when set alongside grace flags
- **Black tiles persist despite correct flags, items, regions, and graces**

**Approaches tested (all failed to remove black tiles):**
1. Event flags 62080-62084 + 82002 only
2. Above + DLC map fragment items
3. Above + CsDlc byte[1]=1 (DLC entry flag)
4. Above + story progression flags
5. FoW bitfield extension beyond 0x10B0
6. er-save-manager flag toggle (same flags, no items/FoW)
7. Above + 105 DLC grace flags (72xxx, 74xxx, 76xxx) + acquisition flags 63080-63084
8. Above + 10 DLC region IDs (6800000-6941000) via byte insertion

**Code changes (on branch, not merged):**
- `backend/db/data/maps.go`: Added `DLCGraces` (105 DLC grace flags), `DLCRegions` (10 DLC overworld region IDs), fixed `IsDLCMapFlag()` range (removed incorrect 62800-62999)
- `backend/core/writer.go`: Added `AddRegionsToSlot()` — bulk region ID insertion with byte shifting and offset update
- `app.go`: Rewrote `revealDLCMap()` — now sets regions, graces, visibility flags, system flags, and items
- `spec/28-dlc-map-reveal.md`: Full research documentation

**Key remaining hypothesis:**
- CsDlc byte[1] (SotE entry flag) = 48 in reference save, 0 in ours. Game only sets it via proper DLC entry (Miquella's hand), not via teleport. This may be the master switch for DLC map rendering. Untested with current region/grace setup.

**Diagnostic tools created (tmp/, not committed):**
- `tmp/diag-dlc/main.go` — applies revealDLCMap to a save copy, compares with clean and reference slot
- Multiple Python analysis scripts used in-session for binary diff, flag scanning, region analysis
