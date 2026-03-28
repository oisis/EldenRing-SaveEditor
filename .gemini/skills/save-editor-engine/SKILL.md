---
name: save-editor-engine
description: Reverse Engineering and Save File Analysis.
---

# save-editor-engine

Expert guidance for reverse engineering game save files.

## Core Workflows

### 1. Hex & Binary Analysis
- **Pattern Discovery**: Identifying recurring structures in hex-dumps.
- **Structure Mapping**: Defining schemas for binary data.

### 2. Checksum & Security
- **Algorithm Identification**: Determining checksum methods (MD5, SHA, XOR).
- **Calculation**: Recalculating checksums after modification.

### 3. Parser & Editor Generation
- **Code Generation**: Creating boilerplate for reading/writing files.
- **GUI Logic**: Designing data models for save editors.

## Standards
- Use **Hex-friendly** output (e.g., `0x00`).
- Document offsets and lengths meticulously.
- Prefer **Construct** (Python) for complex format definitions.

## Common Pitfalls
- **Endianness Errors**: Swapping Little-Endian and Big-Endian data.
- **Padding Misalignment**: Forgetting about byte alignment/padding in structs.
- **Checksum Failures**: Missing a single byte in the calculation range.

## Troubleshooting
- Use `hexdump -C` to inspect file structure.
- Compare modified files with **Original Backups**.
- Log **Calculated vs. Expected** checksums.

## Reference Files
- See `references/elden_ring_offsets.md` for known save offsets.
- See `references/checksum_algorithms.md` for implementations.
