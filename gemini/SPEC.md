# MASTER SPECIFICATION - ELDEN RING SAVE STRUCTURE (IMPLEMENTED)

Ten dokument zawiera specyfikację techniczną struktur binarnych Elden Ring, w 100% zgodną z implementacją w kodzie źródłowym projektu (Go).

## 1. Global File Layout
Plik składa się z nagłówka BND4 (PC), slotów postaci oraz bloków metadanych.

| Block | Size (PS4/memory.dat) | Size (PC/ER0000.sl2) | Description |
| :--- | :--- | :--- | :--- |
| **SaveHeader** | `0x70` | `0x300` | BND4 Container header (PC). |
| **SaveSlot (x10)** | `0x280000` | `0x280010` | Character data. PC includes 16-byte MD5 prefix. |
| **UserData10** | `0x600000` | `0x60010` | Account metadata & Profile Summaries. PC includes MD5. |
| **UserData11** | `0x23FFF0` | `0x23FFF0` | Regulation.bin (NetworkParam, etc.). |

### 1.1. PC Checksum & SteamID
- **Slot MD5**: Każdy slot zaczyna się od 16 bajtów MD5 obliczonego z danych `0x280000` znajdujących się bezpośrednio po nim.
- **SteamID**: Znajduje się na samym końcu każdego slotu (`0x280000 - 8`). Typ: `u64`.
- **BND4 SHA256**: Obliczany dla nagłówka kontenera (pierwsze `0x300` bajtów).

---

## 2. SaveSlot Structure (Dynamic)
Struktura wewnątrz slotu (`0x280000` bajtów) jest wyliczana dynamicznie względem wzorca inwentarza.

### 2.1. Magic Pattern (Inventory Start)
Aplikacja lokalizuje początek inwentarza za pomocą 192-bajtowego wzorca (MagicPattern):
`00 FF FF FF FF 00 00 00 00 00 00 00 00 00 00 00 00 FF FF FF FF ...`
- **MagicOffset**: Adres znalezienia wzorca (domyślnie `0x15420 + 432`).

### 2.2. GaItems (Handles & IDs)
Sekcja znajduje się przed `MagicOffset`. Skanowana od offsetu `0x20` do `MagicOffset`.
- **Dynamiczne rozmiary rekordów**:
    - **Weapon**: 21 bajtów (Handle 0x8...).
    - **Armor**: 16 bajtów (Handle 0x9...).
    - **Inne**: 8 bajtów (Talisman 0xA..., Item 0xB..., AoW 0xC...).
- **InventoryEnd**: Offset ostatniego znalezionego przedmiotu w tej sekcji. Kluczowy dla wyliczania dalszych danych.

### 2.3. PlayerGameData (Stats)
Offsety relatywne do `MagicOffset`:
- **Level**: `MagicOffset - 335` (u32)
- **Vigor**: `MagicOffset - 379` (u32)
- **Mind**: `MagicOffset - 375` (u32)
- **Endurance**: `MagicOffset - 371` (u32)
- **Strength**: `MagicOffset - 367` (u32)
- **Dexterity**: `MagicOffset - 363` (u32)
- **Intelligence**: `MagicOffset - 359` (u32)
- **Faith**: `MagicOffset - 355` (u32)
- **Arcane**: `MagicOffset - 351` (u32)
- **Souls**: `MagicOffset - 331` (u32)
- **Name**: `MagicOffset - 0x11B` (UTF-16LE, 16 chars)
- **Gender**: `MagicOffset - 249` (u8)
- **Class**: `MagicOffset - 248` (u8)

### 2.4. Dynamic Offsets Sequence
Wyliczane w `calculateDynamicOffsets()` na podstawie `InventoryEnd`:
1.  **PlayerDataOffset**: `InventoryEnd + 0x1B0`
2.  **SP_effect**: `PlayerDataOffset + 0xD0`
3.  **Equiped_item_index**: `SP_effect + 0x58`
4.  **Active_equiped_items**: `Equiped_item_index + 0x1C`
5.  **Equiped_items_id**: `Active_equiped_items + 0x58`
6.  **Active_equiped_items_ga**: `Equiped_items_id + 0x58`
7.  **Inventory_held**: `Active_equiped_items_ga + 0x9010`
8.  **Equiped_spells**: `Inventory_held + 0x74`
9.  **Equiped_items**: `Equiped_spells + 0x8C`
10. **Equiped_gestures**: `Equiped_items + 0x18`
11. **Equiped_projectile**: `Equiped_gestures + (count * 8 + 4)`
12. **Equiped_armaments**: `Equiped_projectile + 0x9C`
13. **Equipe_physics**: `Equiped_armaments + 0xC`
14. **FaceDataOffset**: `Equipe_physics + 0x12F`
15. **StorageBoxOffset**: `FaceDataOffset + 0x6010`

---

## 3. Inventory & Storage
- **Main Inventory**: Zaczyna się od `MagicOffset + 505`.
    - **Common Items**: 2688 slotów (0xA80).
    - **Key Items**: 384 sloty (0x180).
    - Każdy rekord: `Handle (u32), Quantity (u32), Index (u32)` = 12 bajtów.
- **Storage Box (Chest)**: Zaczyna się od `StorageBoxOffset + 4`.
    - **Size**: Stałe `0x6000` bajtów (2048 slotów).
    - **Format**: Jedna ciągła lista (bez podziału na Key Items).

---

## 4. Item ID Normalization
Aplikacja stosuje maskowanie prefiksów dla poprawnego mapowania nazw:
- **Weapons**: `id & 0x0FFFFFFF` (usuwa prefiks 0x8 uchwytu).
- **Armor**: `id | 0x10000000` (wymusza prefiks 0x1).
- **Talisman**: `id | 0x20000000` (wymusza prefiks 0x2).
- **Items**: `id | 0x40000000` (wymusza prefiks 0x4).
- **AoW**: `id | 0x80000000` (wymusza prefiks 0x8).

---

## 5. UserData10 (Profile Summaries)
- **SteamID**: Offset `0x10` (PC).
- **Active Slots**: Tablica `[u8; 10]` pod offsetem `0x190`.
- **ProfileSummary**: Dane dla menu "Load Game" (Name, Level).
    - Każdy wpis ma rozmiar `0x100` bajtów.

---

## 6. Shadow of the Erdtree (DLC) Data
Dane specyficzne dla dodatku DLC, zlokalizowane względem `MagicOffset` lub stałych offsetów wewnątrz slotu:
- **Scadutree Blessing Level**: Offset relatywny `0x19188`. Max: 20.
- **Revered Spirit Ash Level**: Offset relatywny `0x1918C`. Max: 10.
- **DLC Item IDs (Prefix 0x40000000)**:
    - Scadutree Fragment: `0x40005140`
    - Revered Spirit Ash: `0x4000514A`

---

## 7. World Progress & Event Flags
- **EventFlags Start**: Offset `0x1BF99F` wewnątrz slotu (może ulegać przesunięciom w zależności od wersji).
- Służą do zarządzania odblokowanymi regionami, bossami i polami przywołań (Summoning Pools).

---

## 8. Online Safety & Ban Risks (EAC)
Aby uniknąć bana przez Easy Anti-Cheat (EAC), należy przestrzegać zasad:
- **Statystyki**: Poziom (Level) musi zawsze zgadzać się z sumą atrybutów minus bazowe statystyki klasy (zazwyczaj suma - 79).
- **Przedmioty**: Unikać dodawania przedmiotów "Cut Content" lub niemożliwych ilości (np. 99 unikalnych broni).
- **Spójność DLC**: Nie ustawiać maksymalnego poziomu Scadutree bez posiadania odpowiednich fragmentów w ekwipunku lub ustawionych flag zdarzeń.

---

## 10. Item Metadata & Database
Baza danych przedmiotów (`backend/db/data`) została rozszerzona o metadane niezbędne do walidacji i poprawnego wyświetlania w UI.

### 10.1. Metadata Structure
Każdy przedmiot w bazie posiada następujące atrybuty:
- **MaxInventory**: Maksymalna ilość w inwentarzu postaci (np. 999 dla materiałów, 99 dla amunicji, 1 dla sprzętu).
- **MaxStorage**: Maksymalna ilość w skrzyni (Storage Box).
- **MaxUpgrade**: Maksymalny poziom ulepszenia (+25 dla broni zwykłych, +10 dla unikalnych i prochów duchów).
- **IconPath**: Statyczna ścieżka do pliku ikony (np. `items/weapons/dagger.png`).

### 10.2. Granular Categorization
Aplikacja implementuje precyzyjny podział na kategorie w celu ułatwienia nawigacji:
- **Equipment**: Weapons, Bows & Ballistae, Shields, Glintstone Staffs, Sacred Seals, Talismans, Ashes of War.
- **Armor**: Helms, Chest Armor, Gauntlets, Leggings.
- **Magic**: Sorceries, Incantations, Spirit Ashes.
- **Items**: Consumables, Crafting Materials, Upgrade Materials, Ammunition.
- **Progress**: Key Items.

### 10.3. Validation Rules (EAC Safety)
- **Materials**: Limit 999/999.
- **Ammo**: Limit 99/600.
- **Consumables**: Zazwyczaj 10-99 w inwentarzu.
- **Key Items**: Zawsze 1/0 (brak możliwości przechowywania w skrzyni).
- **Upgrades**: Blokada ulepszania przedmiotów nieulepszalnych (MaxUpgrade: 0).
