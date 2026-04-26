# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Branch: fix/dbtab-flag-filter-too-broad — Restore visibility of arrows/bolstering/crafting/most tools in Item Database

**Goal:** Regression fix. After the `dlc` (701 entries) and `stackable` (532 entries) flag sweeps, the "Cut & Ban-Risk" toggle in Settings (when off — the default for many users) was hiding **all** items with any flag in `Flags`. That meant nearly every entry in `arrows_and_bolts.go`, `bolstering_materials.go`, `crafting_materials.go`, and most of `tools.go` became invisible in both the Item Database tab and the inventory list.

**Root cause:** `DatabaseTab.tsx:120` and `InventoryTab.tsx:273` had:
```ts
if (!showFlaggedItems && item.flags?.length > 0) return false;
```
The check is intentionally inverse-of-toggle, but `length > 0` was too broad — it matched any flag including informational ones (`dlc`, `stackable`).

**Fix:** Both files now filter only on the four "risky" flag values that the toggle is actually designed for:
```ts
const RISKY_FLAGS = ['cut_content', 'ban_risk', 'pre_order', 'dlc_duplicate'];
if (!showFlaggedItems && item.flags?.some(f => RISKY_FLAGS.includes(f))) return false;
```

`WorldTab.tsx` and `WorldProgressTab.tsx` were not affected — they use the narrow check `flags?.includes('ban_risk')` directly.

**Impact:** With the toggle off, users see all stackable consumables/materials/arrows again. Risky items (cut content, ban risk, pre-order, duplicates) remain hidden as before — toggle behavior unchanged.

**Tests:** `cd frontend && npx tsc --noEmit` ✅, `make build` ✅.

### Branch: fix/ui-remove-redundant-hex — Drop hex-below-item-name (duplicates the optional ID column)

**Goal:** When the user enabled the "ID (HEX)" column toggle in Settings, the hex value rendered twice — once below the item name, once in the dedicated column. Per user feedback, remove the always-shown hex below the name. Users who want the hex still get it via the column toggle.

**Change:**
- `frontend/src/components/DatabaseTab.tsx` — the cell-below-name had a `showPreview ? <preview> : <hex>` ternary. Removed the else-branch so hex is no longer rendered as a fallback. Preview (e.g. `Heavy +25` for upgradeable weapons) still shows when applicable; otherwise the cell is empty.
- `frontend/src/components/InventoryTab.tsx` — the `<span>` rendering hex below the name was always shown. Removed entirely. The outer `flex flex-col gap-0.5` wrapper now has a single child (the name+flags row); kept as-is — cosmetic, no visual difference.

**No state, props, or styling changes elsewhere.** Settings checkbox `columnVisibility.id` continues to gate the optional "ID (HEX)" column unchanged.

**Tests:** `cd frontend && npx tsc --noEmit` ✅, `make build` ✅.

### Branch: feat/ban-risk-warning-modal — Warn when adding ban-risk-flagged items + minor icon downloader iteration

#### Part 1 — Ban-risk warning modal in `DatabaseTab`

**Goal:** Per user direction — when a user adds an item flagged `"ban_risk"` from the Item Database, gate the regular Add modal behind a warning that explains the risk (Easy Anti-Cheat detection if going online with cut-content / cheat-flagged items in inventory). Modal includes an "Ignore all ban risk warnings" checkbox so power users can opt out permanently.

**Change (`frontend/src/components/DatabaseTab.tsx`):**
- New state: `banRiskWarning: ItemEntry[] | null` and `ignoreBanRisk: boolean` (initialized from `localStorage["setting:ignoreBanRiskWarning"]`).
- `openModal(items)` now branches:
    - If `!ignoreBanRisk && items.some(i => i.flags?.includes('ban_risk'))` → set `banRiskWarning(items)` and return.
    - Otherwise → call `openConfirmModal(items)` (the previous body of `openModal`, refactored).
- New "Add Anyway" path: calls `openConfirmModal(items)` directly, bypassing the gate for this batch.
- New `handleIgnoreBanRiskChange(checked)`: writes the toggle to `localStorage` so it persists across sessions.

**Modal UI (red-themed, z-index above confirmModal):**
- Triangle alert icon + "Ban Risk Warning" header.
- Single-item path: bold item name in copy. Multi-item path: count + bullet list of all ban-risk items in the selection (scrollable when long).
- Checkbox: "Ignore all ban risk warnings" — bound to `ignoreBanRisk` state, persists immediately on toggle.
- Two actions: "Cancel" (closes warning, no add) vs "Add Anyway" (closes warning, opens regular Add modal).

**No backend changes required:** The `Flags []string` field on `ItemEntry` is already exposed via Wails bindings (used elsewhere in `WorldTab.tsx` / `WorldProgressTab.tsx` for similar filtering). Only frontend logic added.

**Tests:** `cd frontend && npx tsc --noEmit` ✅, `make build` ✅ (Wails generate-bindings + Vite + Go compile + macOS package — 9.3s).

**Manual verification deferred** to user — they confirmed previous UI changes manually after build green.

#### Part 2 — Icon downloader iteration 5 (minor refinements)

**Bugs fixed in `scripts/download_icons.go`:**
- Removed `"Bloody"` from affinity-strip list — it's NOT an infusion (like Heavy/Keen/Magic) but rather part of unique DLC weapon names ("Bloody Lance", "Bloody Buckler"). Stripping yielded base weapons, breaking lookups.
- Replaced `insertPossessive` (single-variant) with `insertPossessiveVariants` (returns 0..N variants). Handles double-`s` words like `Thopss` (from "Thops's") that the previous heuristic skipped.
- Added "un-stripped name" as a fallback variant for weapons/shields/ranged/arrows — when affinity stripping yielded a name and that fails, retries with the full original name (catches cases like "Flame_Art_Main_Gauche" where wiki actually has the full name).

**Imports:** 1 new icon — `frontend/public/items/sorceries/thopss_barrier.png` (Thops's Barrier, double-s possessive case proven).

**Remaining:** ~78 icons still not auto-fetchable — mostly cut-content gestures (no wiki page), DLC notes with combined wiki entries (e.g. Zorayas's & Rogier's Letter on a single page), items with multi-apostrophe names. Manual import or per-item WebFetch needed; defer.

### Branch: feat/stackable-flag-sweep — Add `stackable` flag to all items with MaxInventory > 1

**Goal:** Per user direction (audit cycle) — explicitly mark which items stack (vs single-instance) so future UI work can filter without re-deriving from `MaxInventory`. Adds `Flags: []string{"stackable"}` to every entry where `MaxInventory > 1`.

**Implementation:** Subagent ran a one-shot Go script that:
1. For each backend file, scanned every `0xXX:` map entry.
2. Matched `MaxInventory: <N>` with `N >= 2`.
3. Added/merged `"stackable"` into the existing `Flags` slice (preserves existing flags like `dlc`).
4. Skipped `MaxInventory == 1` entries (intrinsically non-stackable).

**Per-file count: 532 new `"stackable"` markers**

| File | +stackable |
|---|---:|
| `tools.go` | +281 (68 merged with existing flags, e.g. `dlc → dlc, stackable`) |
| `bolstering_materials.go` | +76 (2 merged) |
| `crafting_materials.go` | +71 (9 merged) |
| `arrows_and_bolts.go` | +64 (5 merged) |
| `key_items.go` | +40 (0 merged) |

**Other 13 backend files unchanged:** all entries have `MaxInventory == 1` (equipment, spells, gestures, ashes-of-war, talismans, etc.) — none stackable.

**Total merge cases: 84** — `Flags: ["dlc"]` became `Flags: ["dlc", "stackable"]`. No entries already contained `"stackable"`.

**Verification:**
- Independent script confirmed 0 `MaxInventory > 1` entries are missing the flag.
- Spot-checked diffs in 3 files — only Flags changed, no other fields touched.
- `go build ./backend/...` ✅, `go test ./backend/db/...` ✅.

**App impact:** Zero — flag is data-only marker until UI work consumes it.

### Branch: feat/icon-downloader-improvements — Improve `scripts/download_icons.go` + import 60 icons + DLC flag sweep across DB

**Two related changes shipped together** — both flow from the audit (`tmp/items_audit_report.md`) follow-up.

#### Part 1 — Icon downloader fixes + 60 new PNG imports

**Goal:** Pre-fix run had ~12% success rate on missing icons (15/130). Most failures were script bugs in URL/name generation, not genuinely-missing assets on `eldenring.wiki.gg`.

**Bugs fixed in `scripts/download_icons.go`:**
1. **Lowercase prepositions** — `toWikiName` capitalized "of"/"the"/"a"/"in"/"to"/"with"/"from"/"for"/"on"/"at"/"by"/"or"/"as"/"and". Wiki URLs keep these lowercase (e.g. `Rain_of_Fire`, not `Rain_Of_Fire`). Added `lowercaseWords` set, applied for non-leading positions.
2. **Missing prefixes** — `isCrafting` only tried `ER_Icon_Item_`/`ER_Icon_Tool_`. Added category-specific prefixes: `ER_Icon_Crafting_Material_`, `ER_Icon_Bolstering_Material_` (for `bolstering_materials/` paths). For `isTools` added `ER_Icon_Key_Item_` (cookbooks), `ER_Icon_Note_`, `ER_Icon_Info_` (notes/letters), `ER_Icon_Map_`, `ER_Info_` (combined entries like Zorayas's Letter).
3. **Possessive variant** — many local filenames strip apostrophes (`tibias_cookbook.png`) while wiki uses URL-encoded apostrophe (`Tibia%27s_Cookbook.png`). New `insertPossessive` helper inserts `%27` before trailing `s` of any 4+-char word ending in `s` (skipping double-`s` like "Hess"). Tries both original and possessive variants per prefix.
4. **`commons.wiki.gg` fallback** — wiki.gg serves some images on the commons subdomain. Added as host fallback after `eldenring.wiki.gg`.
5. **Affinity stripping** — added missing `Bloody` (DLC infusion variant for some weapons). Added multi-word `Flame_Art` BEFORE single-word `Flame` to avoid prefix-shadowing on names like `Flame_Art_Main_Gauche`.

**Result:** Success rate climbed from ~12% to ~46% (60/130 net icons imported across 4 iterations). Remaining ~80 are mostly genuinely-not-on-wiki assets (cut content like `The Carian Oath` / `Fetal Position`, obscure DLC notes with combined entries on wiki, items with multi-apostrophe names).

**Imports** (sample — 60 total): `revered_spirit_ash.png`, `scadutree_fragment.png`, 7 DLC crafting materials (`frozen_maggot`, `ghostflame_bloom`, etc.), 5 DLC incantations (`furious_blade_of_ansbach`, `dragonbolt_of_florissax`, etc.), 6 sorceries (`rain_of_fire`, `glintstone_nail`, etc.), 5 Prattling Pates, ~15 DLC quest items (cookbooks, keys, maps, notes), various others.

**Not committed:** `missing_icons.txt` (build artifact — regenerated on demand by re-running the script).

#### Part 2 — DLC flag sweep across entire item database

**Goal:** Mark every Elden Ring DLC (Shadow of the Erdtree) item with `Flags: []string{"dlc"}` so future UI can filter. Previously only 16 entries had `"dlc"` (the 14 added in `feat/add-missing-tools-63` plus 2 Crystal Tears in `key_items.go`).

**Implementation:** Subagent ran a one-shot Go script (`/tmp/mark_dlc.go`) that:
1. Loaded all DLC reference files from `tmp/repos/er-save-manager/.../DLC/**/*.txt` and built a decimal-ID → category map.
2. For each backend item, computed decimal from hex (after stripping the per-category prefix), looked up in DLC set, edited Flags field if matched.
3. Preserved existing flags — when an entry already had flags, appended `"dlc"` to the slice; otherwise added `Flags: []string{"dlc"}`.
4. Skipped entries already containing `"dlc"`.

**Per-file count: 701 new `"dlc"` markers** distributed:

| File | +dlc | File | +dlc |
|---|---:|---|---:|
| `ashes.go` | +220 | `arms.go` | +29 |
| `weapons.go` | +73 | `incantations.go` | +27 |
| `tools.go` | +72 | `aows.go` | +25 |
| `helms.go` | +43 | `sorceries.go` | +15 |
| `chest.go` | +43 | `ranged_and_catalysts.go` | +11 |
| `key_items.go` | +43 | `shields.go` | +10 |
| `talismans.go` | +39 | `crafting_materials.go` | +9 |
| `legs.go` | +30 | `arrows_and_bolts.go` | +5 |
| | | `gestures.go` | +5 |
| | | `bolstering_materials.go` | +2 |

Plus 16 pre-existing `"dlc"` flags = **717 total DLC entries marked**.

**Prefix conventions discovered (verified against reference):**
- `weapons.go`/`shields.go`/`arrows_and_bolts.go`/`ranged_and_catalysts.go`: **no prefix** — backend stores raw decimal as hex (e.g. `0x0016E360` = decimal 1500000 = "Main-gauche").
- `helms.go`/`chest.go`/`arms.go`/`legs.go`: prefix `0x10000000`.
- `talismans.go`: prefix `0x20000000`.
- `aows.go`: prefix `0x80000000`.
- All Goods (`tools.go`/`key_items.go`/`ashes.go`/`crafting_materials.go`/`bolstering_materials.go`/`cookbooks.go`/`gestures.go`/`sorceries.go`/`incantations.go`): prefix `0x40000000`.

**Flag merge cases:** 1 entry — `tools.go` `Keep Wall Key` → `Flags: []string{"cut_content", "ban_risk", "dlc"}` (was `cut_content, ban_risk`).

**Skipped:** `cookbooks.go` (uses `CookbookData` struct keyed by event-flag IDs, not item IDs — no Flags field). `gestures.go` `AllGestures` slice (`GestureDef` struct, separate from the `Gestures` ItemData map; agent only modified the latter).

**Tests:** `go build ./backend/...` ✅, `go test ./backend/db/...` ✅, `go test ./tests/roundtrip_test.go` ✅, `go vet ./backend/...` ✅.

**App impact:** Zero — `Flags` slice is data-only until UI work consumes it. The new `"dlc"` flag is a marker for future filter functionality (per user direction in this audit cycle: *"chcę mieć te informacje w db, mogą być na razie jako komentarze, później zaimplementujemy w apce"*).

### Branch: feat/add-missing-tools-63 — Add 63 missing consumables/tools/tears + introduce `dlc` flag

**Goal:** Item-DB audit (`tmp/items_audit_report.md`) flagged 63 items present in `er-save-manager` reference but absent from our backend `tools.go` (and 2 DLC Crystal Tears that belong in `key_items.go` per project convention). Filling these gaps unblocks players who want to add them via the Item Database UI.

**Cross-validation:**
- All 63 IDs converted from reference decimal → hex with `0x40000000` prefix and verified.
- Icons confirmed present for 54 of 63 entries (sampled `find frontend/public/items -iname "<keyword>*"`).
- 9 Prattling Pates have no icons yet — placeholder paths `items/tools/multiplayer/prattling_pate_*.png` written so a later icon import slots in without code change. Frontend `DatabaseTab` already gracefully handles missing icons (broken-icon "?" placeholder).
- Empty Crimson/Cerulean flask variants (26) reuse the filled-flask icons (filled flasks already share one icon across all upgrade levels).

**Change:**
- `backend/db/data/tools.go` — added 61 entries:
    - Batch 1 (12) — base game consumables: Boiled Prawn, Neutralizing/Thawfrost/Preserving/Rejuvenating/Clarifying Boluses, Pickled Turtle Neck, Gold-Pickled Fowl Foot, Soft Cotton, Baldachin's Blessing, Poison Spraymist, Acid Spraymist.
    - Batch 2 (27) — empty flask variants: Wondrous Physick (Empty), Crimson Tears (Empty) +0..+12, Cerulean Tears (Empty) +0..+12.
    - Batch 3 (9) — Prattling Pates: Hello / Thank you / Apologies / Wonderful / Please help / My beloved / Let's get to it / You're beautiful / Lamentation (DLC).
    - Batch 4 (13) — DLC consumables: Lulling Branch, Dragonscale Flesh, 5 Pickled Livers (Spellproof/Fireproof/Lightningproof/Holyproof/Opaline), Well-Pickled Turtle Neck, Sacred Bloody Flesh, Silver/Golden Horn Tender, Innard Meat, Horned Bairn.
- `backend/db/data/key_items.go` — added 2 entries: Bloodsucking Cracked Tear (`0x401EAFAA`), Glovewort Crystal Tear (`0x401EAFB4`). Routed to `key_items.go` (not `tools.go` per ref) because the project's existing Crystal Tears all live in `key_items.go` with icons under `items/key_items/`.

**New flag value `"dlc"`:** introduced for the 14 DLC entries added in this branch (batches 3-DLC + 4 + 2 Crystal Tears). Existing flag vocabulary was `cut_content`, `ban_risk`, `dlc_duplicate`, `pre_order`. The new `dlc` flag marks legitimate DLC content (not duplicates) — used for filtering in future UI work. **No app changes in this branch** — flag is data-only for now.

**Defaults applied (all new entries):**
- Most consumables: `MaxInventory: 99, MaxStorage: 600` (matches existing convention for Stanching Boluses, Pickled livers, etc.).
- Wondrous Physick (Empty): `1 / 0` (matches filled variant).
- Crystal Tears: `1 / 0` (matches existing Crystal Tear entries in `key_items.go`).
- Empty flasks (Crimson/Cerulean): `99 / 600` (matches filled-flask convention; quirky for "1 sacred flask total in game" but not changing existing pattern in this branch).

**Intentionally NOT done in this branch:**
- 101 "category mismatches" reported by audit (Remembrances, Maps, Notes, Letters classified as `Consumables.txt` in reference but as `key_items.go` in our backend). Our project aligns categories with the in-game UI Inventory tabs; reshuffling would diverge from UI. **Confirmed false-positive.** Memory updated.
- 2 spell category swaps reported by audit (`Death Lightning`, `Night Maiden's Mist`). Verified against Fextralife: backend was already correct — the reference `Magic.txt` tags are wrong for these entries. **Confirmed false-positive.** Memory updated.
- 3 cut-content entries flagged in `gestures.go` (`?GoodsName?`, Carian Oath, Fetal Position) are intentionally retained with `cut_content`+`ban_risk` flags per documented design. **Confirmed false-positive.**

**Tests:** `go build ./backend/...` ✅, `go test ./backend/db/...` ✅, `go test ./tests/roundtrip_test.go` ✅.

### Branch: feat/add-missing-armor-12 — Add 12 base-game armor pieces missing from item DB

**Goal:** Item-DB audit (`tmp/items_audit_report.md`) flagged 12 base-game armor entries present in `er-save-manager` reference but absent from our backend. All 12 already had matching icon assets shipped under `frontend/public/items/{head,chest,arms}/` from a previous icon import — only the Go entries were missing.

**Cross-validation:**
- All 12 IDs verified by computing `decimal // 0x10000000 prefix` from the audit report.
- Each item belongs to a known set whose other pieces already exist in our DB (set-completion check):
    - **Scaled** set — `Scaled Armor` (chest) was missing; `Scaled Helm`, `Scaled Armor (Altered)`, `Scaled Gauntlets`, `Scaled Greaves` already present.
    - **Sanguine Noble** set (DLC) — `Hood` (head) + `Robe` (chest) missing; `Waistcloth` (legs) already present.
    - **Fire Prelate** set — `Helm` was missing; chest/arms/legs present.
    - **Elden Lord** set — `Crown` was missing; chest/arms/legs present.
    - **Depraved Perfumer** set — `Gloves` (arms) missing; headscarf/robe/trousers present.
    - **Prophet** set — `Blindfold` (head) + `Robe` (chest, base, non-altered) missing; `(Altered)` and `Trousers` present.
    - **Imp Head (Elder)** — joins the existing `Cat`/`Fanged`/`Long-Tongued`/`Corpse`/`Wolf`/`Lion` series.
    - **Mushroom** set — `Head` + `Body` missing; `Crown` (different item), `Mushroom Arms`, `Mushroom Legs` already present.

**Change:**
- `backend/db/data/helms.go` — added 7 entries: Great Horned Headband (`0x100493E0`), Sanguine Noble Hood (`0x1004E200`), Fire Prelate Helm (`0x10057E40`), Elden Lord Crown (`0x100704E0`), Prophet Blindfold (`0x100975E0`), Imp Head (Elder) (`0x10108E48`), Mushroom Head (`0x10113E10`).
- `backend/db/data/chest.go` — added 4 entries: Scaled Armor (`0x100138E4`), Sanguine Noble Robe (`0x1004E264`), Prophet Robe (`0x10097E14`), Mushroom Body (`0x10113E74`).
- `backend/db/data/arms.go` — added 1 entry: Depraved Perfumer Gloves (`0x10083E28`).

All entries: `MaxInventory: 1, MaxStorage: 1, MaxUpgrade: 0`, `Category: "head"|"chest"|"arms"`, IconPath references the existing PNG that was already shipped.

**Tests:** `go build ./backend/...` ✅, `go test ./backend/db/...` ✅, `go test ./tests/roundtrip_test.go` ✅.

### Branch: fix/wails-dev-restart-loop — UI preview mode without save + embed dotfile fix

**Two unrelated changes shipped together on the same branch** (per user direction).

#### Part 1 — Wails dev restart-loop fix

**Goal:** `make dev` window kept closing and reopening on its own, even without code changes. Logs showed Vite HMR reload + `[AssetServer] Unable to write content index.html: request has been stopped` × N + `Done.`, then app restart.

**Root cause:** `main.go:13` had `//go:embed all:frontend/dist`. The `all:` prefix includes hidden files (dotfiles). `frontend/dist/` contains 6 `.DS_Store` files which macOS Finder/Spotlight touch periodically. `wails dev` watches the embed source tree — every `.DS_Store` mtime bump triggered a Go rebuild + app restart cycle.

**Change:**
- `main.go:13`: `//go:embed all:frontend/dist` → `//go:embed frontend/dist`. Without `all:`, Go's default embed semantics exclude `.` and `_` prefixed files. Maps, icons, `index.html`, JS/CSS bundles are normal files — still embedded. Production binary identical in content (118 MB after rebuild).

**Why not delete `.DS_Store`:** macOS Finder recreates them on first directory browse — would not be a stable fix. The embed-directive change makes it irrelevant whether they exist.

#### Part 2 — Preview mode without loaded save

**Goal:** With no save file loaded, the editor previously showed all 5 tabs but each one was an empty "No Save File" placeholder. Wanted: 3 tabs (Character / Inventory / Settings) with view-only content, so users can browse the appearance presets and item database before opening a save.

**Change:**
- `frontend/src/App.tsx`:
    - `tabs` array gated on `platform`: 5 tabs when save loaded, 3 tabs (`character`, `inventory`, `settings`) otherwise. Hides `world` and `tools` until a save is loaded.
    - Replaced the centered "No Save File" placeholder with a slim "Preview mode — load a save file to enable editing" banner + `Open Save` button on top of the tab content.
    - In preview mode, `character` renders `<AppearanceTab readOnly />`, `inventory` renders `<DatabaseTab readOnly />`. `settings` continues to use the existing branch unchanged (Steam ID already self-disables when `!platform`).
- `frontend/src/components/AppearanceTab.tsx`:
    - New `readOnly?: boolean` prop. When `true`: skip `GetFavoritesStatus()` call (avoids backend error), hide preset checkbox overlay, hide action buttons block (Apply / Add to Mirror), hide existing Mirror Favorites section. Image preview/zoom remains active. Description text adjusted to "Click image to preview. Load a save file to apply presets to a character."
- `frontend/src/components/DatabaseTab.tsx`:
    - New `readOnly?: boolean` prop. When `true`: omit checkbox column from header and rows, hide "Add Selected (N)" button, adjusted `colCount` accordingly. Search, sort, category filter, ItemDetailPanel preview all still work.

**Notes:**
- `AppearanceTab.tsx` was previously orphan code (imported nowhere). Now wired into App.tsx for the preview Character tab.
- `DatabaseTab` is the same component used inside the loaded-save Inventory tab (database sub-view), reused in both modes — single source of truth.
- No backend / Wails-binding changes. No regression risk in save-loaded code paths (the existing 5-tab branch is untouched).

**Tests:** `make build` ✅ (frontend tsc + vite build + go compile + macOS package). Manual UI verification done by user.

### Branch: fix/gourmet-scorpion-stew-limits — Correct stack limits for Gourmet Scorpion Stew

**Goal:** Fix MaxInventory / MaxStorage for `0x401E8933` Gourmet Scorpion Stew. Database had `99 / 600` (default consumable values) — actual game limits are `1 / 1`. Bug originated when the entry was first imported with placeholder defaults; never verified against wiki.

**Cross-validation:**
- **Fextralife wiki** (Gourmet Scorpion Stew page): "You can hold up to 1 in inventory" + "You can store up to 1 in your item box" — explicit non-stackable. Note from page also distinguishes regular Scorpion Stew (which sends overflow to storage) vs Gourmet (strict 1-per-location).
- Symmetric with regular Scorpion Stew (`0x401E8932`, also `1 / 1`) added in previous commit `ac18cd7`.

**Change:**
- `backend/db/data/tools.go`: `0x401E8933` Gourmet Scorpion Stew — MaxInventory `99 → 1`, MaxStorage `600 → 1`. Other fields unchanged.

**Tests:** `go build ./backend/...` ✅, `go test ./backend/...` ✅, `go test ./tests/roundtrip_test.go` ✅, `npx tsc --noEmit` ✅.

### Branch: feat/add-scorpion-stew — Add missing regular Scorpion Stew

**Goal:** Add `0x401E8932` "Scorpion Stew" (regular). We already had Gourmet variant (`0x401E8933`); the canonical non-Gourmet was missing.

**Cross-validation:**
- **er-save-manager**: `DLC/DLCGoods/DLCConsumables.txt:37` — `2001200 Scorpion Stew`. `2001200 dec = 0x1E8932` → `0x401E8932`.
- **Elden-Ring-Save-Editor** (Final.py): `goods.json:33` — `"Scorpion Stew": "32 89 1E B0"` → matches.
- **Fextralife wiki:** Item Type "Consumable", +10% physical damage negation + 8 HP/s regen for 60s, MaxInv 1, MaxStorage 1. Obtained from Hornsent Grandam (infinite supply on revisit).

**Change:**
- `backend/db/data/tools.go`: added `0x401E8932: {Name: "Scorpion Stew", Category: "tools", MaxInventory: 1, MaxStorage: 1, MaxUpgrade: 0, IconPath: "items/tools/consumables/scorpion_stew.png"}` directly above the existing Gourmet entry. Icon already shipped.

**Intentionally NOT added — ESM duplicate IDs:**
- `0x401E8934` (ESM `2001202 Scorpion Stew`) and `0x401E8935` (ESM `2001203 Gourmet Scorpion Stew`) appear in `AllGoods.txt` and `DLCConsumables.txt` but have no Fextralife page or distinct in-game role. Likely cut content / quest-state variants. Adding them blindly would pollute the Item Database UI; defer until function is verified.

**Counts after:** `tools.go` 291 (+1).

**Tests:** `go build ./backend/...` ✅, `go test ./backend/...` ✅, `go test ./tests/roundtrip_test.go` ✅, `npx tsc --noEmit` ✅.

### Branch: feat/add-blessing-of-marika — Add missing DLC consumable

**Goal:** Add `0x401E8804` "Blessing of Marika" to the item database. Item is missing from our DB despite being a known DLC consumable (Shadow of the Erdtree); icon already shipped at `frontend/public/items/tools/consumables/blessing_of_marika.png` from a previous icon import.

**Cross-validation:**
- **er-save-manager** (priority 1): `DLC/DLCGoods/DLCConsumables.txt:28` — `2000900 Blessing of Marika`. `2000900 dec = 0x1E8804`; with our `0x40000000` base prefix → `0x401E8804`.
- **Elden-Ring-Save-Editor** (Final.py): `goods.json:22` — `"Blessing of Marika": "04 88 1E B0"`. Bytes read LE = `0xB01E8804`; strip top nibble (item-type marker) → `0x01E8804` → matches.
- **ER-Save-Editor** (Rust): not in DB (predates SoE additions). Not blocking — two refs agree.
- **Fextralife wiki:** Item Type "Consumable", full HP restore + clears all status ailments, 3 per playthrough (Church of Consolation / Fort of Reprimand / Two Tree Sentinels in Scaduview), no respawn at Grace. MaxInventory 1, MaxStorage 600.

**Change:**
- `backend/db/data/tools.go`: added `0x401E8804: {Name: "Blessing of Marika", Category: "tools", MaxInventory: 1, MaxStorage: 600, MaxUpgrade: 0, IconPath: "items/tools/consumables/blessing_of_marika.png"}` next to other DLC consumables in the `0x401E88xx` block.

**Counts after:** `tools.go` 290 (+1).

**Tests:** `go build ./backend/...` ✅, `go test ./backend/...` ✅, `go test ./tests/roundtrip_test.go` ✅, `npx tsc --noEmit` ✅.

### Branch: fix/call-of-tibia-category — Revert Call of Tibia mismove

**Goal:** Undo the mistaken move of `0x401E90CE` Call of Tibia from `tools.go` → `incantations.go` in `0834d8b`. The previous fix justified the move with "mirrors prior Furious Blade of Ansbach fix in 1ad864e" — but the precedents do not match. Furious Blade of Ansbach has `IconPath: items/incantations/...` (data self-consistent → keep in incantations). Call of Tibia has `IconPath: items/tools/consumables/call_of_tibia.png` and is listed in `er-save-manager/.../DLC/DLCGoods/DLCConsumables.txt:84` — both sources say "DLC consumable". In game it is a Sky Chariot summon item dropped by Tibia Mariner, used from inventory. Not an incantation.

**Change:**
- `0x401E90CE` Call of Tibia: `incantations.go` → **`tools.go`** with `Category: "tools"` (icon path unchanged, was already `tools/consumables/`).

**Counts after:** `tools.go` 289 (+1), `incantations.go` 128 (−1).

**Tests:** `go build ./backend/...` ✅, `go test ./backend/...` ✅, `go test ./tests/roundtrip_test.go` ✅, `npx tsc --noEmit` ✅.

### Branch: docs/spec-map-reveal — Map reveal documentation overhaul

**Goal:** align `spec/` with the actual `RevealAllMap` / `RemoveFogOfWar` / `SetUnlockedRegions` implementation. The old `spec/27-fog-of-war.md` advertised "fill bitfield 0xFF" as the recommended map-reveal path — the editor has not used that approach for months. `spec/11-regions.md` was a placeholder ("requires verification") and warned about a 10–20-region byte-shift crash that was eliminated by `RebuildSlot` in R-1 Step 14.

**Changes:**
- **Renamed** `spec/27-fog-of-war.md` → `spec/27-map-reveal.md` (git-tracked rename, history preserved). Rewritten as a 4-layer model: Unlocked Regions / Detailed Bitmap (event flags 62xxx + Map Fragment items + system flags 62000/62001/62002/82001/82002) / DLC Cover Layer (cross-ref `spec/29`) / FoW bitfield. Each layer maps to its `app.go` / `core` entry point. FoW is documented as `RemoveFogOfWar` (separate user action, not part of `RevealAllMap`) — corrected after audit found the function does exist and is wired into `WorldTab.tsx` / `WorldProgressTab.tsx`.
- **Rewrote** `spec/11-regions.md` — now documents the binary format precisely (`u32 count + count × u32`, little-endian), points at `core.SetUnlockedRegions` as the only supported entry point, lists what regions do (fast travel + multiplayer state) and explicitly what they do NOT do (no map texture, no FoW, no Cover Layer). Removed the outdated "max 10–20 regions" warning.
- **Updated** `spec/README.md` index — refreshed entry #11, added missing entries for #27, #29, #30.
- **Updated** code/doc references — `app.go::RemoveFogOfWar` comment + 3 `ROADMAP.md` mentions now point at `27-map-reveal.md`.

**Tests:** `go build ./backend/...` ✅, `go vet ./backend/...` ✅, `go build .` (root incl. `app.go`) ✅. Pre-existing `tmp/`-script build errors are unrelated to this change.



**Goal:** Cross-validate item category assignments across `key_items.go`, `tools.go`, `crafting_materials.go`, `bolstering_materials.go` against three independent sources — `er-save-manager`, `ER-Save-Editor` (Rust), `Elden-Ring-Save-Editor` (Final.py / Goods/*.txt) — and Fextralife wiki.

**Process:** Spawn agent for full cross-check → 91 candidate "miscategorisations" reported. Hand-verified each finding by grep + wiki lookup; agent's count was inflated ~10× — all runes (Golden, Hero's, Numen's, Lord's) live correctly in `bolstering_materials.go`, not scattered as agent claimed. Talisman Pouch confirmed as Key Item per wiki, not Upgrade Material per ESM. Tools/Consumables boundary kept arbitrary on our side (no `consumables.go` split).

**Real bugs found (6 entries):**

A) Internal Category mismatch (file vs `Category` field):
- `0x401E90CE` Call of Tibia — was in `tools.go` with `Category: "incantations"`. **Moved to `incantations.go`** (mirrors prior Furious Blade of Ansbach fix in `1ad864e`).
- `0x400000B6` Furlcalling Finger Remedy — was in `key_items.go` with `Category: "tools"`. **Fixed `Category` to "key_items"** (file is correct, field was wrong).

B) ESM-confirmed shifts (3 ref repos + wiki agree):
- `0x4000085C` Margit's Shackle: `key_items.go` → **`tools.go`** (ESM Tools.txt:1; tactical multiplayer/boss tool)
- `0x40000866` Mohg's Shackle: `key_items.go` → **`tools.go`** (ESM Tools.txt:2)
- `0x40000870` Pureblood Knight's Medal: `key_items.go` → **`tools.go`** (ESM Tools.txt:3; multiplayer summon tool)
- `0x40002005` Sewer-Gaol Key: `tools.go` → **`key_items.go`** (ESM KeyItems.txt:92; it's a door key)

**Counts after:** `tools.go` 288 (+1), `key_items.go` 388 (−2), `incantations.go` 129 (+1).

**Rejected agent suggestions (after verification):**
- ❌ Create `consumables.go` (245 items): Tools/Consumables boundary in-game is arbitrary; split would force frontend Item Database filter refactor with low ROI.
- ❌ Add 470 "missing" items: most are flask variants (already covered), DLC merchant junk, covenant duplicates.
- ❌ Audit 744 "extras": legit flask variants and already-flagged cut content.
- ❌ Move Talisman Pouch to consumables: wiki confirms it's a Key Item.

**Tests:** `go build ./backend/...` ✅, `go test ./backend/...` ✅, `go test ./tests/roundtrip_test.go` ✅, `npx tsc --noEmit` ✅, `make build` ✅.

### Branch: fix/bosses-data-correctness — Boss name/region accuracy

**Goal:** Cross-validate `bosses.go` (110 entries) against three independent sources — `er-save-manager` (Python, flag-based), `ER-Save-Editor` (Rust, arena-flag-based), and Fextralife wiki — and fix wording where the references plus wiki agreed our entry was wrong.

**Process:** Spawn agent for diff against both ref repos → 91 discrepancies found (6 MAJOR-NAME, 6 MINOR-NAME, 79 MINOR-REGION). Web-verified each MAJOR/MINOR-NAME case. Most apparent "errors" turned out to be ref-repo issues (e.g. `er-save-manager` ships "Lorretta" / "Bayle, the Dread" / "Spirit-Caller Snail" for the Spiritcaller Cave fight — all wrong per wiki). 79 region differences kept as-is: our specific dungeon names ("Tombsward Catacombs") are more useful than ESM's broad regions ("Weeping Peninsula").

**Corrected — `bosses.go` (5 entries):**
- `9210` "Crucible Knight Ordovis" → **"Crucible Knight & Crucible Knight Ordovis"** (Auriza Hero's Grave is a duo fight)
- `9238` "Crystalians" → **"Crystalian Duo"** (Academy Crystal Cave = Spear + Staff; consistent with `9265` Crystalian Duo)
- `9239` "Kindred of Rot" → **"Kindred of Rot Duo"** (Seethewater Cave has two)
- `9241` "Omenkiller & Miranda" → **"Omenkiller & Miranda, the Blighted Bloom"** (Miranda's full Fextralife name)
- `9246` "Putrid Crystalians" → **"Putrid Crystalian Trio"** (Sellia Hideaway has three; canonical name)

**Confirmed correct (no change despite ref-repo disagreement):**
- `9119` Loretta — wiki spells "Loretta" (one t)
- `9163` Bayle the Dread — no comma per wiki
- `9173` Godskin Apostle / Divine Tower of Caelid — wiki confirms tower-specific location
- `9248` Godskin Apostle & Noble (Spiritcaller's Cave) — Snail is summoner only, real fight is Apostle + Noble

**Tests:** `go build ./backend/...` ✅, `go test ./backend/...` ✅, `go test ./tests/roundtrip_test.go` ✅ (4/4), `npx tsc --noEmit` ✅.

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
