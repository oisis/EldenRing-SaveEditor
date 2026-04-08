# Project Roadmap: ER-Save-Editor-Go (100% Python Parity)

> **Status:** 🛠 Phase 9 In Progress | **Source of Truth:** `tmp/repos/Elden-Ring-Save-Editor` (Python) | **Test Strategy:** Round-trip & Golden Files

## Phase 1: Environment & Infrastructure ✅
- [x] **1.1. Go Initialization**
    - [x] `go mod init github.com/oisis/EldenRing-SaveEditor` (Go 1.26+)
    - [x] Setup project structure: `backend/core`, `backend/db`, `backend/vm`
- [x] **1.2. Wails Setup**
    - [x] `wails init` with React + TypeScript template
    - [x] Configure `wails.json` (window size, title, etc.)
- [x] **1.3. Automation & Scripts**
    - [x] Create `Makefile` for `build`, `test`, `lint`, and `extract-data`
    - [x] Setup `golangci-lint` configuration
- [x] **1.4. Test Infrastructure**
    - [x] Setup `tests/data` for Golden Files
    - [x] Implement `RoundTrip_test.go` skeleton for bit-perfect verification

## Phase 2: Data Extraction (Python -> Go) ✅
- [x] **2.1. Extraction Tooling**
    - [x] Create Go script to parse `tmp/repos/Elden-Ring-Save-Editor/src/Resources/Json/*.json` and generate Go maps.
    - [x] Use `Final.py` as the primary reference for binary logic and offsets.
- [x] **2.2. Constant Extraction**
    - [x] Port item names and IDs from JSON files to Go maps.
    - [x] Port grace and boss flag IDs from Python logic.
- [x] **2.3. Stats & Classes**
    - [x] Port HP/FP/SP tables and base class stats matching Python implementation.

## Phase 3: Binary Core (The "encoding/binary" Layer) ✅
- [x] **3.1. Crypto Implementation**
    - [x] Port AES-128-CBC logic for PC saves (`backend/core/crypto.go`)
    - [x] Implement MD5 and SHA256 checksum logic (1:1 with Python)
- [x] **3.2. Binary Structures (PC)**
    - [x] Define `BND4` container struct
    - [x] Define `PCSaveSlot` and `PCUserData` structs matching Python offsets
    - [x] **Test:** Round-trip validation for PC raw saves
- [x] **3.3. Binary Structures (PlayStation)**
    - [x] Define `PSSaveSlot` and `PSUserData` structs
    - [x] Implement raw binary reading/writing for Save Wizard compatibility
    - [x] **Test:** Round-trip validation for PS raw saves
- [x] **3.4. SteamID Logic**
    - [x] Implement SteamID detection and modification in `UserData10`
    - [x] **Test:** Verify MD5 checksum recalculation after SteamID change
- [x] **3.5. Backup System**
    - [x] Implement automatic `.bak` creation with timestamps before any write

## Phase 4: Logic & ViewModel (Go Backend) ✅
- [x] **4.1. Save Manager**
    - [x] Implement `LoadSave(path)` with auto-detection (PC vs PS)
    - [x] Implement `SaveFile()` with integrity check (Round-trip Validation)
- [x] **4.2. ViewModel Mapping**
    - [x] Map raw bytes to `CharacterViewModel` (Name, Stats, Souls)
    - [x] Map `EventFlags` (bits) to boolean flags for Graces/Bosses
- [x] **4.3. Validation Logic**
    - [x] Implement stat recalculation (Level = sum of attributes - 79)
    - [x] Implement Weapon Matchmaking Level scanner (Somber vs Normal)
    - [x] **Test:** Unit tests for level calculation formula matching Python logic

## Phase 5: UI Implementation (Wails Frontend) ✅
- [x] **5.1. Base Layout**
    - [x] Sidebar navigation and Title bar (Tailwind CSS v4)
    - [x] Dark/Light mode toggle (matching Python version aesthetic)
- [x] **5.2. General Tab**
    - [x] Character selection, Name edit, SteamID edit, Stats edit
- [x] **5.3. Inventory Tab**
    - [x] Item list with search, "Bulk Add" buttons
- [x] **5.4. World Progress Tab**
    - [x] Tree view for Graces and Bosses (grouped by region)

## Phase 6: Advanced Tools ✅
- [x] **6.1. Character Importer**
    - [x] Logic for copying slot + `ProfileSummary` between files
- [x] **6.2. Slot Management**
    - [x] Activate/Deactivate slots in `UserData10`

## Phase 7: Quality & Finalization ✅
- [x] **7.1. Testing**
    - [x] Unit tests for binary parsing using files from `tmp/save` (2x PS4, 1x PC)
    - [x] Integration tests for SteamID modification
- [x] **7.2. Packaging**
    - [x] `make` for Windows (.exe), macOS (.app), Linux

## Phase 8: Item Browser (New Feature) ✅
- [x] **8.1. Binary Inventory Parsing**
    - [x] Implement sequential `GaItem` reader in `backend/core/reader.go`
    - [x] Handle dynamic sizes for Weapons (21b) and Armor (16b)
- [x] **8.2. Data Mapping & ViewModel**
    - [x] Map `GaItem` handles to names using `backend/db`
    - [x] Expose `Inventory` list in `CharacterViewModel`
- [x] **8.3. UI Implementation**
    - [x] Create searchable inventory list in `InventoryTab.tsx`
    - [x] Add granular category filters (Bows, Shields, Staffs, Seals, Armor parts, etc.)
    - [x] Implement quantity editing with `MaxInventory` and `MaxStorage` validation
    - [x] Implement multi-selection and bulk "Add Selected" action

## Phase 9: PvP Optimization & Advanced Tweaks (In Progress) 🛠
- [ ] **9.1. Faster Invasions (Meliodas Method)**
    - [x] Research exact offsets for `NetworkParam` in `UserData11` (Regulation block)
    - [x] Implement matchmaking interval reduction (20s -> 4s)
    - [x] Implement search scope expansion (Global region polling)
    - [x] Add "Enable Faster Invasions" toggle in Settings tab
- [x] **9.2. World Progress Automation**
    - [x] Group Sites of Grace by region (Limgrave, Liurnia, Caelid, etc.)
    - [x] Fix scrolling issue on World Progress tab
    - [x] Add interactive map thumbnails for each region
    - [x] Add "Unlock All Invasion Regions" button
    - [x] Add "Activate All Summoning Pools" button
- [ ] **9.3. Matchmaking Safety Tools**
    - [x] Implement Weapon Level scanner and "Safe De-leveling" logic
    - [x] Add "Global Weapon Level" setting for bulk item addition (Future)

## Phase 10: Item Icons & Visual Assets ✅
- [x] **10.1. Icon Directory Structure**
    - [x] Setup `frontend/public/items/` with flat subdirectories for categories
- [x] **10.2. Database Integration**
    - [x] Add static `IconPath` to database for all 4000+ items
    - [x] Implement `MaxUpgrade` metadata for weapons and spirit ashes
    - [x] Unify `MaxInventory` and `MaxStorage` across all categories
- [x] **10.3. Asset Coverage & Validation**
    - [x] Audit missing icons for DLC items
    - [x] Implement fallback icon (placeholder) for missing assets
    - [x] Migrate all icons to flat structure and remove legacy folders
- [x] **10.4. UI Integration**
    - [x] Display icons in Inventory, Storage, and Database tables using static paths
    - [x] Implement "Icon Popover" for high-resolution preview
    - [x] Add "Upgrade" column with sorting capability
    - [x] Implement "Add Item" modal with upgrade level selection (+0 to +25/+10)
    - [x] Fix UI scaling issues (removed max-w-5xl and fixed heights for full-width/height support)
    - [x] Improve scrollbar visibility and fix nested scrolling conflicts
    - [x] Enable window maximization on macOS (explicitly set DisableResize: false and added Mac options)
    - [x] Fix "Unknown Item" spam in Storage Box (stop reading at first empty slot and strict DB filtering)
    - [x] Fix Ash of War categorization and item mapping (100% Python parity)
    - [x] Fix incorrect quantities for non-stackable items (forced to 1 for Weapons, Armor, Talismans, AoW)
    - [x] Implement quantity editing with `MaxInventory` and `MaxStorage` validation

## Phase 11: Compatibility & Integrity Fixes (Critical) 🛠
- [x] **11.1. UserData10 Data Range Validation**
    - [x] **Research:** Verify `UserData10` size in `Final.py` and compare with actual `ER0000.sl2` bytes (0x60000 vs 0x600000).
    - [x] **Fix:** Update `save_manager.go` and `structures.go` to use the correct data range for PC metadata.
- [x] **11.2. DLC Stats Implementation**
    - [x] **Research:** Verify Scadutree (-187) and Shadow Realm (-186) offsets in `Final.py`.
    - [x] **Fix:** Add `ScadutreeBlessing` and `ShadowRealmBlessing` to `PlayerGameData` and update mapping logic.
- [x] **11.3. ProfileSummary Expansion**
    - [x] **Research:** Analyze how `Final.py` handles character summaries and if it exceeds 0x100 bytes.
    - [x] **Fix:** Expand `ProfileSummary` struct to prevent data shifting and preserve equipment previews.
- [x] **11.4. SteamID Logic Unification**
    - [x] **Research:** Audit `save_manager.go` and `steamid.go` to resolve offset inconsistencies (0x00 vs 0x14).
    - [x] **Fix:** Standardize SteamID reading/writing across all modules.
- [x] **11.5. Inventory Write Integration**
    - [x] **Research:** Verify if `AddItemsToSlot` in `writer.go` correctly follows Python's insertion logic.
    - [x] **Fix:** Integrate inventory writing into the main `SaveFile()` workflow.
- [x] **11.6. Backup System Robustness**
    - [x] **Research:** Audit `app.go` and `save_manager.go` for conflicting backup logic. Verify if `CreateBackup` is called *before* any write operation.
    - [x] **Fix:** Implement retention policy (max 10 versions) in `backup.go`. Remove redundant/broken backup logic from `save_manager.go`. Ensure `SaveFile` fails if backup fails.

## Phase 12: World Progress Tab — Full Implementation ✅

> **Root cause analysis (pre-implementation research completed 2026-04-08):**
> - ~120 grace entries in `graces.go` lack `(Region)` annotation → all fall under "Unknown" region.
> - `EventFlagsOffset` / `IngameTimerOffset` declared in `SaveSlot` but never computed in `calculateDynamicOffsets()` — always 0.
> - `GraceEntry.Region` has a broken JSON tag (`region"` missing opening quote).
> - `WorldProgressTab` checkboxes are purely visual — no `checked` prop, no `onChange`, no slot context.
> - No API methods `GetGraces(slotIdx)` or `SetGraceVisited(slotIdx, graceID, visited)` exist.
> - Map thumbnail system (23 PNG files) is structurally correct; coverage gaps caused by missing data annotations.

---

- [x] **12.1. Grace Data Cleanup (`backend/db/data/graces.go`)**
    - [x] Fix typo at ID `0x00011642`: `"The Nameless Eternal Cityssssssssssssssssssssssssssssssss"` → `"The Nameless Eternal City"`.
    - [x] Fix `GraceEntry.Region` JSON tag in `db.go`: `region"` → `"region"`.
    - [x] Annotate all ~120 entries missing `(Region)` suffix. Grouping:
        - `0x00011558–0x00011560` → `(Stormveil Castle)`
        - `0x000115BC–0x000115C5` → `(Leyndell Royal Capital)`
        - `0x000115D0–0x000115D5` → `(Leyndell Ashen Capital)`
        - `0x00011616` → `(Roundtable Hold)`
        - `0x0001162A–0x0001163B` → Ainsel River / Siofra River / Lake of Rot (per sub-area)
        - `0x0001163E–0x00011648` → `(Deeproot Depths)` / `(Mohgwyn Palace)`
        - `0x00011652–0x00011655` → `(Mohgwyn Palace)`
        - `0x00011666–0x00011667` → `(Siofra River)`
        - `0x00011684–0x0001168E` → `(Crumbling Farum Azula)`
        - `0x000116E8–0x000116EB` → `(Liurnia of the Lakes)` (Raya Lucaria Academy)
        - `0x0001174C–0x00011754` → `(Miquella's Haligtree)`
        - `0x000117B0–0x000117B7` → `(Mt. Gelmir)` (Volcano Manor)
        - `0x00011878–0x00011879` → `(Limgrave West)` (Tutorial / Stranded Graveyard)
        - `0x000118DC` → `(Leyndell Ashen Capital)` (Fractured Marika)
        - `0x00011940–0x00011C62` → Shadow of the Erdtree sub-regions (see note below)
        - `0x00011D28–0x00011D3C` → Catacombs — per-entry region (Limgrave, Liurnia, Altus, Caelid, etc.)
        - `0x00011D8C–0x00011DA2` → Caves — per-entry region
        - `0x00011DF0–0x00011E29` → Mining Tunnels — per-entry region
        - `0x00011EC2–0x00011EF4` → Divine Towers — per-entry region
        - `0x00011F1C–0x00011F20` → `(Leyndell Royal Capital)` (Underground / Frenzied Flame)
        - `0x000120AC–0x000120AE` → `(Altus Plateau)` (Ruin-Strewn Precipice)
        - `0x00012110–0x00012176` → Shadow of the Erdtree dungeons
        - `0x000121D8–0x0001226F` → Shadow of the Erdtree forges / caves
        - `0x00012B06–0x00012B07` → `(Consecrated Snowfield)`
        - `0x00012B6C–0x00012B6D` → `(Consecrated Snowfield)` (Ordina / Apostate Derelict)
        - `0x00012C00–0x00012CA0` → Shadow of the Erdtree open world sub-regions
    - [x] **DLC sub-region mapping** — all DLC graces grouped under `(Shadow of the Erdtree)` for reliable single map thumbnail.
    - [x] **Verify**: Run `GetAllGraces()` — 0 graces in "Unknown" region after annotation.

- [x] **12.2. EventFlagsOffset Calculation (`backend/core/structures.go`)**
    - [x] Extend `calculateDynamicOffsets()` to compute `IngameTimerOffset` and `EventFlagsOffset` following `Final.py: save_struct()` chain (see reference below).
    - [x] The full chain from `StorageBoxOffset`:
        ```
        gesturesOff    = StorageBoxOffset + 0x100
        unlockedRegSz  = read_u32(Data[gesturesOff])
        unlockedRegion = gesturesOff + unlockedRegSz*4 + 4
        horse          = unlockedRegion + 0x29
        bloodStain     = horse + 0x4C
        menuProfile    = bloodStain + 0x103C
        gaItemsOther   = menuProfile + 0x1B588
        tutorialData   = gaItemsOther + 0x40B
        ingameTimer    = tutorialData + 0x4+0x4+0x1+0x4+0x4+0x1+0x8  (= +0x1A)
        eventFlags     = ingameTimer + 0x1C0000
        ```
    - [x] Assign `s.IngameTimerOffset = ingameTimer` and `s.EventFlagsOffset = eventFlags`.
    - [x] **Validate**: round-trip tests (PS4 + PC) pass after change.

- [x] **12.3. Grace State API (`app.go`, `backend/db/db.go`)**
    - [x] Add `Visited bool` field to `GraceEntry` struct in `db.go`.
    - [x] Add `GetGraces(slotIndex int) ([]db.GraceEntry, error)` to `app.go`:
        - Reads `slot.Data[slot.EventFlagsOffset:]` as the event flags byte array.
        - Calls `db.GetEventFlag(flags, graceID)` for each grace.
        - Returns populated `GraceEntry` list with `Visited` set.
    - [x] Add `SetGraceVisited(slotIndex int, graceID uint32, visited bool) error` to `app.go`:
        - Calls `db.SetEventFlag(slot.Data[slot.EventFlagsOffset:], graceID, visited)`.
        - **Note**: modifies `slot.Data` in-place (write-back already handled by `slot.Write()`).
    - [x] Regenerate Wails bindings: `wails generate module`.

- [x] **12.4. WorldProgressTab — Interactive Checkboxes (`frontend/src/components/WorldProgressTab.tsx`)**
    - [x] Add `charIdx: number` prop. Receive it from the parent tab router (same pattern as `GeneralTab`, `InventoryTab`).
    - [x] Replace `GetAllGraces()` call with `GetGraces(charIdx)` — load visited state per character.
    - [x] Re-fetch graces when `charIdx` changes (`useEffect` dependency).
    - [x] Add `checked={grace.visited}` and `onChange` to each checkbox calling `SetGraceVisited` + optimistic state update.
    - [x] Add per-region progress counter: `"X / Y"` badge in the region header row (highlighted when all visited).
    - [x] Add "Unlock All" button per region (calls `SetGraceVisited` for each unvisited grace in the region).

- [x] **12.5. Map Thumbnail Assignment Verification**
    - [x] Verify `getRegionMapPath()` generates correct filenames for ALL regions after 12.1 data cleanup.
    - [x] All 23 post-cleanup region names resolve to existing PNGs. Only `stormveil_castle.png` missing — acceptable (no world map for this dungeon, `onError` hides the thumbnail).
    - [x] DLC graces share `shadow_of_the_erdtree.png` — single file covers all DLC entries.

---

## Phase 13: Database Tab — Global Add Controls & Infuse Support ✅

> **Root cause analysis:**
> - `AddItemsToCharacter` ma jeden `upgradeLevel int` dla wszystkich itemów — brak rozróżnienia +25/+10/Spirit Ash.
> - Infuse (Heavy/Keen/Blood/…) w ogóle nie jest obsługiwane — offset +100/+200/… nie jest nigdzie aplikowany.
> - Level slider jest w modalnym oknie (per single-item) — dla bulk add nie ma żadnej kontroli poziomu.
> - Tabela ma kolumnę `Max Upgrade` — zbędna skoro level jest globalny.
> - Modal nie jest potrzebny dla upgrade level — przenieść do globalnego paska w nagłówku.

---

- [x] **13.1. Backend: infuse offset constants + nowa sygnatura `AddItemsToCharacter`**
    - [x] Dodać w `backend/db/db.go` typ i slice `InfuseType`:
        ```go
        type InfuseType struct {
            Name   string `json:"name"`
            Offset int    `json:"offset"`
        }
        var InfuseTypes = []InfuseType{
            {"Standard", 0}, {"Heavy", 100}, {"Keen", 200}, {"Quality", 300},
            {"Fire", 400}, {"Flame Art", 500}, {"Lightning", 600}, {"Sacred", 700},
            {"Magic", 800}, {"Cold", 900}, {"Poison", 1000}, {"Blood", 1100}, {"Occult", 1200},
        }
        ```
    - [x] Dodać `GetInfuseTypes() []db.InfuseType` do `app.go` — eksponuje listę infuse do frontendu.
    - [x] Zmienić sygnaturę `AddItemsToCharacter` w `app.go`:
        ```go
        func (a *App) AddItemsToCharacter(
            charIdx int, itemIDs []uint32,
            upgrade25 int, upgrade10 int, infuseOffset int, upgradeAsh int,
            invMax bool, storageMax bool,
        ) error
        ```
    - [x] W `app.go` pre-obliczać `finalID` per item korzystając z `db.GetItemData(id, "")`:
        - `maxUpgrade == 25` → `finalID = id + uint32(infuseOffset) + uint32(upgrade25)`
        - `maxUpgrade == 10` → `finalID = id + uint32(upgrade10)`
        - `category == "ashes"` → `finalID = id + uint32(upgradeAsh)`
        - pozostałe → `finalID = id` (bez zmian)
    - [x] Wywołać `core.AddItemsToSlot(slot, finalIDs, 0, invMax, storageMax)` — `upgradeLevel=0` bo ID już zawierają offset.
    - [x] Regenerować Wails bindings: `wails generate module`.

- [x] **13.2. Frontend: globalny pasek kontrolny w `DatabaseTab`**
    - [x] Wywołać `GetInfuseTypes()` przy inicjalizacji komponentu — załadować listę infuse.
    - [x] Dodać pod paskiem search/category drugi pasek "Global Add Settings" (sticky, widoczny zawsze gdy kategoria zawiera upgradeable items):
        - **Weapon Level (+25)**: `<input type="range" min=0 max=25>` z etykietą `+{val}` — dla weapons/bows/shields/staffs/seals z `maxUpgrade=25`.
        - **Boss Weapon Level (+10)**: `<input type="range" min=0 max=10>` — dla weapons z `maxUpgrade=10` (np. miecze bossów).
        - **Infuse Type**: `<select>` z opcjami z `InfuseTypes` — tylko dla `maxUpgrade=25`.
        - **Spirit Ash Level**: `<input type="range" min=0 max=10>` — dla kategorii `ashes`.
    - [x] Kontrolki widoczne kontekstowo: przy kategorii `ashes` pokazać tylko Spirit Ash Level; przy `weapons/bows/shields/staffs/seals` pokazać Weapon Level + Infuse; przy `all` pokazać wszystkie; przy pozostałych kategoriach ukryć cały pasek.
    - [x] State: `upgrade25`, `upgrade10`, `infuseOffset`, `upgradeAsh` — persystować między zmianami kategorii.

- [x] **13.3. Frontend: uproszczenie tabeli i modalu**
    - [x] Usunąć kolumny `Max Upgrade`, `Max Inv`, `Max Storage` z tabeli — lista jest czystsza.
    - [x] Usunąć slider upgrade level z modalu (był tylko dla single-item add z `maxUpgrade > 0`).
    - [x] Modal zachować tylko dla potwierdzenia: pokazuje liczbę itemów + checkboxy `Inventory Max` / `Storage Max`.
    - [x] Przycisk "Add Selected" wywołuje `AddItemsToCharacter` z globalnymi wartościami upgrade25/10/infuse/ash.
    - [x] Alternatywnie: usunąć modal całkowicie — "Add Selected" dodaje bezpośrednio z globalnymi ustawieniami + toastem potwierdzenia.

- [x] **13.4. Frontend: preview nazwy infuse/level w tabeli (opcjonalne)**
    - [x] W kolumnie `Name` pod nazwą wyświetlać małym tekstem podgląd: np. `"Heavy +15"` jeśli globalny infuse ≠ Standard lub level > 0.
    - [x] Dotyczy tylko itemów z `maxUpgrade > 0`.

---

### Technical Note: Faster Invasions (Meliodas Method)
A recent discovery (popularized by Steelovsky and Meliodas) allows for significantly faster matchmaking by modifying the `NetworkParam` structure within the `Regulation` block of the save file.
- **Refresh Interval**: Reduced from 20s to 4s.
- **Search Scope**: Increased simultaneous region checks for "Near/Far" invasions.
- **Location**: `UserData11` (Offset `0x1960070` on PC).
- **Status**: Researching exact byte offsets for automated implementation.
