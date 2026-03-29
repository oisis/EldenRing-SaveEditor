# Elden Ring Save File Specification

This document provides a detailed technical specification of the Elden Ring save file structure for PC (.sl2) and PlayStation 4 (decrypted).

## 1. File Encryption (PC Only)
PC save files are BND4 containers encrypted using AES-128-CBC.

- **AES Key**: `99 AD 2D 50 ED F2 FB 01 C5 F3 EC 3A 2B CA B6 9D`
- **IV (Initialization Vector)**: The first 16 bytes of the `.sl2` file.
- **Payload**: The rest of the file after the first 16 bytes.

## 2. Global Layout & Offsets
After decryption (PC) or in raw format (PS4), the data is organized into sequential blocks.

| Block Name | Size | PS4 Offset | PC Offset* | Description |
| :--- | :--- | :--- | :--- | :--- |
| **SaveHeader** | `0x70` | `0x0000000` | `0x0000000` | Versioning and MapID. |
| **SaveSlot 0** | `0x280000` | `0x0000070` | `0x0000080` | Character 1 data. |
| **SaveSlot 1** | `0x280000` | `0x0280070` | `0x0280090` | Character 2 data. |
| **SaveSlot 2** | `0x280000` | `0x0500070` | `0x05000A0` | Character 3 data. |
| **SaveSlot 3** | `0x280000" | `0x0780070` | `0x07800B0` | Character 4 data. |
| **SaveSlot 4** | `0x280000` | `0x0A00070` | `0x0A000C0` | Character 5 data. |
| **SaveSlot 5** | `0x280000` | `0x0C80070` | `0x0C800D0` | Character 6 data. |
| **SaveSlot 6** | `0x280000` | `0x0F00070` | `0x0F000E0` | Character 7 data. |
| **SaveSlot 7** | `0x280000` | `0x1180070` | `0x11800F0` | Character 8 data. |
| **SaveSlot 8** | `0x280000` | `0x1400070` | `0x1400100` | Character 9 data. |
| **SaveSlot 9** | `0x280000` | `0x1680070` | `0x1680110` | Character 10 data. |
| **UserData10** | `0x60000` | `0x1900070` | `0x19001B0` | Profiles, SteamID, Active Slots. |
| **UserData11** | `0x23FFF0` | `0x1960070` | `0x19601C0` | Regulation.bin / Game Params. |

*\*On PC, each SaveSlot and UserData block is preceded by a 16-byte MD5 checksum, which shifts subsequent offsets.*

## 3. Checksum Mechanism (PC)
Integrity is maintained via MD5 hashes. If a hash is incorrect, the game will report the save as corrupted.

- **Block MD5**: Each block (Slot 0-9, UserData10) starts with 16 bytes of MD5 hash.
- **Calculation**: The MD5 is calculated over the entire block data (e.g., `0x280000` bytes) **excluding** the first 16 bytes (the hash itself).
- **Recalculation**: Any change to character stats, name, or SteamID requires recalculating the MD5 for that specific block.

## 4. Internal Slot Structure (0x280000 bytes)
Key data points within each character slot (offsets relative to Slot Start):

### 4.1 PlayerGameData (Offset: 0x15420)
- `+0x00`: Health/FP/SP (Current, Max, Base Max).
- `+0x30`: Attributes (u32: Vigor, Mind, Endurance, Strength, Dexterity, Intelligence, Faith, Arcane).
- `+0x50`: Level (u32).
- `+0x54`: Souls / Runes (u32).
- `+0x58`: Souls Memory / Total Runes (u32).
- `+0x94`: **Character Name** (UTF-16, 16 chars + null).
- `+0xBC`: Gender (u8: 0=Male, 1=Female).
- `+0xBD`: Archetype / Class (u8).

### 4.2 Advanced Progress & World
- **0x1BF99F**: **EventFlags** start (Size: 0x1BF99F).
- **0x15420 + 0x108**: **NG+ Cycle** (u32: 0=NG, 1=NG+1, etc.).
- **0x15420 + 0x10C**: **Play Time** (u32, in milliseconds).
- **0x15420 + 0x110**: **Death Counter** (u32).

### 4.3 Appearance Data (Offset: 0x15420 + 0x120)
A block of bytes containing all slider values, skin colors, hair types, etc.
- **Size**: ~0x1000 bytes.

## 5. UserData10 (Account Metadata)
- **SteamID**: Offset `0x00` (PS4) or `0x10` (PC). Type: `u64`.
- **Active Slots**: Offset `0x08` (PS4) or `0x18` (PC). Array of 10 bytes (1 = active, 0 = empty).
- **Profile Summary**: 10 blocks (0x120 bytes each) containing character name and level for the main menu.

## 6. Inventory Data (GaItems)
Located before `PlayerGameData`. Each item has a dynamic structure:
- `Handle` (u32)
- `ItemID` (u32)
- **Weapon Specific**: If `(id & 0xf0000000) == 0`:
    - `Upgrade Level`: Encoded in the ItemID (e.g., `ItemID + level`).
    - `Ash of War`: Handle to the assigned AoW.

## 7. Online Safety & Ban Risks (Easy Anti-Cheat)
Modifying the save file carries risks when playing online. EAC validates consistency.

### 7.1 High Risk (Immediate Ban)
- **Impossible Stats**: Level must match the sum of attributes minus starting class base.
- **Illegal Items**: Cut content (e.g., Deathbed Smalls), unreleased DLC items, or items with impossible quantities.
- **Invalid Combinations**: Ash of War applied to an incompatible weapon type.
- **Impossible Spells/Gestures**: Spells added without meeting world-state requirements.

### 7.2 Low Risk (Generally Safe)
- **Runes**: Modifying current rune count (within u32 limits).
- **Consumables**: Adding standard items (Smithing Stones, Arrows) within stack limits.
- **Appearance/Name**: Changing character name or visual sliders.
- **SteamID**: Moving saves between accounts (requires correct MD5 recalculation).

## 8. Safety & Validation
To prevent save corruption, the editor must:
1. Create a backup (`.bak`) before any write operation.
2. Recalculate MD5 checksums for every modified block.
3. Perform a **Round-trip Validation**: Read the file immediately after writing to verify checksums match.
