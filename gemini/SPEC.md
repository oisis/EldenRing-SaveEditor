# Elden Ring Save File Format — Master Specification

> **Scope**: Complete binary format specification for `.sl2` (PC) and `memory.dat` (PS4) save files.
> Sufficient to implement a save editor from scratch without access to game source code.
>
> **Reference implementations**: [ER-Save-Editor](https://github.com/ClayAmore/ER-Save-Editor) (Rust),
> [er-save-manager](https://github.com/Jeius/er-save-manager) (Python),
> Elden-Ring-Save-Editor/Final.py (Python).
>
> All numeric values are **little-endian** unless stated otherwise.

---

## 1. Platforms & Detection

| Property | PC (Steam) | PS4 |
|---|---|---|
| Filename | `ER0000.sl2` | `memory.dat` or `*.txt` |
| Container | BND4 (FromSoftware) | None |
| Encryption | AES-128-CBC (optional) | None |
| Detection | `data[0:4] == "BND4"` or decrypted | `data[0:4] == CB 01 9C 2C` |
| MD5 prefixes | Yes (per slot + UserData10) | No |
| SteamID | In UserData10 at offset `0x00` | N/A |

### Detection Algorithm

```
1. Read entire file into memory.
2. If data[0:4] == "BND4":
       -> PC, unencrypted.
3. Else try DecryptSave(data):
       If result[0:4] == "BND4":
       -> PC, encrypted AES-128-CBC. IV = data[0:16].
4. Else:
       -> PS4.
```

---

## 2. Global File Layout

### 2.1. PC Layout (ER0000.sl2)

```
[0x000]      Header BND4           0x300 bytes    (see S3.1)
[0x300]      MD5[0]                0x010 bytes    MD5 of slot 0
[0x310]      SaveSlot[0]           0x280000 bytes
[0x280310]   MD5[1]                0x010 bytes
[0x280320]   SaveSlot[1]           0x280000 bytes
...          (x10 slots, each = 0x10 MD5 + 0x280000 data = 0x280010)
[0x19003A0]  MD5[UserData10]       0x010 bytes
[0x19003B0]  UserData10            0x60000 bytes  (account + profile summaries)
[0x19603B0]  UserData11            ~0x240020 bytes (regulation.bin)
```

**Slot N offset formula (PC):**
- Checksum: `0x300 + N * 0x280010`
- Data: `0x310 + N * 0x280010`

**Total file size**: ~28.9 MB (varies with UserData11).

### 2.2. PS4 Layout (memory.dat)

```
[0x000]      Header PS4            0x070 bytes    (fixed, see S3.2)
[0x070]      SaveSlot[0]           0x280000 bytes (no MD5 prefix)
[0x280070]   SaveSlot[1]           0x280000 bytes
...          (x10 slots, each = 0x280000)
[0x1900070]  UserData10            0x60000 bytes  (no MD5 prefix)
[0x1960070]  UserData11            ~0x240020 bytes
```

---

## 3. File Headers

### 3.1. PC — BND4 Container Header (0x300 bytes)

BND4 is a standard FromSoftware file container.

**Main block (0x00-0x3F, 64 bytes):**

| Offset | Size | Value | Description |
|---|---|---|---|
| `0x00` | 4 | `"BND4"` | Magic bytes |
| `0x04` | 4 | `0x00000000` | Reserved |
| `0x08` | 4 | `0x00010000` | Version/flags |
| `0x0C` | 4 | `12` (u32 LE) | Number of entries |
| `0x10` | 8 | `0x40` (u64 LE) | Entry table offset |
| `0x18` | 8 | `"00000001"` | ASCII version string |
| `0x20` | 8 | `0x20` (u64 LE) | Entry size (32 bytes) |
| `0x28` | 8 | `0x300` (u64 LE) | First data offset |
| `0x30` | 8 | `0x2001` (u64 LE) | Unknown |
| `0x38` | 8 | `0` | Padding |

**Entry table (0x40-0x1BF, 12 x 32 bytes):**

| Entry offset | Size | Description |
|---|---|---|
| `+0x00` | 4 | Flags (`0x50` for all entries) |
| `+0x04` | 4 | `0xFFFFFFFF` (uncompressed size, N/A) |
| `+0x08` | 4 | Data size in file (u32 LE) |
| `+0x0C` | 4 | `0` (high word of size) |
| `+0x10` | 4 | Data offset in file (u32 LE) |
| `+0x14` | 4 | Name offset in header (u32 LE) |
| `+0x18` | 8 | `0` (padding) |

**12 entries (index -> file):**

| # | Size | Offset | Name (UTF-16LE) |
|---|---|---|---|
| 0-9 | `0x280010` | `0x300 + i*0x280010` | `USER_DATA000`-`USER_DATA009` |
| 10 | `0x60010` | `0x300 + 10*0x280010` | `USER_DATA010` |
| 11 | `len(UserData11)` | after USER_DATA010 | `USER_DATA011` |

> Entry sizes include the 0x10 MD5 prefix: slot = 0x10 + 0x280000, UD10 = 0x10 + 0x60000.

**Name table (0x1C0-0x2F7):** UTF-16LE, each = 12 ASCII chars + null = 26 bytes (`0x1A`).

### 3.2. PS4 Header (0x70 bytes, fixed)

Identical across all PS4 save files:

```
CB 01 9C 2C  00 00 00 00  7F 7F 7F 7F  00 00 00 00
07 00 00 00  7F 7F 7F 7F  08 00 00 00  7F 7F 7F 7F
09 00 00 00  7F 7F 7F 7F  0A 00 00 00  7F 7F 7F 7F
0B 00 00 00  7F 7F 7F 7F  0C 00 00 00  7F 7F 7F 7F
0D 00 00 00  7F 7F 7F 7F  0E 00 00 00  7F 7F 7F 7F
0F 00 00 00  7F 7F 7F 7F  10 00 00 00  7F 7F 7F 7F
11 00 00 00  7F 7F 7F 7F  12 00 00 00  7F 7F 7F 7F
```

> First 4 bytes `CB 01 9C 2C` serve as PS4 magic.
> PS4->PC conversion: build BND4 header programmatically (S3.1).
> PC->PS4 conversion: use this fixed header.

---

## 4. Cryptography (PC Only)

### 4.1. AES-128-CBC

**Key (hardcoded, unchanged across game versions):**
```
99 AD 2D 50  ED F2 FB 01  C5 F3 EC 3A  2B CA B6 9D
```

**Scheme:**
```
Encrypted file  = [IV (16 bytes)] + AES-CBC-Encrypt(key, IV, plaintext)
Decrypted file  = AES-CBC-Decrypt(key, IV=data[0:16], data[16:])
```

- IV is random (generated on each save for security).
- Plaintext must be a multiple of 16 bytes (AES block size) — BND4 data satisfies this.
- Not all PC saves are encrypted — older game versions / some tools write plaintext BND4.

### 4.2. MD5 Checksums

Each slot (0-9) and UserData10 in the PC file have a 16-byte MD5 prefix:

```
MD5 = md5(slot_data[0x280000])      // for slots
MD5 = md5(userdata10_data[0x60000]) // for UserData10
```

Must be recomputed after every modification. **UserData11 has no MD5 prefix.**

---

## 5. SaveSlot — Internal Structure

Each slot is exactly `0x280000` bytes (2,621,440 bytes). The structure is **dynamic** — key offsets are computed at load time, not hardcoded.

### 5.1. Slot Version

```
offset 0x00: version (u32)
  - 0 = empty slot (skip parsing)
  - <= 81: GaItem count = 5118 (0x13FE)
  - > 81:  GaItem count = 5120 (0x1400)
```

### 5.2. MagicPattern — Anchor Point

All stat and inventory offsets are relative to `MagicOffset` — the address of a unique 64-byte pattern:

```
00 FF FF FF FF  00 00 00 00 00 00 00 00 00 00 00 00
FF FF FF FF     00 00 00 00 00 00 00 00 00 00 00 00
FF FF FF FF     00 00 00 00 00 00 00 00 00 00 00 00
FF FF FF FF     00 00 00 00 00 00 00 00 00 00 00 00
```

Found via `bytes.Index` scan from the start of slot data.
Fallback if not found: `MagicOffset = 0x15420 + 432`.

> **Why anchor-based?** Reference editors parse slots sequentially from offset 0x20 — a single
> error in GaItem record size cascades to all subsequent offsets. Our anchor-based approach
> is safer: MagicPattern is unique and immutable, so stat/inventory offsets are always correct
> regardless of GaItem parsing accuracy.

### 5.3. Complete Slot Field Sequence

The table below describes the **exact** field order in a slot, matching the sequential read order of reference editors. Fields with fixed sizes have the size in the "Size" column. Dynamic fields have a formula.

| # | Field | Size | Type |
|---|---|---|---|
| 1 | `version` | 4 | u32 |
| 2 | `map_id` | 4 | u8x4 |
| 3 | `padding_0x18` | 0x18 (24) | bytes |
| 4 | **`gaitem_map`** | **VARIABLE** | GaItem[] — see S6 |
| 5 | `player_game_data` | 0x1B0 (432) | struct — see S5.4 |
| 6 | `sp_effects` | 0xD0 (208) | 13x SPEffect |
| 7 | `equipped_items_equip_index` | 0x58 (88) | ChrAsm |
| 8 | `active_weapon_slots` | 0x1C (28) | struct |
| 9 | `equipped_items_item_id` | 0x58 (88) | ChrAsm |
| 10 | `equipped_items_gaitem_handle` | 0x58 (88) | ChrAsm |
| 11 | **`inventory_held`** | **FIXED** | EquipInventoryData — see S7 |
| 12 | `equipped_spells` | 0x74 (116) | 14 spell slots |
| 13 | `equipped_items` | 0x8C (140) | Quick + pouch |
| 14 | `equipped_gestures` | 0x18 (24) | i32x6 |
| 15 | **`acquired_projectiles`** | **DYNAMIC** | count + array — see S5.6 |
| 16 | `equipped_armaments` | 0x9C (156) | struct |
| 17 | `equipped_physics` | 0x0C (12) | Physick tears |
| 18 | `face_data` | 0x12F (303) | bytes |
| 19 | **`inventory_storage_box`** | **FIXED** | EquipInventoryData — see S7 |
| 20 | `gesture_game_data` | 0x100 (256) | i32x64 |
| 21 | **`unlocked_regions`** | **DYNAMIC** | count + array — see S5.6 |
| 22 | `ride_game_data` | 0x28 (40) | Torrent (horse) |
| 23 | `control_byte` | 1 | u8 |
| 24 | `blood_stain` | 0x44 (68) | struct + 8 bytes padding |
| 25 | `menu_profile_save_load` | ~0x1008 | bytes (4104 B) |
| 26 | `trophy_equip_data` | 0x34 (52) | bytes |
| 27 | **`gaitem_game_data`** | **FIXED** | GaItemData — see S8 |
| 28 | `tutorial_data` | **DYNAMIC** | size + data |
| 29 | `gameman_flags` | 3 | u8x3 |
| 30 | `total_deaths` | 4 | u32 |
| 31 | `character_type` | 4 | i32 |
| 32 | `in_online_session` | 1 | u8 |
| 33 | `character_type_online` | 4 | u32 |
| 34 | `last_rested_grace` | 4 | u32 |
| 35 | `not_alone_flag` | 1 | u8 |
| 36 | `in_game_timer` | 4 | u32 |
| 37 | `padding_4` | 4 | u32 |
| 38 | **`event_flags`** | **0x1BF99F** | 1,833,375 bytes bitfield |
| 39 | `event_flags_terminator` | 1 | u8 |
| 40 | 5x `unknown_list` | **DYNAMIC** | size + data |
| 41 | `player_coordinates` | 0x39 (57) | position + angle |
| 42 | `padding_2` | 2 | bytes |
| 43 | `spawn_point_entity_id` | 4 | u32 |
| 44 | `game_man_0xb64` | 4 | u32 |
| 45 | `temp_spawn_point` | 4 | u32 (version >= 65 only) |
| 46 | `game_man_0xcb3` | 1 | u8 (version >= 66 only) |
| 47 | `net_man` | 0x20004 | 131,076 bytes |
| 48 | `world_area_weather` | 0x0C (12) | struct |
| 49 | `world_area_time` | 0x0C (12) | struct |
| 50 | `base_version` | 0x10 (16) | bytes |
| 51 | `steam_id` | 8 | u64 (per-slot, sequential chain) |
| 52 | `ps5_activity` | 0x20 (32) | bytes |
| 53 | `dlc` | 0x32 (50) | DLC flags — see S5.7 |
| 54 | `player_game_data_hash` | 0x80 (128) | see S9 |
| 55 | `rest` | variable | zero-padded to 0x280000 |

### 5.4. PlayerGameData (Field #5) — 432 bytes (0x1B0)

All offsets are **relative to MagicOffset** (negative = before the pattern):

| Offset from MagicOffset | Absolute PGD offset | Type | Field |
|---|---|---|---|
| `-379` | `0x34` | u32 | Vigor |
| `-375` | `0x38` | u32 | Mind |
| `-371` | `0x3C` | u32 | Endurance |
| `-367` | `0x40` | u32 | Strength |
| `-363` | `0x44` | u32 | Dexterity |
| `-359` | `0x48` | u32 | Intelligence |
| `-355` | `0x4C` | u32 | Faith |
| `-351` | `0x50` | u32 | Arcane |
| `-335` | `0x60` | u32 | Level |
| `-331` | `0x64` | u32 | Souls (Runes) |
| `-283` (`-0x11B`) | `0x94` | [16]uint16 UTF-16LE | CharacterName (32 bytes) |
| `-249` | `0xB6` | u8 | Gender (0=Male, 1=Female) |
| `-248` | `0xB7` | u8 | Class (0-9) |
| `-187` | — | u8 | ScadutreeBlessing (DLC, max 20) |
| `-186` | — | u8 | ShadowRealmBlessing (DLC, max 10) |

**Full PlayerGameData layout (0x1B0 bytes from start of struct):**

| PGD Offset | Type | Field |
|---|---|---|
| 0x00-0x07 | i32, i32 | unknown |
| 0x08-0x0B | u32 | health |
| 0x0C-0x0F | u32 | max_health |
| 0x10-0x13 | u32 | base_max_health |
| 0x14-0x17 | u32 | fp |
| 0x18-0x1B | u32 | max_fp |
| 0x1C-0x1F | u32 | base_max_fp |
| 0x20-0x23 | i32 | unknown |
| 0x24-0x27 | u32 | sp (stamina) |
| 0x28-0x2B | u32 | max_sp |
| 0x2C-0x2F | u32 | base_max_sp |
| 0x30-0x33 | i32 | unknown |
| **0x34** | **u32** | **Vigor** |
| **0x38** | **u32** | **Mind** |
| **0x3C** | **u32** | **Endurance** |
| **0x40** | **u32** | **Strength** |
| **0x44** | **u32** | **Dexterity** |
| **0x48** | **u32** | **Intelligence** |
| **0x4C** | **u32** | **Faith** |
| **0x50** | **u32** | **Arcane** |
| 0x54 | i32 | unknown (Humanity) |
| 0x58-0x5F | i32, i32 | unknown |
| **0x60** | **u32** | **Level** |
| **0x64** | **u32** | **Souls (Runes)** |
| 0x68 | u32 | souls_memory |
| 0x6C-0x93 | bytes (0x28) | padding/unknown |
| **0x94** | **[16]uint16** | **CharacterName** (UTF-16LE, 32 bytes) |
| 0xB4-0xB5 | bytes (2) | padding |
| **0xB6** | **u8** | **Gender** |
| **0xB7** | **u8** | **Class** |
| 0xB8 | u8 | unknown (used in hash computation) |
| 0xB9-0xBA | bytes | unknown |
| 0xBB | u8 | gift |
| 0xBC-0xD9 | bytes (0x1E) | padding/unknown |
| 0xDA | u8 | match_making_wpn_lvl |
| 0xDB-0x10F | bytes (0x35) | padding |
| 0x110-0x17B | bytes (90) | 5x group password (5 x 0x12 = 90 bytes) |
| 0x17C-0x1AF | bytes (0x34) | padding |

**Level formula:**
```
Level = Vigor + Mind + Endurance + Strength + Dexterity + Intelligence + Faith + Arcane - 79
```
(Minimum Level = 1; base attribute sum for Level 1 = 80, hence offset -79.)

### 5.5. Per-Slot SteamID (Field #51)

The per-slot SteamID exists at a dynamic offset within the sequential parsing chain (after `base_version`, before `ps5_activity`). However, it is **NOT the authoritative source** — the authoritative SteamID is stored in UserData10 at offset `0x00` (see S10).

> **Warning:** The per-slot SteamID is NOT at `SlotSize - 8`. That address falls inside the
> `CSPlayerGameDataHash` region (last 0x80 bytes). This is a common misconception.

Our editor does not parse the per-slot SteamID from the sequential chain. It reads the SteamID from UserData10 and propagates it to all slots via `flushMetadata()`.

### 5.6. Dynamic Fields — Projectiles and Unlocked Regions

#### acquired_projectiles (Field #15)

```
projectile_count:  u32          // 4 bytes header
projectiles:       [count]      // count x 8 bytes (projectile_id:u32 + unk:i32)
Total section size: 4 + (count * 8)
```

#### unlocked_regions (Field #21)

```
region_count:  u32          // 4 bytes header
region_ids:    [count]u32   // count x 4 bytes
Total section size: 4 + (count * 4)
```

**How reference editors parse these:**
- Rust / er-save-manager: read count, iterate count times, read stride bytes each.
- Final.py: read count, compute `count * stride + 4`.

All three produce identical results.

> **Our approach:** Since we use MagicOffset as anchor, the cumulative fixed-size sections
> between MagicOffset and these dynamic fields have a known total. We only need to skip
> the 4-byte header (`+4`), not read and multiply the count. See S5.8 for the full chain.

### 5.7. DLC Section (Field #53) — 50 bytes (0x32)

Located at `SlotSize - 0x80 - 0x32` = `0x27FF4E`.

| Byte | Description |
|---|---|
| 0 | Pre-order gesture "The Ring" |
| 1 | Shadow of the Erdtree entry flag (non-zero = entered DLC) |
| 2 | Pre-order gesture "Ring of Miquella" |
| 3-49 | Must be 0x00 |

> **Warning:** DLC byte[1] should be zeroed on platform conversion to prevent infinite loading
> on targets without DLC installed.

### 5.8. Dynamic Offset Chain

Computed from `MagicOffset` as anchor. The chain is a series of fixed-size sections:

```
MagicOffset (= PlayerGameData end / SpEffects start)
    + 0xD0   = EquipedItemIndex
    + 0x58   = ActiveEquipedItems
    + 0x1C   = EquipedItemsID
    + 0x58   = ActiveEquipedItemsGa
    + 0x58   = InventoryHeld
    + 0x9010 = EquipedSpells
    + 0x74   = EquipedItems
    + 0x8C   = EquipedGestures
    + 0x18   = acquired_projectiles header
    + 4      = EquipedProjectile        ← DYNAMIC: skip 4-byte header only
    + 0x9C   = EquipedArmaments
    + 0x0C   = EquipePhysics
    + 0x12F  = FaceDataOffset
    + 0x6010 = StorageBoxOffset
    + 0x100  = GestureGameData -> unlocked_regions header
    + 4      = UnlockedRegion           ← DYNAMIC: skip 4-byte header only
    + 0x29   = Horse
    + 0x4C   = BloodStain
    + 0x103C = MenuProfile
    + 0x1B588 = GaItemDataOffset
    + 0x40B  = TutorialData
    + 0x1A   = IngameTimerOffset
    + 0x1C0000 = EventFlagsOffset
```

**Key computed offsets stored in SaveSlot:**
- `PlayerDataOffset` = MagicOffset
- `FaceDataOffset`
- `StorageBoxOffset`
- `GaItemDataOffset`
- `IngameTimerOffset`
- `EventFlagsOffset`

> Bounds-checked at each step via `SlotAccessor.CheckBounds()`.

---

## 6. GaItems — Item Instance Table (Field #4)

### 6.1. Location

- Start: offset **0x20** in slot data.
- End: just before `player_game_data` (field #5), i.e., `MagicOffset - 0x1B0`.

### 6.2. Entry Count

| Slot version | Max entries |
|---|---|
| <= 81 | 5118 (0x13FE) |
| > 81 | 5120 (0x1400) |

The game reads a **fixed count** of entries. Empty entries (handle=0, itemID=0xFFFFFFFF) fill the remaining space.

### 6.3. Record Format (Variable Length)

Base record: 8 bytes.

```
+-------------------------------+-------------------------------+
| gaitem_handle (4B, u32 LE)    | item_id (4B, u32 LE)         |
+-------------------------------+-------------------------------+
```

Record size depends on **handle type bits** (`handle & 0xF0000000`):

| Type | Handle mask | Record size | Extra fields |
|---|---|---|---|
| Weapon | `0x80000000` | **21 bytes** | unk2(i32), unk3(i32), aow_handle(u32), unk5(u8) |
| Armor | `0x90000000` | **16 bytes** | unk2(i32), unk3(i32) |
| Accessory (Talisman) | `0xA0000000` | **8 bytes** | none |
| Item/Goods | `0xB0000000` | **8 bytes** | none |
| Ash of War | `0xC0000000` | **8 bytes** | none |
| Empty | `0x00000000` | **8 bytes** | item_id = 0xFFFFFFFF |

**Default values for new records:**

| Field | Weapon | Armor | Others |
|---|---|---|---|
| unk2 | 0xFFFFFFFF (-1) | 0xFFFFFFFF (-1) | N/A |
| unk3 | 0xFFFFFFFF (-1) | 0xFFFFFFFF (-1) | N/A |
| aow_handle | 0xFFFFFFFF | N/A | N/A |
| unk5 | 0 | N/A | N/A |

### 6.4. Stackable Items — NOT in GaItems

**Critical:** Stackable items (Talismans with handle prefix `0xA0`, Goods/Items with prefix `0xB0`) are **NEVER stored in the GaItems array**. Real save files contain zero entries with `0xA0` or `0xB0` handle prefixes in GaItems.

For stackable items, the handle IS the item ID with a swapped prefix:
```
handle = (itemID & 0x0FFFFFFF) | handlePrefix
```

The game resolves stackable items directly from the handle — no GaItems lookup needed.

> **Writing GaItem entries for stackable items displaces empty fill markers, causing the game
> to read past the fixed-count section boundary -> EXCEPTION_ACCESS_VIOLATION crash.**

### 6.5. Scanning Algorithm

```
curr = 0x20
gaLimit = MagicOffset - 0x1B0
maxEntries = 5120 (or 5118 for version <= 81)
entriesRead = 0
lastEnd = 0x20

while curr + 8 <= gaLimit AND entriesRead < maxEntries:
    handle = read_u32(data[curr])
    itemID = read_u32(data[curr+4])

    if handle == 0 OR handle == 0xFFFFFFFF:
        curr += 8           // empty: always 8 bytes
        entriesRead++
        continue

    typeBits = handle & 0xF0000000
    switch typeBits:
        0x80000000: recordSize = 21   // Weapon
        0x90000000: recordSize = 16   // Armor
        0xA0000000, 0xB0000000, 0xC0000000: recordSize = 8
        default: break                // unknown type -> stop

    GaMap[handle] = itemID
    curr += recordSize
    lastEnd = curr
    entriesRead++

InventoryEnd = lastEnd
```

### 6.6. Handle Generation

For **non-stackable** items (weapons, armor, AoW):
```
handle = handlePrefix | 0x00010000
while handle exists in GaMap:
    handle++
```

For **stackable** items (talismans, goods):
```
handle = (itemID & 0x0FFFFFFF) | handlePrefix
```

### 6.7. Handle <-> ItemID Prefix Mapping

| Type | Handle prefix | ItemID prefix |
|---|---|---|
| Weapon | `0x80xxxxxx` | `0x00xxxxxx` |
| Armor | `0x90xxxxxx` | `0x10xxxxxx` |
| Accessory | `0xA0xxxxxx` | `0x20xxxxxx` |
| Item/Goods | `0xB0xxxxxx` | `0x40xxxxxx` |
| Ash of War | `0xC0xxxxxx` | `0x80xxxxxx` (Rust) / `0x60xxxxxx` (other refs) |

---

## 7. Inventory & Storage (Fields #11, #19)

### 7.1. Inventory Record — 12 bytes

```
+-------------------------------+-------------------------------+-------------------------------+
| gaitem_handle (4B, u32 LE)    | quantity (4B, u32 LE)         | index (4B, u32 LE)            |
+-------------------------------+-------------------------------+-------------------------------+
```

**Empty slot:** `GaItemHandle == 0` or `GaItemHandle == 0xFFFFFFFF`.

### 7.2. Section Layout

```
common_item_count:        u32                        // header
common_items:             InventoryItem[CAPACITY]    // always full capacity (pre-allocated)
key_item_count:           u32                        // header
key_items:                InventoryItem[CAPACITY]    // always full capacity (pre-allocated)
next_equip_index:         u32                        // trailing counter
next_acquisition_sort_id: u32                        // trailing counter
```

### 7.3. Capacities

| List | Common capacity | Key capacity | Total bytes |
|---|---|---|---|
| **Held (inventory)** | 0xA80 (2688) | 0x180 (384) | 4 + 2688x12 + 4 + 384x12 + 8 = **36,872** |
| **Storage Box** | 0x780 (1920) | 0x80 (128) | 4 + 1920x12 + 4 + 128x12 + 8 = **24,584** |

All reference editors read the **full capacity** — even empty slots. This is a fixed size, not dynamic.

**Inventory start offset:** `MagicOffset + 505`
**Storage start offset:** `StorageBoxOffset + 4` (skip 4-byte header)

### 7.4. Reserved Indices

Indices 0-432 in the `index` field are reserved for equipment slots. New items added via save editor **must** have `index > 432` (`InvEquipReservedMax`).

If `next_equip_index` or `next_acquisition_sort_id` from the save is <= 432, it must be clamped to `max(432, max_existing_index) + 1`.

### 7.5. Trailing Counters

Both `next_equip_index` and `next_acquisition_sort_id` must be incremented and written back to `slot.Data` after adding items. This applies to both inventory and storage paths.

Offsets of trailing counters relative to section start:

**Held inventory** (relative to `MagicOffset + 505`):
```
next_equip_index_off     = CommonItemCount*12 + 4 + KeyItemCount*12
next_acq_sort_id_off     = next_equip_index_off + 4
```

**Storage** (relative to `StorageBoxOffset + 4`):
```
next_equip_index_off     = StorageCommonCount*12 + 4 + StorageKeyCount*12
next_acq_sort_id_off     = next_equip_index_off + 4
```

---

## 8. GaItemData / GaItemGameData (Field #27)

Records every weapon/AoW ID ever acquired. The game looks up weapon properties (reinforce_type, etc.) from this list on load. **Missing entry for a weapon -> crash.**

### 8.1. Layout

```
distinct_acquired_items_count:   i32        // 4 bytes
unk1:                            i32        // 4 bytes
ga_items:                        GaItem2[7000]  // 7000 x 16 = 112,000 bytes
```

**Fixed total size: 112,008 bytes** (8 + 7000 x 16).

### 8.2. GaItem2 Record — 16 bytes

```
id:              u32    // Item ID
unk:             u32    // Unknown
reinforce_type:  u32    // Upgrade/reinforce type
unk1:            u32    // Unknown
```

### 8.3. Upsert Logic

When adding a weapon or AoW (NOT arrows, NOT armor/talismans/goods):

1. Scan `ga_items[0..count]` for matching `item_id`.
2. If not found: append new entry at position `[count]`, increment `count`.
3. `reinforce_type` derived from item ID: `id % 100` gives the upgrade level offset.

---

## 9. PlayerGameDataHash (Field #54) — 128 bytes (0x80)

Located at the last 0x80 bytes of the slot: offset `0x27FF80`.

### 9.1. Algorithm

A custom modified Adler-like checksum using magic constant `0x80078071`. Produces 12 meaningful u32 entries covering level, stats, class, souls, soul memory, equipment, quick items, spells, etc.

The hash function (`bytesHash`) computes a 32-bit value from input bytes using modular reduction with the magic constant.

### 9.2. Write Behavior

**No reference editor recomputes this hash.** All three (Rust, er-save-manager, Final.py) preserve the original bytes verbatim. Our editor follows the same approach — `RecalculateSlotHash()` is implemented and tested but intentionally not called in the write path.

---

## 10. UserData10 — Account & Profile Summaries

### 10.1. Layout (0x60000 bytes)

**PC (offsets within UserData10 block):**

| Offset | Type | Description |
|---|---|---|
| `0x00` | u64 LE | **SteamID** — 64-bit Steam identifier (authoritative) |
| `0x310` | [10]u8 | **ActiveSlots** — 1=active, 0=empty |
| `0x31A` | [10]x0x100 | **ProfileSummaries** — "Load Game" menu data |

**PS4 (offsets within UserData10 block):**

| Offset | Type | Description |
|---|---|---|
| `0x300` | [10]u8 | **ActiveSlots** |
| `0x30A` | [10]x0x100 | **ProfileSummaries** |

> PS4 has no SteamID.

### 10.2. ProfileSummary (0x100 bytes each)

```
[0x00]: CharacterName  [16]uint16 UTF-16LE  (32 bytes)
[0x20]: Level          uint32 LE
[0x24-0xFF]: padding/reserved
```

ProfileSummary is displayed in the "Load Game" menu without loading the full slot. Must be synchronized with slot data after every edit.

---

## 11. Item IDs — Normalization & Infuse System

### 11.1. Two Distinct Prefix Systems

Elden Ring uses **two independent prefix systems** that must not be confused:

#### A) Item ID Prefix — database identifier

| Prefix (upper 4 bits) | Type | Example |
|---|---|---|
| `0x0` | Weapon | `0x00C95000` = Dagger |
| `0x1` | Armor | `0x10000000`+ |
| `0x2` | Accessory (Talisman) | `0x20000000`+ |
| `0x4` | Goods (Consumable) | `0x40000000`+ |
| `0x8` | Ash of War | `0x80000000`+ |

> Arrows/bolts use subtype prefix `0x02`/`0x03` — subtypes of Weapon, NOT separate categories.

#### B) Handle Prefix — GaItem instance identifier

| Prefix (upper 4 bits) | Type | GaItem record size |
|---|---|---|
| `0x8` | Weapon | 21 bytes |
| `0x9` | Armor | 16 bytes |
| `0xA` | Accessory (Talisman) | 8 bytes |
| `0xB` | Item/Goods | 8 bytes |
| `0xC` | Ash of War | 8 bytes |

> Handle prefix != Item ID prefix! Dagger has itemID `0x00C95000` but handle `0x80xxxxxx`.

#### C) Stackable Item Handle Resolution

For non-stackable items (weapons, armor, AoW): handle is a **unique** instance identifier.
`GaMap[handle] -> itemID`. Multiple handles can point to the same itemID.

For stackable items (goods, talismans): handle **IS** the itemID with swapped prefix.
```
handle = (itemID & 0x0FFFFFFF) | handlePrefix
```

The item database stores **only PC-style item IDs** (`0x00`, `0x10`, `0x20`, `0x40`, `0x80`).
Handle<->itemID conversion for stackable items happens at runtime.

### 11.2. Upgrade Level

Encoded directly in the item ID as an offset to the base ID:

```
FinalID = baseID + upgradeLevel
Example: Dagger +15 = 0x00C95000 + 15
```

### 11.3. Infuse Types

Also encoded in the item ID as an offset:

| Infuse type | Offset |
|---|---|
| Standard | +0 |
| Heavy | +100 |
| Keen | +200 |
| Quality | +300 |
| Fire | +400 |
| Flame Art | +500 |
| Lightning | +600 |
| Sacred | +700 |
| Magic | +800 |
| Cold | +900 |
| Poison | +1000 |
| Blood | +1100 |
| Occult | +1200 |

```
FinalID = baseID + infuseOffset + upgradeLevel
Example: Heavy Dagger +10 = 0x00C95000 + 100 + 10 = 0x00C9506E
```

### 11.4. Spirit Ash Upgrade

Spirit Ashes use the same mechanism as weapons — upgrade level added to base ID, max +10.

---

## 12. Adding Items — Algorithm

### 12.1. AddItemsToSlot

```
for each itemID in items:
    1. Convert itemID prefix -> handlePrefix
       (0x00->0x80, 0x10->0x90, 0x20->0xA0, 0x40->0xB0, 0x80->0xC0)

    2. Determine record size: 21 (weapon), 16 (armor), 8 (others)

    3. Check stackability:
       isStackable = (handlePrefix == 0xA0 || handlePrefix == 0xB0)

    4. Search for existing handle (reuse for stackable items)

    5. If no existing handle:
       if isStackable:
           handle = (itemID & 0x0FFFFFFF) | handlePrefix
           GaMap[handle] = itemID
           // NO writeGaItem — stackable items are NOT in GaItems array!
       else:
           handle = generateUniqueHandle(handlePrefix)
           writeGaItem(slot, handle, itemID, recordSize)
           GaMap[handle] = itemID

    6. Register in GaItemData (weapons and AoW only, NOT arrows):
       if (handlePrefix == 0x80 && !isArrow(itemID)) || handlePrefix == 0xC0:
           upsertGaItemData(slot, itemID)

    7. Add to inventory/storage:
       addToInventory(slot, handle, qty, isStorage)
```

### 12.2. writeGaItem

```
1. Check space: InventoryEnd + recordSize <= MagicOffset - 0x1B0
2. Write record at InventoryEnd:
   data[pos+0] = handle    (u32 LE)
   data[pos+4] = itemID    (u32 LE)
   For weapon (21B): data[pos+8..11] = 0xFFFFFFFF, [+12..15] = 0xFFFFFFFF,
                     [+16..19] = 0xFFFFFFFF, [+20] = 0x00
   For armor  (16B): data[pos+8..11] = 0xFFFFFFFF, [+12..15] = 0xFFFFFFFF
3. InventoryEnd += recordSize
4. CRITICAL: Clear remaining GaItem region (InventoryEnd -> gaLimit)
   with empty markers {handle=0x00000000, itemID=0xFFFFFFFF} every 8 bytes.
   This prevents game scanner desync after variable-length weapon/armor records.
```

### 12.3. addToInventory

```
1. Check if handle already exists in inventory (stackable quantity update):
   If found -> SET quantity (not ADD), write to data, return.

2. Compute safe index:
   nextIndex = max(NextEquipIndex, max_existing_index + 1)
   if nextIndex <= 432: nextIndex = 433   // must be > InvEquipReservedMax

3. Storage: append record at end of list
   Inventory: find first empty slot (handle == 0 || 0xFFFFFFFF), overwrite

4. Write record (12 bytes): handle, quantity, index

5. Update BOTH trailing counters:
   NextEquipIndex = nextIndex + 1
   NextAcquisitionSortId = nextIndex + 1
   Write both back to slot.Data
```

---

## 13. Removing Items

Inventory uses a fixed pre-allocated array — zero the matching slot(s).
Storage uses a dynamic list — zero the matching slot(s); game stops reading at handle==0.
GaMap entry is removed only when the handle is absent from both lists after removal.

---

## 14. Event Flags

### 14.1. Structure

Event flags are a bitfield array at `slot.Data[EventFlagsOffset:]` (1,833,375 bytes).

**Locating a flag by ID:**

```
byteIdx = id / 8
bitIdx  = 7 - (id % 8)
```

Flag is set if: `flags[byteIdx] & (1 << bitIdx) != 0`

### 14.2. Setting/Clearing

```
set:   flags[byteIdx] |=  (1 << bitIdx)
clear: flags[byteIdx] &= ^(1 << bitIdx)
```

Modification is **in-place** in `slot.Data` — no separate write-back needed.

> Some flags (bosses, summon pools) have precomputed byte/bit values in the `data.EventFlags` table.
> Grace IDs always use the formula.

---

## 15. Write Flow

### 15.1. Operation Order

```
1. flushMetadata():
   - Write SteamID to UserData10.Data[0x00] (PC only)
   - Write ActiveSlots to UserData10.Data[0x310] (PC) or [0x300] (PS4)
   - Serialize ProfileSummaries to UserData10.Data[0x31A] (PC) or [0x30A] (PS4)

2. Build output buffer:
   PC:
     write(BND4 header)
     for i in 0..9:
         slotData = slot[i].Write()    // flush stats/name to slot.Data, return Data
         md5 = md5(slotData)
         write(md5)                    // 16 bytes
         write(slotData)               // 0x280000 bytes
     ud10md5 = md5(UserData10.Data)
     write(ud10md5)
     write(UserData10.Data)            // 0x60000 bytes
     write(UserData11)
   PS4:
     write(PS4 header)
     for i in 0..9:
         write(slot[i].Write())        // no MD5
     write(UserData10.Data)            // no MD5
     write(UserData11)

3. If PC and Encrypted:
     finalData = IV + AES-CBC-Encrypt(key, IV, buffer)
   Else:
     finalData = buffer

4. Atomic write:
     WriteFile(path + ".tmp", finalData)
     Rename(path + ".tmp", path)
```

### 15.2. SaveSlot.Write() — In-Place Patching

Our editor modifies `slot.Data[]` in-place (unlike Rust/er-save-manager which serialize the entire slot from scratch). Both approaches are correct provided:

- GaItem region is cleaned after InventoryEnd (empty fill markers).
- Trailing counters (NextEquipIndex, NextAcquisitionSortId) are written back to Data[].
- Offset chain is validated before save.

Stats are written as negative offsets from MagicOffset:
```
sa.WriteU32(MagicOffset - 335, Level)
sa.WriteU32(MagicOffset - 379, Vigor)
// ... etc.
```

**Sections NOT modified:**
- PlayerGameDataHash (last 0x80 bytes) — preserved verbatim
- Per-slot SteamID (sequential chain field #51)
- UserData11 (game regulation)
- FaceData (character appearance)
- Event flags terminator + 5x UnknownList + PlayerCoordinates + NetMan

**Sections modified:**
- PlayerGameData (stats) — via Write()
- GaItem map (offset 0x20+) — via writeGaItem()
- Inventory held — via addToInventory()
- Storage box — via addToInventory(isStorage=true)
- GaItemData — via upsertGaItemData()
- Event flags — via SetEventFlag()
- DLC flags — via sanitizeDLCFlags() on platform conversion
- UserData10 — via flushMetadata()
- MD5 checksums (PC) — recomputed in SaveFile()

### 15.3. Backup

If the target file already exists (overwrite):
```
backupPath = originalPath + "." + timestamp + ".bak"   // YYYYMMDD_HHMMSS
CopyFile(originalPath, backupPath)
PruneBackups: keep max 10 most recent, delete older
```

---

## 16. Platform Conversion

### 16.1. PS4 -> PC

- Replace PS4 header (0x70 bytes) with newly generated BND4 header (S3.1).
- Generate random 16-byte IV.
- Set `Encrypted = true`.
- Add MD5 prefixes before each slot and UserData10.

### 16.2. PC -> PS4

- Replace BND4 header (0x300 bytes) with fixed PS4 header (S3.2).
- Remove MD5 prefixes.
- Set `Encrypted = false`.

**Both directions:**
- Slot data (0x280000 bytes) is **identical** on both platforms.
- DLC byte[1] zeroed on conversion (prevents infinite loading without DLC).

---

## 17. Item Database

### 17.1. ItemData Structure

```
Name         string    // Display name
Category     string    // Granular category (e.g., "weapons", "helms")
MaxInventory uint32    // Max quantity in held inventory
MaxStorage   uint32    // Max quantity in storage box
MaxUpgrade   uint32    // 0=not upgradeable, 10=boss/spirit ash, 25=regular weapon
IconPath     string    // Icon path
```

### 17.2. Quantity Limits (EAC Safety)

| Type | MaxInventory | MaxStorage |
|---|---|---|
| Crafting materials | 999 | 999 |
| Ammunition | 99 | 600 |
| Consumables | 10-99 | 600 |
| Weapons, Armor, Talismans, AoW | 1 | 1 |
| Key Items | 1 | 0 (cannot be stored) |

### 17.3. Categories

| Category | Handle prefix | Item ID prefix | DB key |
|---|---|---|---|
| weapons, bows, shields, staffs, seals | `0x8` | `0x0` | `data.Weapons`, `data.Bows`, ... |
| arrows, bolts | `0x8` | `0x0` (sub: `0x02`, `0x03`) | `data.ArrowsAndBolts` |
| helms, chest, gauntlets, leggings | `0x9` | `0x1` | `data.Helms`, `data.Chest`, ... |
| talismans | `0xA` | `0x2` | `data.Talismans` |
| consumables, crafting, flasks, tools | `0xB` | `0x4` | `data.Tools`, `data.Consumables`, ... |
| aows (Ashes of War) | `0xC` | `0x8` | `data.Aows` |
| ashes (Spirit Ashes) | `0xB` | `0x4` | `data.StandardAshes` |
| sorceries, incantations | `0xB` | `0x4` | `data.Sorceries`, ... |
| keyitems | `0xB` | `0x4` | `data.Keyitems` |

> The database stores **only PC-style item IDs**. No "PS4 ID" variants exist.
> Platform conversion for stackable handles (`0x40` <-> `0xB0`) happens at runtime.

---

## 18. EAC Safety Rules

Easy Anti-Cheat scans save consistency. Safe editing rules:

1. **Level**: must equal `sum(attributes) - 79`. Never set higher than attributes allow.
2. **Attributes**: each in range 1-99.
3. **Weapons**: do not add Cut Content weapons (removed from game, nonexistent IDs).
4. **Quantities**: do not exceed MaxInventory / MaxStorage.
5. **DLC Scadutree**: do not set `ScadutreeBlessing > 20` or `ShadowRealmBlessing > 10`.
6. **DLC consistency**: having high Scadutree level without fragments in equipment may be detected.
7. **Matchmaking Level**: must match the actual highest weapon upgrade level in inventory.

---

## 19. Matchmaking Level

```
Offset: PlayerGameData + 0xDA (= MagicOffset - 0xD6)
Type:   u8
```

Determines the player's maximum weapon upgrade level for PvP matchmaking.
Must be updated when adding weapons with higher upgrade level than current value.

---

## 20. Constants Reference

```
// Sizes
SlotSize                = 0x280000    // 2,621,440
PCHeaderSize            = 0x300       // 768
PSHeaderSize            = 0x70        // 112
MD5Size                 = 0x10        // 16
HashSize                = 0x80        // 128
UserData10Size          = 0x60000     // 393,216
EventFlagsSize          = 0x1BF99F    // 1,833,375

// GaItems
GaItemsStart            = 0x20
GaItemCountOld          = 0x13FE      // 5118 (version <= 81)
GaItemCountNew          = 0x1400      // 5120 (version > 81)
GaItemVersionBreak      = 81
GaRecordWeapon          = 21
GaRecordArmor           = 16
GaRecordDefault         = 8

// GaItemData
GaItemDataMaxCount      = 7000        // 0x1B58
GaItemDataEntryLen      = 16
GaItemDataArrayOff      = 8

// Inventory
HeldCommonCapacity      = 0xA80       // 2688
HeldKeyCapacity         = 0x180       // 384
StorageCommonCapacity   = 0x780       // 1920
StorageKeyCapacity      = 0x80        // 128
StorageItemCount        = 2048        // read limit
InvRecordLen            = 12
InvEquipReservedMax     = 432
InvStartFromMagic       = 505

// Handle type masks
HandleWeapon            = 0x80000000
HandleArmor             = 0x90000000
HandleAccessory         = 0xA0000000
HandleItem              = 0xB0000000
HandleAoW               = 0xC0000000
HandleTypeMask          = 0xF0000000
HandleEmpty             = 0x00000000
HandleInvalid           = 0xFFFFFFFF

// ItemID prefixes
ItemIDWeapon            = 0x00000000
ItemIDArmor             = 0x10000000
ItemIDAccessory         = 0x20000000
ItemIDGoods             = 0x40000000
ItemIDAoW               = 0x80000000

// Dynamic offset chain (from MagicOffset)
DynPlayerGameData       = 0x1B0
DynSpEffect             = 0xD0
DynEquipItemIndex       = 0x58
DynActiveWeapon         = 0x1C
DynEquipItemsID         = 0x58
DynActiveEquipGa        = 0x58
DynInventoryHeld        = 0x9010
DynEquipSpells          = 0x74
DynEquipItems           = 0x8C
DynEquipGestures        = 0x18
DynEquipArmaments       = 0x9C
DynEquipPhysics         = 0x0C
DynFaceData             = 0x12F
DynStorageBox           = 0x6010
DynGestureGameData      = 0x100
DynHorse                = 0x29
DynBloodStain           = 0x4C
DynMenuProfile          = 0x103C
DynGaItemsOther         = 0x1B588
DynTutorialData         = 0x40B
DynIngameTimer          = 0x1A
DynEventFlags           = 0x1C0000

// Hash
HashOffset              = 0x27FF80    // SlotSize - HashSize
HashMagic               = 0x80078071

// DLC
DlcSize                 = 0x32        // 50
DlcOffset               = 0x27FF4E   // SlotSize - HashSize - DlcSize
DlcEntryFlagByte        = 1

// PC BND4
BND4Magic               = "BND4"
PS4Magic                = [0xCB, 0x01, 0x9C, 0x2C]

// Sanity limits
MaxProjSize             = 256
MaxUnlockedRegSz        = 1024
MaxHandleAttempts       = 10000

// AES
AESKeySize              = 16
AESBlockSize            = 16
```

---

## 21. Binary Conventions

- **Endianness**: all numbers **little-endian** (LE).
- **Strings**: UTF-16LE, null-terminated (`uint16 = 0` as terminator).
- **Flags**: bitwise, read/write via masking (`|=`, `&= ^`).
- **Addresses**: absolute from start of `slot.Data` (0x280000 bytes).
- **Dynamic offsets**: computed on every `Read()` — do not cache between sessions.
