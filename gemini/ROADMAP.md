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

## Phase 14: Database Tab — Ukrycie wariantów infuse z listy broni ✅

> **Root cause:**
> Mapy `data.Weapons`, `data.Bows`, `data.Shields`, `data.Staffs`, `data.Seals` zawierają osobne wpisy
> dla każdego wariantu infuse (`Heavy X`, `Keen X`, `Blood X`, …). IDs wariantów mają ścisłą zależność:
> `variantID = baseID + N×100` (N=1..12, odpowiadające offsetom infuse).
> `GetItemsByCategory` zwraca wszystkie wpisy — stąd "Heavy Main-gauche", "Keen Main-gauche" itd. są widoczne.
>
> **Metoda filtrowania:** Dla każdego wpisu z `maxUpgrade == 25` sprawdzić, czy
> `id - N×100` (N=1..12) istnieje w tej samej mapie. Jeśli tak → to wariant infuse → pominąć.
> Bezpieczne dla broni o nazwach zawierających słowa infuse: "Bloody Buckler", "Bloodstained Dagger" itp.
> nie mają odpowiadającego base'a w mapie i zostaną zachowane.

---

- [x] **14.1. Backend: filtrowanie wariantów infuse w `GetItemsByCategory` (`backend/db/db.go`)**
    - [ ] Dodać helper `filterInfuseVariants(items []ItemEntry) []ItemEntry`:
        - Buduje `idSet := map[uint32]bool` ze wszystkich ID na liście.
        - Dla każdego itemu z `maxUpgrade == 25`: sprawdza czy `id - N×100` ∈ `idSet` dla N=1..12.
        - Jeśli tak → pomija (to wariant); jeśli nie → zachowuje.
        - Itemy z `maxUpgrade != 25` (boss weapons, nieupgradeable) są zawsze zachowywane.
    - [ ] Wywołać `filterInfuseVariants` po `processMap` dla kategorii:
        `weapons`, `bows`, `shields`, `staffs`, `seals` — oraz dla `all` (przez `GetAllItems`).
    - [ ] Zweryfikować liczbę wpisów przed/po filtrze: oczekiwane ~12× mniej dla każdej kategorii
        zawierającej infuse warianty.

- [x] **14.2. Walidacja i testy**
    - [x] "Heavy Crossbow" pozostaje (brak base'a w bows), "Bloodstained Dagger" pozostaje.
    - [x] "Bloody Buckler" i "Bloody Longsword" są filtrowane — to Blood-infused warianty (ID = base + 1100).
    - [x] "Heavy Main-gauche", "Keen Main-gauche", "Occult Dagger" znikają z listy.
    - [x] 0 false positives w weapons/shields/bows/staffs/seals. `go build ./backend/...` OK.

---

## Phase 15: Save/Backup/Write Pipeline — Naprawa krytycznych błędów ✅

> **Audyt przeprowadzony 2026-04-08. Znalezione problemy:**

### Bug #1 — KRYTYCZNY: Zapis do nowego pliku zawsze kończy się błędem (save aborted)

`WriteSave` w `app.go` wywołuje `CreateBackup(path)` gdzie `path` to ścieżka DOCELOWA wybrana przez
użytkownika w `SaveFileDialog`. `CreateBackup` próbuje otworzyć ten plik przez `os.Open`. Jeśli
plik nie istnieje (użytkownik zapisuje do nowego pliku) → `os.Open` zwraca `ENOENT` → `CreateBackup`
zwraca błąd → `WriteSave` zwraca `"backup failed, save aborted"` → **zapis jest anulowany**.
Użytkownik może zapisywać TYLKO nadpisując istniejący plik.

### Bug #2 — KRYTYCZNY: Parametr `platform` w `WriteSave` jest ignorowany → konwersja zepsuta

Frontend wywołuje `WriteSave(targetPlatform)` gdzie `targetPlatform ∈ {"PC", "PS4"}`. Ale `app.go`:
```go
func (a *App) WriteSave(platform string) error {
    ...
    return a.save.SaveFile(path)  // ignoruje `platform`, używa a.save.Platform z LoadSave
}
```
`SaveFile()` używa `s.Platform` ustawionego przy wczytaniu pliku. Konwersja PS4→PC i PC→PS4
jest całkowicie zepsuta — zawsze zapisuje w formacie oryginalnym. Dodatkowo: przy konwersji
PS4→PC należy ustawić `s.Encrypted = true` i wygenerować losowe IV (PC save jest szyfrowany AES).

### Bug #3 — ISTOTNY: Zapis nieatomowy — ryzyko uszkodzenia pliku

`os.WriteFile(path, finalData, 0644)` w `SaveFile()` nie jest atomowy. Przerwanie zapisu
(brak prądu, pełny dysk, kill procesu) zostawia plik w uszkodzonym stanie. Backup chroni
przed nadpisaniem, ale nie przed częściowym zapisem do celu.

### Bug #4 — DO WERYFIKACJI: BND4 header SHA256 po edycji może być nieaktualny

Header 0x300 bajtów jest zachowywany verbatim z wczytanego pliku. Jeśli bajty 0x10–0x2F
zawierają SHA256 zaszyfrowanego lub odszyfrowanego payloadu (jak w Python `Final.py`),
to po edycji (zmiana danych slotów → nowe MD5 → inny payload) hash w headerze jest stary.
Wymaga weryfikacji z `Final.py`.

---

- [x] **15.1. Naprawa `CreateBackup` / `WriteSave` — backup tylko gdy plik docelowy istnieje**
    - [x] W `app.go` `WriteSave`: przed `CreateBackup` sprawdzić `os.Stat(path)`.
        Jeśli plik nie istnieje → pomiń backup. Jeśli istnieje → backup jak dotychczas.

- [x] **15.2. Naprawa `WriteSave` — zastosowanie parametru `platform`**
    - [x] W `app.go` przed `SaveFile(path)` ustawić `a.save.Platform = core.Platform(platform)`.
    - [x] PS4→PC: generowanie losowego IV + `Encrypted=true` (`crypto/rand`).
    - [x] PC→PS4: `Encrypted=false`.
    - [x] Naprawa headerów przy konwersji (`save_manager.go`):
        - PS4→PC: `buildPCBND4Header()` — programatyczne generowanie 0x300-bajtowego BND4 headera.
        - PC→PS4: `ps4HeaderTemplate` — stały 0x70-bajtowy header PS4.
    - [x] Wszystkie 5 testów round-trip i konwersji przechodzą (`go test ./tests/...`).

- [x] **15.3. Atomowy zapis — write-then-rename**
    - [x] W `SaveFile()` (`backend/core/save_manager.go`): zapis do `.tmp` + `os.Rename` (POSIX atomic).
    - [x] Cleanup `.tmp` przy błędzie zapisu.

- [x] **15.4. Weryfikacja BND4 header SHA256 — N/A**
    - [x] `Final.py` używa wyłącznie MD5 per-slot (nie SHA256 w BND4 headerze).
        MD5 jest już przeliczany w `SaveFile()`. Brak action required.

---

## Phase 16: Live Inventory Sync — Natychmiastowa widoczność dodanych itemów ✅

> **Root cause:**
> `InventoryTab` fetchuje dane (`GetCharacter`) wyłącznie w `useEffect([charIndex])` — trigger odpala się
> tylko przy zmianie wybranego slotu. Przejście na zakładkę Inventory po dodaniu itema w DatabaseTab
> nie triggeruje re-fetcha → nowe itemy są niewidoczne do ręcznej zmiany slotu i powrotu.
> `DatabaseTab` po `AddItemsToCharacter` nie emituje żadnego zdarzenia do reszty aplikacji.

---

- [x] **16.1. App.tsx — `inventoryVersion` counter**
    - [x] `const [inventoryVersion, setInventoryVersion] = useState(0)`.
    - [x] Tab click handler: `if (tab === 'inventory') setInventoryVersion(v => v + 1)` przed `setActiveTab`.
    - [x] `onItemsAdded={() => setInventoryVersion(v => v + 1)}` przekazany do `DatabaseTab`.
    - [x] `inventoryVersion` przekazany do `InventoryTab`.

- [x] **16.2. DatabaseTab.tsx — callback `onItemsAdded`**
    - [x] Prop `onItemsAdded?: () => void` dodany do interfejsu.
    - [x] Po udanym `AddItemsToCharacter`: `onItemsAdded?.()`.

- [x] **16.3. InventoryTab.tsx — `inventoryVersion` w useEffect**
    - [x] Prop `inventoryVersion: number` dodany do interfejsu.
    - [x] `useEffect([charIndex, inventoryVersion])` — re-fetch przy każdej inkrementacji.

- [x] **16.4. Walidacja**
    - [x] `cd frontend && npx tsc --noEmit` — czysto (0 błędów).

---

## Phase 17: Database Tab — Fix Upgraded Items + UI Polish + Modal Redesign

> **Root causes:**
> - `mapItems` w `character_vm.go` filtruje itemy, których ID nie ma w DB (upgraded ID np. `0x802ED835`
>   nie istnieje — tylko base `0x802ED830`). `GetItemName` w `db.go` już robi fuzzy lookup
>   (`id & 0xFFFFFF00 == baseID & 0xFFFFFF00`) ale `mapItems` jej nie używa.
> - Label "Boss +10" jest mylący — wszystkie bronie z maxUpgrade=10 to boss weapons, ale slider powinien
>   mówić "Weapon +10".
> - Search bar i filter są w odwrotnej kolejności; filter ma inny styl niż w InventoryTab.
> - Modal pokazuje tylko checkboxy bez możliwości podania ilości.

---

- [x] **17.1. Fix: upgraded weapons niewidoczne w Inventory (`backend/vm/character_vm.go`)**
    - [x] Dodać `GetItemDataFuzzy(id uint32) (data.ItemData, uint32)` w `backend/db/db.go`:
        - Próbuje exact match (`GetItemData`).
        - Dla upper nibble 0x8 (Weapon): iteruje po `Weapons, Bows, Shields, Staffs, Seals`;
          sprawdza `id & 0xFFFFFF00 == baseID & 0xFFFFFF00`; jeśli match → zwraca base ItemData + baseID.
    - [x] W `mapItems` (`backend/vm/character_vm.go`): zastąpić `GetItemData(itemID)` przez `GetItemDataFuzzy(itemID)`.
    - [x] Dla wyświetlania nazwy: jeśli `itemID != baseID`, append `" +N"` gdzie N = `itemID - baseID`.
    - [x] `go build ./backend/... && go build .` — OK.

- [x] **17.2. Rename "Boss +10" → "Weapon +10" (`frontend/src/components/DatabaseTab.tsx`)**
    - [x] Zmieniono label suwaka `upgrade10` z `"Boss +10"` na `"Weapon +10"`.

- [x] **17.3. Swap search/filter + resize filter (`frontend/src/components/DatabaseTab.tsx`)**
    - [x] `<select>` kategorii przeniesiony PRZED `<input>` wyszukiwarki.
    - [x] Styl filtra: `w-56 appearance-none bg-muted/30 border border-border rounded-md px-4 py-2.5 pr-10 text-[10px] font-black uppercase tracking-widest` — zgodny z InventoryTab.

- [x] **17.4. Modal redesign — qty inputs + max checkboxes**
    - [x] **Backend `app.go`**: sygnatura `AddItemsToCharacter` zmieniona:
        `invMax, storageMax bool` → `invQty, storageQty int`; `resolveQty()` helper.
    - [x] **Backend `writer.go`**: `AddItemsToSlot` zmienione:
        `invMax, storageMax bool` → `invQty, storageQty int`; qty przekazane do `addToInventory`.
    - [x] Wails bindings zregenerowane: `wails generate module`.
    - [x] **Frontend `DatabaseTab.tsx`**: stany `addToInv`, `invMax`, `invQtyVal`, `addToStorage`, `storageMax`, `storageQtyVal`.
    - [x] Modal — dwa wiersze (Inventory / Storage):
        - Dla niestack. (maxInventory == 1): tylko checkbox.
        - Dla stack. (maxInventory > 1): `[qty input] [x Max]` — Max checked → input disabled, wysyłane -1.
    - [x] `cd frontend && npx tsc --noEmit` — 0 błędów.
    - [x] Round-trip testy: 4/4 PASS.

---

---

## Phase 18: Character Slot Management — Clone & Delete

> **Scope:** Zarządzanie slotami postaci bezpośrednio w edytorze bez wychodzenia do gry.
> Klonowanie kopiuje cały slot binarnie (0x280000 B) + metadata. Usuwanie przesuwa pozostałe
> sloty w dół eliminując luki. Brak potrzeby zewnętrznego szablonu — operacje wyłącznie na
> istniejących danych save file.

---

- [x] **18.1. Backend: `CloneSlot(srcIdx, destIdx int) error` (`app.go`)**
    - [x] Walidacja: `src` aktywny, `dest` nieaktywny, `src != dest`, oba 0–9.
    - [x] Głęboka kopia: `make([]byte, 0x280000)` + `copy` dla `Slots[src].Data`.
    - [x] `ActiveSlots[dest] = true`, `ProfileSummaries[dest] = ProfileSummaries[src]`.
    - [x] Wails bindings zaktualizowane ręcznie (App.js + App.d.ts).

- [x] **18.2. Backend: `DeleteSlot(idx int) error` (`app.go`)**
    - [x] Walidacja: `idx` 0–9, slot aktywny.
    - [x] Shift w dół: `for i := idx; i < 9; i++` →
        `Slots[i] = Slots[i+1]`, `ActiveSlots[i] = ActiveSlots[i+1]`, `ProfileSummaries[i] = ProfileSummaries[i+1]`.
    - [x] Zerowanie ostatniego slotu: `Slots[9].Data = make([]byte, 0x280000)`,
        `ActiveSlots[9] = false`, `ProfileSummaries[9] = ProfileSummary{}`.
    - [x] `MagicOffset` ustawiony na fallback (0x15420+432) — zapobiega panice w `Write()`.

- [x] **18.3. Frontend: przycisk Clone w sidebarze (`App.tsx`)**
    - [x] Ikona clone przy każdym aktywnym slocie (widoczna on-hover).
    - [x] Klik → modal z listą wolnych slotów (nieaktywnych).
    - [x] Potwierdzenie → `CloneSlot(src, dest)` → `refreshSlots()`.

- [x] **18.4. Frontend: przycisk Delete w sidebarze (`App.tsx`)**
    - [x] Ikona kosza przy każdym aktywnym slocie (widoczna on-hover).
    - [x] Dialog potwierdzenia: `"Delete [name]? This cannot be undone."`.
    - [x] Potwierdzenie → `DeleteSlot(idx)` → `refreshSlots()`.

- [x] **18.5. Walidacja**
    - [x] `go test -v ./tests/roundtrip_test.go` — 4/4 PASS.
    - [x] `cd frontend && npx tsc --noEmit` — 0 błędów.
    - [ ] `make build`.

---

## Phase 19: Category Refactor — Wyrównanie do wiki.gg/Inventory ✅

> **Cel:** Zastąpić obecne 24 granularne filtry UI przez 17 kategorii zgodnych z podziałem
> ekwipunku w grze (https://eldenring.wiki.gg/wiki/Inventory). Wiąże się to z połączeniem
> plików Go w `backend/db/data/`, reorganizacją katalogów ikon i aktualizacją `Category`/`IconPath`
> we wszystkich wpisach bazy danych.

### Mapa kategorii (obecne → nowe)

| Obecne pliki / kategorie | Nowa kategoria | Akcja |
|---|---|---|
| `weapons.go` — `"weapons"` | `"melee_armaments"` | rename Category + IconPath dir |
| `bows.go` + `staffs.go` + `seals.go` | `"ranged_and_catalysts"` | merge 3 pliki → 1, rename dir |
| `arrows_and_bolts.go` | bez zmian | — |
| `shields.go` | bez zmian | — |
| `helms.go` — `"helms"` | `"head"` | rename Category + move icons `armor/helms/` → `head/` |
| `chest.go` | `"chest"` | move icons `armor/chest/` → `chest/` |
| `gauntlets.go` — `"gauntlets"` | `"arms"` | rename Category + move icons `armor/gauntlets/` → `arms/` |
| `leggings.go` — `"leggings"` | `"legs"` | rename Category + move icons `armor/leggings/` → `legs/` |
| `talismans.go` | bez zmian | — |
| `aows.go` — `"aows"` | `"ashes_of_war"` | rename Category + rename icon dir `aow/` → `ashes_of_war/` |
| `ashes.go` | bez zmian | — |
| `sorceries.go` | bez zmian | — |
| `incantations.go` | bez zmian | — |
| `crafting_materials.go` | bez zmian | — |
| `bolstering_materials.go` + `golden_runes.go` | `"bolstering_materials"` | merge 2 pliki → 1, rename icon dir `bolstering/` → `bolstering_materials/` |
| `keyitems.go` + `remembrances.go` | `"key_items"` | merge 2 pliki → 1, rename icon dir `keyitems/` → `key_items/` |
| `consumables.go` + `sacred_flasks.go` + `throwing_pots.go` + `perfume_arts.go` + `throwables.go` + `grease.go` + `misc_tools.go` + `quest_tools.go` | `"tools"` | merge 8 pliki → 1, merge icon dirs → `tools/` |

> `gestures.go` — nie jest kategorią ekwipunku w UI; plik pozostaje bez zmian (używany wewnętrznie).

---

- [x] **19.1. Reorganizacja katalogów ikon (`frontend/public/items/`)**
    - [x] Rename `items/aow/` → `items/ashes_of_war/`
    - [x] Rename `items/bows/` → `items/ranged_and_catalysts/`, przenieś tam zawartość `items/staffs/` i `items/seals/`
    - [x] Rename `items/armor/helms/` → `items/head/`
    - [x] Rename `items/armor/chest/` → `items/chest/`
    - [x] Rename `items/armor/gauntlets/` → `items/arms/`
    - [x] Rename `items/armor/leggings/` → `items/legs/`
    - [x] Usuń pusty katalog `items/armor/`
    - [x] Rename `items/bolstering/` → `items/bolstering_materials/`
    - [x] Rename `items/keyitems/` → `items/key_items/`
    - [x] Rename `items/weapons/` → `items/melee_armaments/`
    - [x] Merge `items/consumables/` + `items/grease/` + `items/misc_tools/` → `items/tools/` (przenieś wszystkie pliki)
    - [x] Usuń opustoszałe katalogi: `items/staffs/`, `items/seals/`, `items/bows/`, `items/consumables/`, `items/grease/`, `items/misc_tools/`

- [x] **19.2. Merge plików Go w `backend/db/data/`**
    - [x] Połącz `bows.go` + `staffs.go` + `seals.go` → `ranged_and_catalysts.go`
    - [x] Zmień `Category` w `weapons.go` z `"weapons"` → `"melee_armaments"`
    - [x] Zmień `Category` w `helms.go` z `"helms"` → `"head"`
    - [x] Zmień `Category` w `chest.go`, `IconPath`: `items/armor/chest/` → `items/chest/`
    - [x] Zmień `Category` w `gauntlets.go` → `arms.go` z `"gauntlets"` → `"arms"`
    - [x] Zmień `Category` w `leggings.go` → `legs.go` z `"leggings"` → `"legs"`
    - [x] Zmień `Category` w `aows.go` z `"aows"` → `"ashes_of_war"`
    - [x] Połącz `bolstering_materials.go` + `golden_runes.go` → `bolstering_materials.go`
    - [x] Połącz `keyitems.go` + `remembrances.go` → `key_items.go`
    - [x] Połącz 8 plików consumables → `tools.go`

- [x] **19.3. Aktualizacja `backend/db/db.go`**
    - [x] Zaktualizowany `switch category` z nowymi case'ami
    - [x] Zaktualizowane mapowanie handle prefix → category string
    - [x] Usunięte nieużywane importy, nowe nazwy zmiennych (`data.RangedAndCatalysts`, etc.)
    - [x] Zaktualizowany `GetAllItems()`

- [x] **19.4. Aktualizacja `frontend/src/components/DatabaseTab.tsx`**
    - [x] 17 wiki-aligned kategorii z optgroup'ami
    - [x] Zaktualizowana logika upgrade/infuse sliders dla nowych nazw kategorii

- [x] **19.5. Walidacja**
    - [x] `go build ./backend/...` — PASS
    - [x] `go test -v ./tests/...` — PASS
    - [x] `cd frontend && npx tsc --noEmit` — PASS
    - [x] `cd frontend && npm run lint` — PASS
    - [x] Ręczna weryfikacja UI — OK

---

## Phase 20: Offset Safety & Code Hardening 📋

> **Branch:** `feature/phase20-offset-safety`
> **Szczegółowy plan implementacji:** [`gemini/REFACTOR.md`](REFACTOR.md)
>
> **Cel:** Wyeliminować ryzyka uszkodzenia save file (panic, buffer overflow, silent corruption)
> przez bounds-checked offset management, walidację krzyżową łańcucha offsetów, error propagation
> zamiast silent fallback, oraz poprawę spójności i wydajności frontendu.
>
> **Zasada:** Zero zmian w binarnym formacie save file. Refaktor dotyczy WYŁĄCZNIE logiki
> odczytu/zapisu/walidacji. Round-trip testy MUSZĄ przechodzić po każdym etapie.

### Etapy (każdy = osobny commit)

- [x] **20.A. Named Offset Constants** (`backend/core/offset_defs.go` — NOWY)
    - [x] Jedno źródło prawdy: stałe dla stat offsets, dynamic chain, inventory layout, sanity limits.
    - [x] Stałe weryfikowane z `gemini/SPEC.md` §5.2 i §5.4.

- [x] **20.B. SlotAccessor** (`backend/core/slot_access.go` — NOWY)
    - [x] Bounds-checked ReadU32/WriteU32/ReadU8/WriteU8/ReadU16/WriteU16.
    - [x] `ReadDynamicSize(off, maxSize, name)` — clamp + warning zamiast panic.
    - [x] `CheckBounds(off, size, label)` — pre-write validation.
    - [x] `Warnings []string` — non-fatal issues (PS4 garbage).

- [x] **20.C. Error Propagation** (`backend/core/structures.go` — MODIFY)
    - [x] `mapStats()` → zwraca `error`, używa `SlotAccessor` + stałych z `offset_defs.go`.
    - [x] `calculateDynamicOffsets()` → zwraca `error`, używa `ReadDynamicSize` z sanity limits.
    - [x] `Read()` → propaguje errory, dodaje warning przy MagicPattern fallback.
    - [x] `Write()` → używa `SlotAccessor` + named constants.
    - [x] Dodanie pola `Warnings []string` do `SaveSlot` struct.

- [x] **20.D. Cross-Validation** (`backend/core/structures.go` — MODIFY)
    - [x] `validateOffsetChain()` — sprawdza bounds + monotoniczny porządek offsetów.
    - [x] Wywoływana po `calculateDynamicOffsets()`, przed `mapInventory()`.

- [x] **20.E. Writer Safety** (`backend/core/writer.go` — MODIFY)
    - [x] `writeGaItem()`: bounds check przez `SlotAccessor.CheckBounds()`.
    - [x] `addToInventory()`: bounds check przed zapisem do `slot.Data`.
    - [x] `generateUniqueHandle()`: zmiana sygnatury na `(uint32, error)`, limit 10000 iteracji.

- [x] **20.F. Warnings Pipeline** (`backend/vm/character_vm.go`, `frontend/src/components/App.tsx`)
    - [x] `CharacterViewModel.Warnings []string` — propagacja z `SaveSlot.Warnings`.
    - [x] UI: żółty banner "Save loaded with warnings" z listą.

- [x] **20.G. Frontend Hardening** (frontend)
    - [x] `ErrorBoundary` component w `main.tsx`.
    - [x] `useMemo` w `InventoryTab` i `DatabaseTab` (filtered/sorted lists).
    - [x] Fix `window.go.main.App.SaveCharacter` → import z wailsjs.
    - [x] `WorldProgressTab`: nie połykaj błędów w `.catch()`.

- [x] **20.H. Unit Tests** (`backend/core/` — NOWE PLIKI)
    - [x] `slot_access_test.go`: out-of-bounds, negative offset, dynamic size clamp.
    - [x] `offset_validation_test.go`: valid chain, non-monotonic, too-small MagicOffset.
    - [x] Rozszerzenie `roundtrip_test.go`: sprawdzenie `Warnings == nil` na known-good saves.

- [x] **20.I. SaveManager Hardening** (`backend/core/save_manager.go`)
    - [x] Walidacja minimalnego rozmiaru pliku w `LoadSave()`.
    - [x] Error propagation z `ReadBytes()` (zamień `_` na obsługę błędu).
    - [x] Cross-platform atomic write (`os.Rename` fix dla Windows, nie kasuj `.tmp` przy błędzie).

- [x] **20.J. Database & Event Flags Hardening** (`backend/db/db.go`, `app.go`)
    - [x] `GetEventFlag`/`SetEventFlag` → zwracają `error` na OOB zamiast silent no-op.
    - [x] Global item index `map[uint32]ItemEntry` — O(1) lookup zamiast O(18×n) linear search.

- [x] **20.K. Frontend Performance & UI Consistency**
    - [x] Table virtualization z `@tanstack/react-virtual` w InventoryTab i DatabaseTab.
    - [x] Unified toast system (`react-hot-toast`), usunięcie `alert()`.
    - [x] ~~Shared UI components~~ — pominięte (zbyt mała duplikacja, over-engineering).

### Walidacja końcowa

- [ ] `go test -v ./backend/core/...` — PASS
- [ ] `go test -v ./tests/roundtrip_test.go` — 4/4 PASS (PS4, PC, PS4→PC, PC→PS4)
- [ ] `cd frontend && npx tsc --noEmit` — 0 błędów
- [ ] `cd frontend && npm run lint` — 0 błędów
- [ ] `make build` — OK

---

## Phase 19b: Quantity Hard Cap Enforcement ✅

> **Cel:** Uniemożliwić ustawienie większej ilości itemów niż dozwolone w bazie danych (MaxInventory / MaxStorage).
> Przekroczenie limitów ilościowych jest wykrywane przez serwery FromSoftware i grozi banem konta.
> Enforcement na 3 warstwach: backend save-write, frontend add-modal, frontend inventory edit.

---

- [x] **20.1. Backend: fix storage cap w `updateItemsAndSync` (`backend/vm/character_vm.go`)**
    - [x] Bug: `updateItemsAndSync` używał `MaxInventory` zarówno dla inventory jak i storage.
    - [x] Fix: rozdzielenie — `isStorage=true` → cap do `MaxStorage`; `isStorage=false` → cap do `MaxInventory`.

- [x] **20.2. Frontend DatabaseTab: `Math.min` zamiast `Math.max` w modalu (`DatabaseTab.tsx`)**
    - [x] `modalMaxInv` i `modalMaxStorage` zmienione z `Math.max(...)` na `Math.min(...)`.
    - [x] Przy bulk-add mieszanych itemów (np. arrow max=99 + shard max=999) qty jest ograniczone do najniższego limitu.
    - [x] Dodany info label (amber) gdy zaznaczone itemy mają różne maxy: `"Qty capped to lowest max: Inv N, Storage M"`.

- [x] **20.3. Frontend InventoryTab: atrybuty `min`/`max` na inputach qty (`InventoryTab.tsx`)**
    - [x] Dodane `min={0}` i `max={item.maxInv}` na inventory qty input.
    - [x] Dodane `min={0}` i `max={item.maxStorage}` na storage qty input.
    - [x] Dodatkowy guardrail do istniejącego `handleQtyChange` (który już clampuje programatycznie).

- [x] **20.4. Backend `resolveQty` (`app.go`) — bez zmian**
    - [x] Istniejąca logika już poprawnie clampuje per-item: `qty > max → max`. Brak potrzeby zmian.

- [x] **20.5. Walidacja**
    - [ ] `go build ./backend/... && go build .` — 0 błędów
    - [ ] `cd frontend && npx tsc --noEmit` — 0 błędów
    - [ ] `cd frontend && npm run lint` — 0 błędów

---

## Phase 21: Advanced Safety & UX 📋

> **Zależność:** Wymaga ukończonego Phase 20.
> **Szczegóły architektoniczne:** [`gemini/REFACTOR.md` §15](REFACTOR.md)

- [ ] **21.1. Write-ahead validation** — `validateSlotIntegrity()` przed każdym `SaveFile()` jako ostatnia linia obrony przed zapisaniem uszkodzonego save'a.
- [ ] **21.2. `updateItemsAndSync()` transactionality** — walidacja offsetów przed startem zapisu qty, rollback na kopii `slot.Data` przy błędzie. Migracja na `SlotAccessor`.
- [ ] **21.3. Undo/redo** — deep copy `slot.Data` przed edycją, stack operacji w `App`, przycisk "Revert" w UI.
- [ ] **21.4. Save file diffing** — porównanie przed/po zapisie, UI dialog "Review Changes" z listą modyfikacji.

---

### Technical Note: Faster Invasions (Meliodas Method)
A recent discovery (popularized by Steelovsky and Meliodas) allows for significantly faster matchmaking by modifying the `NetworkParam` structure within the `Regulation` block of the save file.
- **Refresh Interval**: Reduced from 20s to 4s.
- **Search Scope**: Increased simultaneous region checks for "Near/Far" invasions.
- **Location**: `UserData11` (Offset `0x1960070` on PC).
- **Status**: Researching exact byte offsets for automated implementation.
