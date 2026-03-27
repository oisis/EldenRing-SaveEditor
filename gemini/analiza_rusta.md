# Analiza struktury pliku zapisu Elden Ring (PS4 & PC) - KOMPENDIUM

## 1. Główne Bloki Danych i Sumy Kontrolne
### Wersja PlayStation (Raw/Save Wizard)
- **SaveHeader**: `0x70` bajtów.
- **SaveSlot (x10)**: `0x280000` bajtów każdy.
- **UserData10**: `0x60000` bajtów.
- **UserData11**: `0x1c5f70` (Regulation) + `0x7A090` (Rest).
- **Sumy**: Brak sum kontrolnych wewnątrz pliku (obsługiwane przez system PS4/Save Wizard).

### Wersja PC (.sl2)
- **Kontener**: BND4 (wymaga deszyfrowania AES-128-CBC).
- **SaveHeader**: `0x70` bajtów.
- **PCSaveSlot (x10)**: `0x10` (MD5 Hash) + `0x280000` (Dane).
- **PCUserData10**: `0x10` (MD5 Hash) + `0x60000` (Dane + PCOptionData).
- **Weryfikacja**: Każdy blok MD5 jest liczony z danych następujących bezpośrednio po nim.

## 2. Dynamiczna Struktura GaItem (Inventory)
Rozmiar `GaItem` zależy od `item_id` (pierwsze 4 bajty po handle):
- **Broń (`id & 0xf0000000 == 0`)**: 17 bajtów (Handle + ID + unk2 + unk3 + AoW_Handle + unk5).
- **Pancerz (`id & 0xf0000000 == 0x10000000`)**: 16 bajtów (Handle + ID + unk2 + unk3).
- **Inne**: 8 bajtów (Handle + ID).

## 3. SteamID i Tożsamość (PC)
Zmiana SteamID wymaga aktualizacji w:
1. **Każdym SaveSlot**: Pole `u64` na samym końcu struktur (przed paddingiem).
2. **UserData10**: Pole `u64` na początku (offset `0x14` od początku bloku, wliczając MD5).
3. **MD5**: Po zmianie w UserData10 należy przeliczyć hash MD5 całego bloku.

## 4. Reguły Walidacji (Safety First)
Przed zapisem należy sprawdzić:
1. **Kompatybilność AoW**: Czy Popiół Wojny pasuje do typu broni (np. `WepType::Dagger`).
2. **Kategorie Pancerza**: Czy ID pancerza w slocie `Head` to faktycznie `ProtectorCategory::Head`.
3. **Unikalność**: Brak duplikatów `gaitem_handle` w ekwipunku.
4. **Physick**: Brak duplikatów łez w Niesamowitym Eliksirze.

## 5. Baza Danych (DB)
- **Graces/Bosses**: Mapowanie ID flagi (np. `76100`) na bit w tablicy `EventFlags` (`0x1bf99f` bajtów).
- **Stats**: Statyczne tablice HP/FP/SP dla poziomów 1-99.
