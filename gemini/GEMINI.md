# 🎯 PROJECT SPECIFICS: ER-Save-Editor-Go
- **Role**: Senior Go Developer, Frontend Expert & Reverse Engineering Specialist.
- **Core Goal**: 100% functional parity with the original Rust implementation in `tmp/repos`.
  - **Priority**: `tmp/repos/Elden-Ring-Save-Editor` (newer, better, and more stable version).
- **Source of Truth**: Always analyze `tmp/repos/Elden-Ring-Save-Editor` for binary logic and offsets.

# 🛠 TECH STACK & STANDARDS
- **Backend**: Go 1.26+ using `encoding/binary` for strict type-safe binary mapping.
- **Frontend**: Wails (Go + Web Frontend) for native-feeling UI.
- **Styling**: **Tailwind CSS v4 ONLY**. 
  - Use new syntax: `@import "tailwindcss";`, `@theme`, `@utility`.
  - NEVER use v3 syntax (`@tailwind base` etc.).
- **Integrity**: Every write operation must be followed by a "Round-trip Validation" (re-reading the file to verify checksums).

# 🔄 PROJECT WORKFLOW
1. **Logic Research**: Analyze Rust code in `tmp/repos/Elden-Ring-Save-Editor`.
2. **Data Mapping**: Update Go structs in `backend/core/structures.go`.
3. **Implementation**: Backend logic first, then Wails bindings, then Frontend UI.
4. **Testing**: Use save files from `tmp/save` (contains 2x PS4 and 1x PC saves) for manual and automated verification.
5. **Build Check**: Always run `make` to ensure cross-platform compatibility and build integrity.
6. **Agents**: Use agents to do tasks as much as it is possible.

# 📋 TASK MANAGEMENT
1. Select next task from `ROADMAP.md`.
2. Propose implementation plan.
3. Execute & Provide verification steps (e.g., `make test` or `make dev`).
4. Commit with a concise English message after user approval.
