# Project Roadmap: ER-Save-Editor-Go (100% Rust Parity)

> **Status:** ✅ Phase 7.1 Complete | **Source of Truth:** `tmp/org-src` | **Test Strategy:** Round-trip & Golden Files

## Phase 1: Environment & Infrastructure ✅
- [x] **1.1. Go Initialization**
- [x] **1.2. Wails Setup**
- [x] **1.3. Automation & Scripts**
- [x] **1.4. Test Infrastructure**

## Phase 2: Data Extraction (Rust -> Go) ✅
- [x] **2.1. Extraction Tooling**
- [x] **2.2. Constant Extraction**
- [x] **2.3. Stats & Classes**

## Phase 3: Binary Core (The "encoding/binary" Layer) ✅
- [x] **3.1. Crypto Implementation**
- [x] **3.2. Binary Structures (PC)**
- [x] **3.3. Binary Structures (PlayStation)**
- [x] **3.4. SteamID Logic**
- [x] **3.5. Backup System**

## Phase 4: Logic & ViewModel (Go Backend) ✅
- [x] **4.1. Save Manager**
- [x] **4.2. ViewModel Mapping**
- [x] **4.3. Validation Logic**

## Phase 5: UI Implementation (Wails Frontend) ✅
- [x] **5.1. Base Layout**
- [x] **5.2. General Tab**
- [x] **5.3. Inventory Tab**
- [x] **5.4. World Progress Tab**

## Phase 6: Advanced Tools ✅
- [x] **6.1. Character Importer**
- [x] **6.2. Slot Management**

## Phase 7: Quality & Finalization 🚀
- [x] **7.1. Testing**
    - [x] Unit tests for binary parsing using files from `tmp/save`
    - [x] Integration tests for SteamID modification
- [ ] **7.2. Packaging**
    - [ ] `wails build` for Windows (.exe), macOS (.app), Linux
