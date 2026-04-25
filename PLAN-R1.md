# PLAN-R1 — Full Slot Rebuild (Invasion Regions Stage 2)

**Branch:** `feature/invasion-regions`
**Goal:** Replace shift-based `core.SetUnlockedRegions()` with a full
sequential slot rebuild matching `er-save-manager/parser/slot_rebuild.py`,
unblocking Stage 2 (write support) for the Invasion Regions feature.

**Decision (2026-04-26):** Option **B** — full struct rebuild. Each post-region
section gets a typed Go struct with `Read()`/`Write()`. More upfront work, but
opens the door to future features (weather edit, teleport, net_man tweaks…)
without re-engineering the parser.

**Reference:** `tmp/repos/er-save-manager/src/er_save_manager/parser/`
(`user_data_x.py`, `slot_rebuild.py`, `world.py`, `inventory.py`).

---

## Context — why we're here

- Stage 1 (read-only Invasion Regions UI) merged at `90db34b`.
- First Stage 2 attempt used **shift-based** in-place patching of the
  `unlocked_regions` block — corrupted saves on Steam Deck (game refused
  to load).
- Slack analysis (`spec/30-slot-rebuild-research.md`, commit `5c7131a`)
  measured 0–41 free regions across 11 test slots — **5/11 slots have
  zero slack**. Hybrid blob trim cannot deliver "Unlock All" (78 regions).
- Spec/22 + er-save-manager confirm `PlayerGameDataHash` size is
  *dynamic* (`slot_end - position_after_other_sections`), not fixed
  0x80. The hash blob holds ~48B of "real" hash + trailing zeros that
  function as tail padding. Sequential rebuild + tail-pad absorbs delta.

---

## Stop gates

| Gate | What |
|---|---|
| **After Step 12** | All section parsers in place; round-trip identity test for unmodified slot must produce byte-for-byte identical output. |
| **After Step 13** | Mutation test: append region to UnlockedRegions, rebuild, verify DLC/hash position shifts left by 4 bytes and tail gains 4 zero bytes. |
| **Step 15 (Steam Deck)** | Manual in-game test on PS4. If save corrupts, gather diff in `tmp/diag/`, identify which section bytes diverge, return to relevant struct parser. |

---

## Checklist

Legend: `[x]` done · `[~]` in progress · `[ ]` pending

### Foundation

- [x] **Step 0** — `RebuildSlot` scaffold (passthrough) + identity test
  - Commit: `d578d31`
  - Files: `backend/core/slot_rebuild.go`, `backend/core/slot_rebuild_test.go`
- [x] **Step 1** — `SaveSlot.SectionMap` populated during Read
  - Commit: `0a1a682`
  - Files: `backend/core/slot_rebuild.go` (+ `buildSectionMap`), `backend/core/structures.go`
- [x] **Step 2** — Hybrid blob rebuild with anchored DLC/hash
  - Commit: `7f3542a`
  - Files: `backend/core/slot_rebuild.go`
  - Note: superseded by Steps 4–13 (sequential rewrite); kept in history.
- [x] **Step 3** — Slack analysis (`spec/30-slot-rebuild-research.md`)
  - Commit: `5c7131a`
  - Files: `backend/core/slot_slack_test.go`, `spec/30-slot-rebuild-research.md`
- [x] **Step 4** — `SectionReader`/`SectionWriter` helpers
  - Commit: `d527ea7`
  - Files: `backend/core/section_io.go`, `backend/core/section_io_test.go`

### Section parsers (full struct, in canonical order)

- [x] **Step 5** — Horse + control byte + BloodStain + 2 unk u32 metadata
  - Commit: `8c938c3`
  - Reference: `parser/world.py` `RideGameData`, `BloodStain`
  - Files: `backend/core/section_types.go`, `backend/core/section_world.go`, `backend/core/section_world_test.go`
- [x] **Step 6** — MenuSaveLoad + TrophyEquipData + GaitemGameData + TutorialData
  - Commit: `0fd1d56`
  - Round-trip: menu=4104B, trophy=52B, gaitem=112008B, tutorial=1032B, total=117196B
  - Files: `backend/core/section_menu.go`, `backend/core/section_menu_test.go`
- [x] **Step 7** — Pre-event_flags scalar block + EventFlags + terminator
  - Commit: `cab4f4e`
  - 11 fields (3×u8, u32, i32, u8, u32, u32, u8, u32, u32) = 29B + event_flags 0x1BF99F + 1B terminator
  - Files: `backend/core/section_eventflags.go`, `backend/core/section_eventflags_test.go`
- [x] **Step 8** — Size-prefixed world sections (5×): field_area, world_area, world_geom_man, world_geom_man2, rend_man
  - Commit: `c8e8e11`
  - Round-trip totals: 1.3KB–7.2KB across 7 slots — confirms variable size between slots
  - Files: `backend/core/section_world_geom.go`, `backend/core/section_world_geom_test.go`
- [x] **Step 9** — PlayerCoordinates + spawn point + version-gated fields
  - Commit: `07066a8`
  - PlayerCoordinates struct = 61B (er-save-manager comment of 57B was stale); SpawnPointBlock 15B for v≥66
  - Files: `backend/core/section_player_coords.go`, `backend/core/section_player_coords_test.go`
- [x] **Step 10** — NetMan section
  - Commit: `58c0380`
  - Confirmed fixed 131,076B (u32 + 0x20000B opaque)
  - Files: `backend/core/section_netman.go`, `backend/core/section_netman_test.go`
- [ ] **Step 11** — Weather + Time + BaseVersion + SteamID + PS5Activity + DLC
  - Trailing fixed-size structs (12B + 12B + 16B + 8B + 32B + 50B)
  - Files: `backend/core/section_trailing.go`
- [ ] **Step 12** — PlayerGameDataHash (dynamic size) + Rest tail padding
  - Hash size = `slot_end - position_after_dlc`, not fixed 0x80
  - Update `offset_defs.go`: `HashSize`/`DlcSectionOffset` become defaults, not absolute
  - Files: `backend/core/section_hash.go`, `backend/core/offset_defs.go`

### Integration

- [ ] **Step 13** — `RebuildSlot` rewrite — sequential write of all sections
  - Drop hybrid blob path; iterate parsed structs, call `.Write()` per section
  - Identity test must still pass for unmodified slot
  - Mutation test: append a region, verify shift + tail-pad absorption
  - Files: `backend/core/slot_rebuild.go`
- [ ] **Step 14** — Wire `SetUnlockedRegions` to use full rebuild
  - Replace shift-based code in `core/writer.go` with `RebuildSlot` call + offset recompute
  - Round-trip test: Set→Save→Load→Get for PS4 + PC
  - Files: `backend/core/writer.go`, `tests/`

### Verification & ship

- [ ] **Step 15** — Manual Steam Deck test (STOP gate)
  - User test: Apply Unlock All Regions → save → load in-game (offline)
  - Verify: game starts, regions visible in multiplayer menu, no other regression
  - If corrupt: collect diff vs original in `tmp/diag-r1/`, locate diverging section
- [ ] **Step 16** — Re-enable Stage 2 UI write handlers
  - Restore `SetRegionUnlocked` + `BulkSetUnlockedRegions` in `app.go`
  - Restore checkboxes + per-area + global Unlock/Lock buttons in `WorldTab.tsx`
  - Regenerate Wails bindings (`wails generate module`)
  - Files: `app.go`, `frontend/src/components/WorldTab.tsx`, `frontend/wailsjs/go/main/*`
- [ ] **Step 17** — CHANGELOG / ROADMAP / merge proposal
  - Update CHANGELOG with Stage 2 details; mark ROADMAP `Invasion Regions Stage 2` as ✅
  - Update R-1 entry to reflect actual implemented scope
  - Final test sweep: `make build`, `go test ./backend/...`, `tsc --noEmit`, roundtrip
  - Propose merge of `feature/invasion-regions` → `main` (wait for OK)

---

## Resumption recipe

If a session is interrupted mid-plan:

1. Read this file; find the last `[x]` and the first `[ ]` step.
2. `git log feature/invasion-regions --oneline` — confirm last commit
   matches the last `[x]` step's expected hash.
3. Re-read the section in the corresponding "Files" list.
4. Continue with the first `[ ]` step.
5. Update the box from `[ ]` to `[x]` and append the commit hash after
   each completed step.

---

## Working notes

- Per-step commits keep diffs reviewable. Each section parser has a
  trivial round-trip test: read N bytes from real save, write to fresh
  buffer, expect byte-for-byte equality.
- `tests/data/pc/` and `tests/data/ps4/` are empty — use `tmp/save/`:
  `oisis_pl-org.txt` (PS4), `oisisk_ps4.txt` (PS4), `ER0000.sl2` (PC).
- Keep `feature/invasion-regions` branch state stable: every commit on
  the branch must build (`go build ./backend/... ./`) and pass
  `go test ./backend/...`. Pre-existing `TestBulkAddPerCategory` failure
  is unrelated and may be ignored.
- DO NOT regenerate Wails bindings until Step 16 — backend refactors
  shouldn't churn the frontend until the API is final.
