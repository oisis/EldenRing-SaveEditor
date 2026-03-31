# MASTER SPECIFICATION - ELDEN RING SAVE STRUCTURE (PS4 & PC)

Ten dokument stanowi kompletną specyfikację techniczną struktur binarnych Elden Ring, opracowaną na podstawie analizy kodu źródłowego Rust (`tmp/org-src`).

## 1. Global File Layout
Plik składa się z nagłówka, 10 slotów postaci oraz dwóch bloków metadanych (UserData).

| Block | Size (PS4) | Size (PC) | Description |
| :--- | :--- | :--- | :--- |
| **SaveHeader** | `0x70` | `0x70` | File identification and versioning. |
| **SaveSlot (x10)** | `0x280000` | `0x280010` | Character data. PC includes 16-byte MD5 prefix. |
| **UserData10** | `0x60000` | `0x60010` | Account metadata & Profile Summaries. PC includes MD5. |
| **UserData11** | `0x23FFF0` | `0x23FFF0` | Regulation.bin and additional data. |

---

## 2. SaveSlot Structure (Sequential)
Każdy slot (`0x280000` bajtów) zawiera następujące struktury w podanej kolejności:

1.  **Header**: `ver` (u32), `map_id` ([u8; 4]), `_0x18` ([u8; 0x18]).
2.  **GaItems**: Tablica `0x1400` (5120) elementów. **Dynamiczny rozmiar!**
    - Nagłówek: `handle` (u32), `item_id` (u32).
    - Jeśli broń (`id & 0xf0000000 == 0`): +9 bajtów (`unk2`, `unk3`, `aow_handle`, `unk5`).
    - Jeśli pancerz (`id & 0xf0000000 == 0x10000000`) : +8 bajtów (`unk2`, `unk3`).
    - *Uwaga*: W Go należy użyć logiki warunkowej podczas czytania każdego elementu.
3.  **PlayerGameData**: Statystyki i tożsamość. **Offset: `0x15420`**.
    - `health`, `max_health`, `base_max_health` (3x u32)
    - `fp`, `max_fp`, `base_max_fp` (3x u32)
    - `sp`, `max_sp`, `base_max_sp` (3x u32)
    - **Stats**: `vigor`, `mind`, `endurance`, `strength`, `dexterity`, `intelligence`, `faith`, `arcane` (8x u32)
    - `level` (u32), `souls` (u32), `soulsmemory` (u32)
    - **Character Name**: `[u16; 0x10]` (UTF-16, max 16 znaków). Offset: `PlayerGameData + 0x94`.
    - `gender` (u8), `arche_type` (u8), `gift` (u8)
    - `match_making_wpn_lvl` (u8)
    - **Passwords**: `password`, `group_password[1-5]` (każde `[u8; 0x12]`).
4.  **EquipData / ChrAsm**: Założony ekwipunek (ID przedmiotów).
5.  **Inventory Data**: Listy przedmiotów w ekwipunku i skrzyni.
6.  **EventFlags**: Postęp świata. **Rozmiar: `0x1bf99f`**.
7.  **SteamID (PC)**: Pole `u64` na końcu slotu (przed paddingiem).

---

## 3. UserData10 (Account Metadata)
Blok ten zarządza tym, co gracz widzi w menu głównym.

- **SteamID**: Offset `0x0` (PS4) lub `0x10` (PC). Typ: `u64`.
- **Active Slots**: Tablica `[u8; 0xA]` (1 = aktywny, 0 = pusty).
- **ProfileSummary (x10)**: Skrócone dane postaci dla menu "Load Game".
    - `character_name`: `[u16; 0x11]`
    - `level`: `u32`
    - `equipment_gaitem`: Szczegółowy stan założonych przedmiotów (Handles).
    - `equipment_item`: Szczegółowy stan założonych przedmiotów (IDs).

---

## 4. Checksum & Security (PC Only)
- **MD5**: Każdy blok (Slot, UserData10) na PC zaczyna się od 16-bajtowego hasha MD5.
- **Kolejność zapisu PC**:
    1. Oblicz MD5 z surowych danych bloku (bez pierwszych 16 bajtów).
    2. Zapisz 16 bajtów MD5 na początku.
    3. Zapisz dane bloku.
- **BND4**: Cały plik `.sl2` jest kontenerem BND4, zaszyfrowanym AES-128-CBC.

---

## 5. Validation Rules (Rust Parity)
- **Recalculate Level**: `Level = sum(attributes) - 79`.
- **Weapon Matchmaking**: Skanuj ekwipunek, znajdź max upgrade, zaktualizuj `match_making_wpn_lvl`.
- **AoW Compatibility**: Sprawdzaj `gemMountType` w `Regulation.bin`.
