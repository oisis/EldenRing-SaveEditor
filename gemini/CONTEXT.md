# TECHNICAL CONTEXT: ER-Save-Editor-Go

## 1. Tech Stack
- **Core**: Go (Golang) 1.26+
- **Binary Parsing**: `encoding/binary` (native Go binary parsing).
- **UI**: `Wails` (Go backend + Web frontend: HTML/CSS/JS/React/Svelte) for modern, responsive, and native-feeling UI.
- **Crypto**: `crypto/aes`, `crypto/cipher` (AES-128-CBC for PC saves).
- **Hashing**: `crypto/sha256` (PC) & `crypto/md5` (PS checksums).
- **Packaging**: `make` for cross-platform, single-file native executables.

## 2. Key Architectural Decisions
- **Native Binary Structures**: All save file offsets and structures must be defined in Go structs using `encoding/binary` for direct, fast, and safe memory mapping.
- **Frontend/Backend Split**: Go handles all heavy lifting (binary parsing, crypto, file I/O). The frontend (Wails) handles only presentation and user interaction.
- **Theme**: Modern Web UI with dynamic Light/Dark mode switching.
- **Styling**: **Tailwind CSS v4 ONLY** (strict adherence to new syntax).
- **Backup**: Every "Write" operation must be preceded by an automatic backup of the original file with a timestamp.
- **Integrity Check**: After writing, the application must perform a "Round-trip Validation" (re-reading the saved file and verifying all checksums/offsets) before confirming success to the user.

## 3. Save File Specifications
- **PC (.sl2)**: BND4 container, AES encrypted, MD5 checksums for slots/userdata, SHA256 for BND4 headers.
- **PlayStation**: Decrypted Save Wizard export (raw binary or hex dump in .txt).
- **Regulation**: Section starting at specific offsets (e.g., 0x1960070) with MD5 validation.
- **Source of Truth**: `tmp/repos/Elden-Ring-Save-Editor` (primary reference for binary logic).

## 4. Development Goals
- **Performance**: Instantaneous startup and file processing thanks to compiled Go code.
- **Maintainability**: Easy to update when Elden Ring patches change save offsets.
- **Portability**: Single codebase for Windows, Linux, and macOS (Silicon), compiling to a single binary without external dependencies.
- **UX**: Professional, responsive Web-based UI that feels like a native desktop application.
- **Parity**: 100% functional parity with the Rust implementation in `tmp/repos/Elden-Ring-Save-Editor`.
