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
    - [x] Add category filters (Weapons, Armor, Talismans, Goods)

## Phase 9: PvP Optimization & Advanced Tweaks (In Progress) 🛠
- [ ] **9.1. Faster Invasions (Meliodas Method)**
    - [ ] Research exact offsets for `NetworkParam` in `UserData11` (Regulation block)
    - [ ] Implement matchmaking interval reduction (20s -> 4s)
    - [ ] Implement search scope expansion (Global region polling)
    - [ ] Add "Enable Faster Invasions" toggle in Settings tab
- [x] **9.2. World Progress Automation**
    - [x] Group Sites of Grace by region (Limgrave, Liurnia, Caelid, etc.)
    - [x] Fix scrolling issue on World Progress tab
    - [x] Add interactive map thumbnails for each region
    - [ ] Add "Unlock All Invasion Regions" button
    - [ ] Add "Activate All Summoning Pools" button
- [ ] **9.3. Matchmaking Safety Tools**
    - [ ] Implement Weapon Level scanner and "Safe De-leveling" logic
    - [ ] Add "Global Weapon Level" setting for bulk item addition (Future)

## Phase 10: Item Icons & Visual Assets (In Progress) 🛠
- [x] **10.1. Icon Directory Structure**
    - [x] Setup `frontend/public/items/` with subdirectories for categories
- [x] **10.2. Name Normalization Logic**
    - [x] Implement `getItemIconPath` in `InventoryTab.tsx`
    - [x] Fix edge cases for special characters (apostrophes, hyphens, dots)
    - [x] Handle "Altered" armor variants mapping (e.g., "Banished Knight Armor (Altered)")
    - [ ] Support for DLC item name normalization
- [x] **10.3. Asset Coverage & Validation**
    - [ ] Audit missing icons for DLC items
    - [x] Implement fallback icon (placeholder) for missing assets
    - [x] Fix broken links for items with non-standard filenames (e.g. "All-Knowing" vs "all_knowing")
- [x] **10.4. UI Integration**
    - [x] Display icons in Inventory, Storage, and Database tables
    - [x] Implement "Icon Popover" for high-resolution preview
    - [x] Add icons to Character Importer and Stats tabs
    - [x] Fix UI scaling issues (removed max-w-5xl and fixed heights for full-width/height support)
    - [x] Improve scrollbar visibility and fix nested scrolling conflicts
    - [x] Enable window maximization on macOS (explicitly set DisableResize: false and added Mac options)
    - [x] Fix "Unknown Item" spam in Storage Box (stop reading at first empty slot and strict DB filtering)
    - [x] Fix Ash of War categorization and item mapping (100% Python parity)
    - [x] Fix incorrect quantities for non-stackable items (forced to 1 for Weapons, Armor, Talismans, AoW)
    - [x] Implement quantity editing with `MaxInventory` and `MaxStorage` validation

---

### Technical Note: Faster Invasions (Meliodas Method)
A recent discovery (popularized by Steelovsky and Meliodas) allows for significantly faster matchmaking by modifying the `NetworkParam` structure within the `Regulation` block of the save file.
- **Refresh Interval**: Reduced from 20s to 4s.
- **Search Scope**: Increased simultaneous region checks for "Near/Far" invasions.
- **Location**: `UserData11` (Offset `0x1960070` on PC).
- **Status**: Researching exact byte offsets for automated implementation.
