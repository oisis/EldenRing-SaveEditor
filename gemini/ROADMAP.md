# Project Roadmap: ER-Save-Editor-Python

> **Core Requirement:** 100% functional parity with the original Rust version (`org-src`). All features must be implemented before adding new enhancements.

## Phase 1: Environment & Infrastructure 🏗️
- [ ] Initialize Python environment (Python 3.11+, `uv`).
- [ ] Setup build tool/task runner (`Makefile`).
- [ ] Create directory structure (`src/core`, `src/ui`, `src/utils`, `db`).
- [ ] Configure linting and formatting (`ruff`).

## Phase 2: Data Extraction & Database 📂
- [ ] Extract game data from `org-src/src/db/` (Rust) to JSON files in `db/`.
- [ ] Implement `src/core/database.py` to load and query JSON data.
- [ ] Define item, grace, and boss models.

## Phase 3: Binary Core (The "Construct" Layer) 🔧
- [ ] Implement AES-256-CBC decryption/encryption for PC (.sl2) in `src/core/crypto.py`.
- [ ] Define `construct.Struct` for PC save header and slots in `src/core/structures.py`.
- [ ] Define `construct.Struct` for PlayStation decrypted saves.
- [ ] Implement checksum validation (SHA256 for PC, MD5 for PS).
- [ ] Implement automatic backup logic before write operations.

## Phase 4: Logic & ViewModel 🧠
- [ ] Create `SaveManager` to handle loading/saving/backups.
- [ ] Implement `CharacterViewModel` for mapping raw bytes to UI-friendly objects.
- [ ] Add validation logic (stat limits, name length).
- [ ] Implement "Round-trip Validation" (verify file integrity after write).

## Phase 5: UI Implementation (PySide6) 🎨
- [ ] Setup `MainWindow` with Sidebar navigation.
- [ ] Implement Dark Mode QSS theme.
- [ ] Create General Stats tab (Name, Level, Soul count).
- [ ] Create Equipment/Inventory editor with search functionality.
- [ ] Create World Progress tab (Graces, Bosses).
- [ ] Ensure HiDPI/Retina scaling support.

## Phase 6: Advanced Features & Tools 🛠️
- [ ] Character Importer (copying characters between saves).
- [ ] Character Slot Management (Add new characters to empty slots, Delete existing characters).
- [ ] SteamID changer for PC saves.
- [ ] Bulk item adder.

## Phase 7: Distribution & Quality 🚀
- [ ] Cross-platform packaging using `PyInstaller` (Single File Executable, no external dependencies).
- [ ] Automated tests for binary parsing and checksums.
- [ ] Final I18n pass (Polish/English).
