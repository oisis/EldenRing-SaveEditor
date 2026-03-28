# Project Roadmap: ER-Save-Editor-Go (100% Rust Parity)

> **Status:** 🏗️ Phase 5 In Progress | **Source of Truth:** `tmp/org-src` | **Test Strategy:** Round-trip & Golden Files

## Phase 1: Environment & Infrastructure ✅
- [x] **1.1. Go Initialization**
    - [x] `go mod init github.com/oisis/EldenRing-SaveEditor`
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

## Phase 2: Data Extraction (Rust -> Go) ✅
- [x] **2.1. Extraction Tooling**
    - [x] Create Go script to parse `tmp/org-src/src/db/*.rs` and generate Go maps
- [x] **2.2. Constant Extraction**
    - [x] Port item names and IDs from `src/db/*.rs` to Go maps/JSON
    - [x] Port grace and boss flag IDs from `src/db/graces.rs` and `src/db/bosses.rs`
- [x] **2.3. Stats & Classes**
    - [x] Port HP/FP/SP tables from `src/db/stats.rs`
    - [x] Port base class stats from `src/db/classes.go`

## Phase 3: Binary Core (The "encoding/binary" Layer) ✅
- [x] **3.1. Crypto Implementation**
    - [x] Port AES-128-CBC logic for PC saves (`backend/core/crypto.go`)
    - [x] Implement MD5 and SHA256 checksum logic (1:1 with Rust)
- [x] **3.2. Binary Structures (PC)**
    - [x] Define `BND4` container struct
    - [x] Define `PCSaveSlot` and `PCUserData` structs matching Rust offsets
    - [ ] **Test:** Round-trip validation for PC raw saves
- [x] **3.3. Binary Structures (PlayStation)**
    - [x] Define `PSSaveSlot` and `PSUserData` structs
    - [x] Implement raw binary reading/writing for Save Wizard compatibility
    - [ ] **Test:** Round-trip validation for PS raw saves
- [x] **3.4. SteamID Logic**
    - [x] Implement SteamID detection and modification in `UserData10`
    - [x] **Test:** Verify MD5 checksum recalculation after SteamID change
- [x] **3.5. Backup System**
    - [x] Implement automatic `.bak` creation with timestamps before any write

## Phase 4: Logic & ViewModel (Go Backend) ✅
- [x] **4.1. Save Manager**
    - [x] Implement `LoadSave(path)` with auto-detection (PC vs PS)
    - [ ] Implement `SaveFile()` with integrity check (Round-trip Validation)
- [x] **4.2. ViewModel Mapping**
    - [x] Map raw bytes to `CharacterViewModel` (Name, Stats, Souls)
    - [ ] Map `EventFlags` (bits) to boolean flags for Graces/Bosses
- [x] **4.3. Validation Logic**
    - [x] Implement stat recalculation (Level = sum of attributes - 79)
    - [x] Implement Weapon Matchmaking Level scanner (Somber vs Normal)
    - [x] **Test:** Unit tests for level calculation formula matching Rust logic

## Phase 5: UI Implementation (Wails Frontend) 🏗️
- [x] **5.1. Base Layout**
    - [x] Sidebar navigation and Title bar
    - [x] Dark/Light mode toggle (matching original aesthetic)
- [x] **5.2. General Tab**
    - [x] Character selection, Name edit, SteamID edit, Stats edit
- [x] **5.3. Inventory Tab**
    - [x] Item list with search, "Bulk Add" buttons
- [ ] **5.4. World Progress Tab**
    - [ ] Tree view for Graces and Bosses (grouped by region)

## Phase 6: Advanced Tools 🛠️
- [ ] **6.1. Character Importer**
    - [ ] Logic for copying slot + `ProfileSummary` between files
- [ ] **6.2. Slot Management**
    - [ ] Activate/Deactivate slots in `UserData10`

## Phase 7: Quality & Finalization 🚀
- [ ] **7.1. Testing**
    - [ ] Unit tests for binary parsing using files from `tmp/save`
    - [ ] Integration tests for SteamID modification
- [ ] **7.2. Packaging**
    - [ ] `wails build` for Windows (.exe), macOS (.app), Linux
