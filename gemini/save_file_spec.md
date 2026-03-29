# Elden Ring Save File Specification (Verified v1.12+)

This document provides a detailed technical specification of the Elden Ring save file structure for PC (.sl2) and PlayStation 4 (decrypted), updated for the Shadow of the Erdtree DLC.

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
| **SaveSlot 3** | `0x280000` | `0x0780070` | `0x07800B0` | Character 4 data. |
| **SaveSlot 4** | `0x280000` | `0x0A00070` | `0x0A000C0` | Character 5 data. |
| **SaveSlot 5** | `0x280000` | `0x0C80070` | `0x0C800D0` | Character 6 data. |
| **SaveSlot 6** | `0x280000` | `0x0F00070` | `0x0F000E0` | Character 7 data. |
| **SaveSlot 7** | `0x280000` | `0x1180070` | `0x11800F0` | Character 8 data. |
| **SaveSlot 8** | `0x280000` | `0x1400070` | `0x1400100` | Character 9 data. |
| **SaveSlot 9** | `0x280000` | `0x1680070` | `0x1680110` | Character 10 data. |
| **UserData10** | `0x60000` | `0x1900070` | `0x19001B0` | Profiles, SteamID, Active Slots. |
| **UserData11** | `0x240010` | `0x1960070` | `0x19601C0` | Regulation.bin / Game Params. |

*\*On PC, each SaveSlot and UserData block is preceded by a 16-byte MD5 checksum, which shifts subsequent offsets.*

## 3. Checksum Mechanism (PC)
Integrity is maintained via MD5 hashes. If a hash is incorrect, the game will report the save as corrupted.

- **Block MD5**: Each block (Slot 0-9, UserData10) starts with 16 bytes of MD5 hash.
- **Calculation**: The MD5 is calculated over the entire block data (e.g., `0x280000` bytes) **excluding** the first 16 bytes (the hash itself).
- **Recalculation**: Any change to character stats, name, or SteamID requires recalculating the MD5 for that specific block.

## 4. Internal Slot Structure (0x280000 bytes)
**CRITICAL: Internal offsets are DYNAMIC** due to the variable size of the `GaItems` list. Key data points relative to structure starts:

### 4.1 PlayerGameData Structure
- `+0x08`: **Health** (Current, Max, Base Max - `u32` x3).
- `+0x14`: **FP** (Current, Max, Base Max - `u32` x3).
- `+0x24`: **SP** (Current, Max, Base Max - `u32` x3).
- `+0x34`: **Attributes** (u32 x8: Vigor, Mind, Endurance, Strength, Dexterity, Intelligence, Faith, Arcane).
- `+0x60`: **Level** (u32).
- `+0x64`: **Runes** (u32).
- `+0x68`: **Total Runes** (u32).
- `+0x94`: **Character Name** (UTF-16, 16 chars + null).
- `+0xBC`: **Gender** (u8: 0=Male, 1=Female).
- `+0xBD`: **Class** (u8).
- `+0x108`: **NG+ Cycle** (u32: 0=NG, 1=NG+1, etc.).
- `+0x10C`: **Play Time** (u32, in milliseconds).
- `+0x110`: **Death Counter** (u32).

### 4.2 Shadow of the Erdtree (DLC) Data
- **Scadutree Blessing Level**: Located at relative offset `0x19188` (or via `magic_pattern` search). Max: 20.
- **Revered Spirit Ash Level**: Located at relative offset `0x1918C`. Max: 10.
- **DLC Item IDs (Prefix 0x40000000)**:
    - **Scadutree Fragment**: `0x40005140` (20800)
    - **Revered Spirit Ash**: `0x4000514A` (20810)
- **DLC Weapon IDs (Prefix 0x00000000)**:
    - **Milady** (Light Greatsword): `0x00A7D8C0` (11000000)
    - **Dryleaf Arts** (Martial Arts): `0x00B71B00` (12000000)
    - **Great Katana**: `0x00C65D40` (13000000)

### 4.3 Advanced Progress & World
- **0x1BF99F**: **EventFlags** start (Size: 0x1BF99F).
- **Summoning Pools**: Managed via Event Flags.
    - Example Pool `1060420040`: Offset `+0x13b85c`, Bit `7`.
    - Example Pool `31030040`: Offset `+0x170c69`, Bit `7`.
- **Player Coordinates**: `f32[3]` (X, Y, Z) located in `PlayerCoords` block.
- **Tutorials**: `_tutorial_data` (0x408 bytes) - bitflags for shown tutorials.

### 4.4 Appearance Data (Offset: PlayerGameData + 0x120)
A block of bytes containing all slider values, skin colors, hair types, etc.
- **Size**: ~0x12F bytes (Face Data sliders).

### 4.5 Equipment & Inventory
- **GaItems**: Main inventory (dynamic size).
- **Storage Box**: `storage_inventory_data` - separate block for items in the chest.
- **ChrAsm (Character Assembly)**: Detailed mapping of equipped items.
- **Gestures**: `gesture_game_data` - all unlocked gestures.
- **Magic**: `equip_magic_data` - currently equipped spells.

## 5. UserData10 (Account Metadata)
- **SteamID**: Offset `+0x04` (PS4) or `+0x14` (PC). Type: `u64`.
- **Active Slots**: Array of 10 bytes (1 = active, 0 = empty) located after `CSMenuSystemSaveLoad`.
- **Profile Summary**: 10 blocks (`0x24C` / 588 bytes each) containing character name and level for the main menu.

## 6. UserData11 (Regulation)
- **Regulation Data**: Offset `+0x10`. Size: `0x1C5F70`.
- **Description**: Contains a copy of game parameters (equivalent to `regulation.bin`).

## 7. Online Safety & Ban Risks (Easy Anti-Cheat)
Modifying the save file carries risks when playing online. EAC validates consistency.

### 7.1 High Risk (Immediate Ban)
- **Impossible Stats**: Level must match the sum of attributes minus starting class base.
- **Illegal Items**: Cut content, unreleased DLC items, or items with impossible quantities.
- **Blessing Inconsistency**: Setting high Scadutree levels without corresponding world event flags (fragment collection).

### 7.2 Low Risk (Generally Safe)
- **Runes**: Modifying current rune count (within u32 limits).
- **Appearance/Name**: Changing character name or visual sliders.
- **SteamID**: Moving saves between accounts (requires correct MD5 recalculation).

## 8. Safety & Validation
To prevent save corruption, the editor must:
1. Create a backup (`.bak`) before any write operation.
2. Recalculate MD5 checksums for every modified block.
3. Perform a **Round-trip Validation**: Read the file immediately after writing to verify checksums match.
