# MASTER SPECIFICATION - ELDEN RING SAVE STRUCTURE (PS4 & PC)

Ten dokument stanowi kompletną specyfikację techniczną struktur binarnych Elden Ring, opracowaną na podstawie analizy kodu źródłowego Rust (`tmp/org-src`) oraz pełnej analizy repozytorium Pythona (`tmp/repos/Elden-Ring-Save-Editor`).

## 1. Global File Layout
Plik składa się z nagłówka, 10 slotów postaci oraz dwóch bloków metadanych (UserData).

| Block | Size (PS4/memory.dat) | Size (PC/ER0000.sl2) | Description |
| :--- | :--- | :--- | :--- |
| **SaveHeader** | `0x70` | `0x300` | File identification and versioning. |
| **SaveSlot (x10)** | `0x280000` | `0x280010` | Character data. PC includes 16-byte MD5 prefix. |
| **UserData10** | `0x60000` | `0x60010` | Account metadata & Profile Summaries. PC includes MD5. |
| **UserData11** | `0x23FFF0` | `0x23FFF0` | Regulation.bin and additional data. |

### 1.1. PC Checksum Offsets (ER0000.sl2)
- **Slot Checksums**: Zaczynają się od `0x300`, powtarzają się co `0x280010`. Każdy checksum (16 bajtów MD5) dotyczy danych o długości `0x280000` znajdujących się bezpośrednio po nim.
- **General Checksum (UserData10)**: Offset `0x019003A0`. MD5 obliczany z danych od `0x019003B0` do `0x019603AF`.

---

## 2. SaveSlot Structure (Sequential)
Każdy slot (`0x280000` bajtów) zawiera następujące struktury:

### 2.1. Header & Inventory (GaItems)
1.  **Header**: `ver` (u32), `map_id` ([u8; 4]), `_0x18` ([u8; 0x18]).
2.  **GaItems**: Tablica `0x1400` (5120) elementów. **Dynamiczny rozmiar!**
    - **AOB Search (Inventory Start)**: `00 FF FF FF FF 00 00 00 00 00 00 00 00 00 00 00 00 FF FF FF FF ...` (tzw. `magic_pattern`).
        - Python używa wzorca o długości 192 bajtów (powtarzające się `00 FF FF FF FF` z zerami).
        - Go używa wzorca 64-bajtowego.
    - **AOB Search (General)**: `00 00 00 00 ?? 00 !! 00 ?? ?? 00 00 00 00 00 00 ??` (używane do lokalizacji innych sekcji).
    - **Handle Generation**: Uchwyty przedmiotów zaczynają się od `0x80000000`. Nowy uchwyt to `max(existing_handles) + 1`.
    - **Empty Slot**: Oznaczony jako `FF FF FF FF` (uchwyt) i `00 00 00 00` (item_id).
    - **Item Structure**:
        - Nagłówek: `handle` (u32), `item_id` (u32).
        - Jeśli broń (`id & 0xf0000000 == 0`): `id = base_id + upgrade_level`.
        - Popioły Wojny (AoW): Przypisywane do broni pod offsetem `+16` (u32 handle).

### 2.2. PlayerGameData (Stats & Identity)
**Offset: `0x15420`** (lub relatywnie od `magic_pattern`).

| Stat | Relative Offset (Python) | Type | Description |
| :--- | :--- | :--- | :--- |
| **Level** | `-335` | u32 | Character Level. |
| **Vigor** | `-379` | u32 | Vigor Attribute. |
| **Mind** | `-375` | u32 | Mind Attribute. |
| **Endurance** | `-371` | u32 | Endurance Attribute. |
| **Strength** | `-367` | u32 | Strength Attribute. |
| **Dexterity** | `-363` | u32 | Dexterity Attribute. |
| **Intelligence** | `-359` | u32 | Intelligence Attribute. |
| **Faith** | `-355` | u32 | Faith Attribute. |
| **Arcane** | `-351` | u32 | Arcane Attribute. |
| **Souls** | `-331` | u32 | Current Runes. |
| **NG+** | `-280` | u32 | New Game Plus cycle. |
| **Gender** | `-249` | u8 | 0: Female, 1: Male. |
| **Class** | `-248` | u8 | Starting Class ID. |
| **Scadutree Blessing** | `-187` | u8 | DLC Blessing Level. |
| **Shadow Realm Blessing**| `-186` | u8 | DLC Blessing Level. |

**Character Name**: Znajduje się pod offsetem `magic_offset - 0x11b`. Kodowanie UTF-16LE, max 16 znaków.

### 2.3. EventFlags (World Progress)
**Rozmiar: `0x1bf99f`**.
- **Graces**: Odblokowywane bitowo. Mapowanie (map_id, offset, index) znajduje się w `graces.json`.
- **Bosses**: Podobna logika bitowa dla flag zwycięstwa.

### 2.4. Dynamic Offsets (Python save_struct Logic)
Wiele sekcji wewnątrz slotu ma zmienne położenie zależne od rozmiaru poprzednich bloków. Poniżej sekwencja obliczeń:
1.  **GA_item_handle_size**: `max(ga_item_offsets) + 8`.
2.  **Player_data**: `GA_item_handle_size + 0x1B0`.
3.  **SP_effect**: `Player_data + 0xD0`.
4.  **Equiped_item_index**: `SP_effect + 0x58`.
5.  **Active_equiped_items**: `equiped_item_index + 0x1c`.
6.  **Equiped_items_id**: `active_equiped_items + 0x58`.
7.  **Active_equiped_items_ga**: `equiped_items_id + 0x58`.
8.  **Inventory_held**: `active_equiped_items_ga + 0x9010`.
9.  **Equiped_spells**: `inventory_held + 0x74`.
10. **Equiped_items**: `equiped_spells + 0x8c`.
11. **Equiped_gestures**: `equiped_items + 0x18`.
12. **Equiped_projectile**: `equiped_gestures + (struct.unpack("<I", data[equiped_gestures:equiped_gestures+4])[0] * 8 + 4)`.
13. **Equiped_armaments**: `equiped_projectile + 0x9C`.
14. **Equipe_physics**: `equiped_armaments + 0xC`.
15. **Face_data**: `equipe_physics + 0x12f`.
16. **Inventory_storage_box**: `face_data + 0x6010`.
17. **Gestures**: `inventory_storage_box + 0x100`.
18. **Unlocked_region**: `gestures + (struct.unpack("<I", data[gestures:gestures+4])[0] * 4 + 4)`.
19. **Horse**: `unlocked_region + 0x28 + 0x1`.
20. **Blood_stain**: `horse + 0x44 + 0x8`.
21. **Menu_profile**: `blood_stain + 0x1008 + 0x34`.
22. **Ga_items_data_other**: `menu_profile + 0x1b588`.
23. **Tutorial_data**: `ga_items_data_other + 0x408 + 0x3`.
24. **Total_death**: `tutorial_data + 0x4`.
25. **Char_type**: `total_death + 0x4`.
26. **In_online**: `char_type + 0x1`.
27. **Online_char_type**: `in_online + 0x4`.
28. **Last_rested_grace**: `online_char_type + 0x4`.
29. **Not_alone_flag**: `last_rested_grace + 0x1`.
30. **Ingame_timer**: `not_alone_flag + 0x4 + 0x4`.
31. **Event_flag**: `ingame_timer + 0x1bf99f + 0x1`.


---

## 3. UserData10 (Account Metadata)
- **SteamID**: Offset `0x0` (PS4) lub `0x10` (PC). Typ: `u64`.
- **Active Slots**: Tablica `[u8; 0xA]` pod offsetem `0x190` (PC).
- **ProfileSummary**: Dane dla menu "Load Game".
    - Slot 0 Name Offset: `0x190 + 0x28`.
    - Slot 0 Level Offset: `0x190 + 0x28 + 0x22`.

---

## 4. Checksum & Security (PC Only)
- **MD5**: Każdy blok (Slot, UserData10) zaczyna się od 16-bajtowego hasha MD5 obliczonego z pozostałej części bloku.
- **SHA256**: Obliczany dla całego kontenera BND4 (nagłówek pliku .sl2).

---

## 5. Game Database (JSON)
Aplikacja korzysta z zewnętrznych plików JSON do mapowania ID na nazwy:
- `weapons.json`: Mapowanie nazw broni na hex ID (little-endian).
- `armor.json`: Mapowanie pancerzy.
- `goods.json`: Przedmioty użytkowe, czary, klucze.
- `talisman.json`: Talizmany.
- `aow.json`: Popioły Wojny.
- `graces.json`: Miejsca łaski z definicją offsetów i bitów.

---

## 6. Character Importer Logic
1.  Kopiuj cały blok `SaveSlot` (0x280000 bajtów).
2.  Kopiuj odpowiadający mu `ProfileSummary` w `UserData10`.
3.  Zaktualizuj `SteamID` wewnątrz skopiowanego slotu.
4.  Przelicz wszystkie sumy kontrolne MD5 i SHA256.
