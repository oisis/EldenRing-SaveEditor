# Project Roadmap: ER-Save-Editor-Go (100% Rust Parity)

> **Status:** ✅ Phase 2 Complete | **Source of Truth:** `tmp/org-src` | **Test Strategy:** Round-trip & Golden Files

## Phase 1: Environment & Infrastructure ✅
- [x] **1.1. Go Initialization**
- [x] **1.2. Wails Setup**
- [x] **1.3. Automation & Scripts**
- [x] **1.4. Test Infrastructure**

## Phase 2: Data Extraction (Rust -> Go) ✅
- [x] **2.1. Extraction Tooling**
    - [x] Create Go script to parse `tmp/org-src/src/db/*.rs` and generate Go maps
- [x] **2.2. Constant Extraction**
    - [x] Port item names and IDs from `src/db/*.rs` to Go maps/JSON
    - [x] Port grace and boss flag IDs from `src/db/graces.rs` and `src/db/bosses.rs`
- [x] **2.3. Stats & Classes**
    - [x] Port HP/FP/SP tables from `src/db/stats.rs`
    - [x] Port base class stats from `src/db/classes.rs`

## Phase 3: Binary Core (The "encoding/binary" Layer) 🔧
- [ ] **3.1. Crypto Implementation**
    - [ ] Port AES-128-CBC logic for PC saves (`backend/core/crypto.go`)
    - [ ] **Test:** Verify AES decryption against a known PC save from `tmp/save`
- [ ] **3.2. Binary Structures (PC)**
    - [ ] Define `BND4` container struct
    - [ ] Define `PCSaveSlot` and `PCUserData` structs matching Rust offsets
    - [ ] **Test:** Round-trip validation for PC raw saves
- [ ] **3.3. Binary Structures (PlayStation)**
    - [ ] Define `PSSaveSlot` and `PSUserData` structs
    - [ ] Implement raw binary reading/writing for Save Wizard compatibility
    - [ ] **Test:** Round-trip validation for PS raw saves
- [ ] **3.4. SteamID Logic**
    - [ ] Implement SteamID detection and modification in `UserData10`
    - [ ] **Test:** Verify MD5 checksum recalculation after SteamID change
- [ ] **3.5. Backup System**
    - [ ] Implement automatic `.bak` creation with timestamps before any write

## Phase 4: Logic & ViewModel (Go Backend) 🧠
...
