# MASTER SPECIFICATION — Elden Ring Save File Format

> **Source of truth**: Go implementation in `backend/core/` + `backend/db/` + Python reference `Final.py`.
> Wszystkie offsety są **little-endian** o ile nie zaznaczono inaczej.
> Dokument opisuje format od zera — wystarczający do napisania edytora bez dostępu do kodu źródłowego.

---

## 1. Platformy i detekcja formatu

Elden Ring używa dwóch różnych formatów pliku save:

| Cecha | PC (Steam) | PS4 |
|---|---|---|
| Nazwa pliku | `ER0000.sl2` | `memory.dat` lub `*.txt` |
| Kontener | BND4 (FromSoftware) | Brak kontenera |
| Szyfrowanie | AES-128-CBC (opcjonalnie) | Brak |
| Wykrycie | `data[0:4] == "BND4"` lub dekrypcja | Pierwsze 4 bajty = `CB 01 9C 2C` |

### 1.1. Detekcja przy wczytywaniu

```
1. Wczytaj cały plik do pamięci.
2. Jeśli data[0:4] == "BND4":
       → PC, niezaszyfrowany. Wczytaj sekwencyjnie (patrz §2.1).
3. Inaczej, spróbuj DecryptSave(data):
       Jeśli wynik[0:4] == "BND4":
       → PC, zaszyfrowany AES-128-CBC. IV = data[0:16]. Wczytaj odszyfrowane.
4. Inaczej:
       → PS4. Wczytaj sekwencyjnie (patrz §2.2).
```

---

## 2. Globalny układ pliku

### 2.1. PC Layout (ER0000.sl2)

```
[0x000]  Header BND4         0x300 bajtów   (patrz §3)
[0x300]  MD5[0]              0x010 bajtów   MD5 slotu 0
[0x310]  SaveSlot[0]         0x280000 baj.
[0x280310] MD5[1]            0x010 bajtów   MD5 slotu 1
[0x280320] SaveSlot[1]       0x280000 baj.
... (×10 slotów, każdy = 0x10 MD5 + 0x280000 danych = 0x280010)
[0x019003A0] MD5[UserData10] 0x010 bajtów
[0x019003B0] UserData10      0x60000 baj.   (konto + profile summaries)
[0x019603B0] UserData11      ~0x240020 baj. (regulation.bin — sieć, parametry)
```

**Rozmiar pliku**: ~28.9 MB (zależnie od UserData11).

### 2.2. PS4 Layout (memory.dat)

```
[0x000]  Header PS4          0x070 bajtów   (stały, patrz §3.2)
[0x070]  SaveSlot[0]         0x280000 baj.  (BEZ prefiksu MD5)
[0x280070] SaveSlot[1]       0x280000 baj.
... (×10 slotów, każdy = 0x280000)
[0x1900070] UserData10       0x60000 baj.   (BEZ prefiksu MD5)
[0x1960070] UserData11       ~0x240020 baj.
```

---

## 3. Nagłówki pliku

### 3.1. PC — BND4 Container Header (0x300 bajtów)

BND4 to standardowy kontener plików FromSoftware. Struktura:

**Blok główny (0x00–0x3F, 64 bajty):**

| Offset | Rozmiar | Wartość | Opis |
|---|---|---|---|
| `0x00` | 4 | `"BND4"` | Magic bytes |
| `0x04` | 4 | `0x00000000` | Zarezerwowane |
| `0x08` | 4 | `0x00010000` | Wersja/flagi |
| `0x0C` | 4 | `12` (u32 LE) | Liczba wpisów w tablicy |
| `0x10` | 8 | `0x40` (u64 LE) | Offset tablicy wpisów |
| `0x18` | 8 | `"00000001"` | ASCII string wersji |
| `0x20` | 8 | `0x20` (u64 LE) | Rozmiar jednego wpisu (32 bajty) |
| `0x28` | 8 | `0x300` (u64 LE) | Offset pierwszych danych (po headerze) |
| `0x30` | 8 | `0x2001` (u64 LE) | Nieznane pole |
| `0x38` | 8 | `0` | Padding |

**Tablica wpisów (0x40–0x1BF, 12 × 32 bajty):**

Każdy wpis opisuje jeden "plik" w kontenerze:

| Offset w wpisie | Rozmiar | Opis |
|---|---|---|
| `+0x00` | 4 | Flagi (`0x50` dla wszystkich wpisów) |
| `+0x04` | 4 | `0xFFFFFFFF` (rozmiar nieskompresowany, N/A) |
| `+0x08` | 4 | Rozmiar danych w pliku (u32 LE) |
| `+0x0C` | 4 | `0` (high word rozmiaru, zawsze 0) |
| `+0x10` | 4 | Offset danych w pliku (u32 LE) |
| `+0x14` | 4 | Offset nazwy w headerze (u32 LE) |
| `+0x18` | 8 | `0` (padding) |

**12 wpisów (indeks → plik):**

| # | Rozmiar | Offset w pliku | Nazwa (UTF-16LE) |
|---|---|---|---|
| 0–9 | `0x280010` | `0x300 + i×0x280010` | `USER_DATA000`–`USER_DATA009` |
| 10 | `0x60010` | `0x300 + 10×0x280010` | `USER_DATA010` |
| 11 | `len(UserData11)` | offset po USER_DATA010 | `USER_DATA011` |

> Rozmiar slotu (0x280010) = 0x10 MD5 + 0x280000 danych.
> Rozmiar UserData10 (0x60010) = 0x10 MD5 + 0x60000 danych.
> Rozmiar UserData11 jest zmienny między save'ami (regulation.bin).

**Tablica nazw (0x1C0–0x2F7):**

Nazwy w UTF-16LE, każda = 12 znaków ASCII + null = 26 bajtów (`0x1A`).
Wzorzec: `USER_DATA` + 3-cyfrowy numer, np. `USER_DATA000`, `USER_DATA011`.

### 3.2. PS4 Header (0x70 bajtów, stały)

Identyczny we wszystkich plikach PS4:

```
CB 01 9C 2C  00 00 00 00  7F 7F 7F 7F  00 00 00 00
07 00 00 00  7F 7F 7F 7F  08 00 00 00  7F 7F 7F 7F
09 00 00 00  7F 7F 7F 7F  0A 00 00 00  7F 7F 7F 7F
0B 00 00 00  7F 7F 7F 7F  0C 00 00 00  7F 7F 7F 7F
0D 00 00 00  7F 7F 7F 7F  0E 00 00 00  7F 7F 7F 7F
0F 00 00 00  7F 7F 7F 7F  10 00 00 00  7F 7F 7F 7F
11 00 00 00  7F 7F 7F 7F  12 00 00 00  7F 7F 7F 7F
```

> Pierwsze 4 bajty `CB 01 9C 2C` pełnią rolę magicu PS4 save.
> Przy konwersji PC→PS4 należy użyć tego stałego headera.
> Przy konwersji PS4→PC należy zbudować BND4 header programatycznie (§3.1).

---

## 4. Kryptografia PC

### 4.1. AES-128-CBC

**Klucz (hardcoded, niezmienny między wersjami gry):**
```
99 AD 2D 50  ED F2 FB 01  C5 F3 EC 3A  2B CA B6 9D
```

**Schemat szyfrowania:**
```
Plik zaszyfrowany = [IV (16 bajtów)] + AES-CBC-Encrypt(key, IV, plaintext)
Plik odszyfrowany = AES-CBC-Decrypt(key, IV=data[0:16], data[16:])
```

- IV jest losowe (generowane przy każdym zapisie dla bezpieczeństwa).
- Plaintext musi być wielokrotnością 16 bajtów (AES block size) — dane BND4 spełniają ten warunek.
- Nie każdy PC save jest zaszyfrowany — starsze wersje gry / niektóre narzędzia zapisują plaintext BND4.

**Detekcja**: jeśli `data[0:4] == "BND4"` → niezaszyfrowany. Inaczej → próbuj dekrypcji.

### 4.2. MD5 Checksums (PC only)

Przed każdym slotu (0–9) oraz przed UserData10 w pliku PC znajdują się 16 bajtów MD5:

```
MD5 = md5(slot_data[0x280000])     // dla slotów
MD5 = md5(userdata10_data[0x60000]) // dla UserData10
```

Przy zapisie pliku MD5 musi być przeliczony po każdej edycji danych.
**UserData11 nie ma prefiksu MD5.**

> ⚠️ Brak SHA256 w headerze BND4 — to mit. Python `Final.py` używa wyłącznie MD5.

---

## 5. SaveSlot — Struktura wewnętrzna

Każdy slot to dokładnie `0x280000` bajtów (2 621 440 bajtów). Struktura jest **dynamiczna** — kluczowe offsety są wyliczane w locie, nie są stałe.

### 5.1. MagicPattern — punkt odniesienia

Lokalizacja inwentarza i statystyk jest względna do `MagicOffset` — adresu wzorca 64-bajtowego:

```
Wzorzec (64 bajty):
00 FF FF FF FF  00 00 00 00 00 00 00 00 00 00 00 00
FF FF FF FF     00 00 00 00 00 00 00 00 00 00 00 00
FF FF FF FF     00 00 00 00 00 00 00 00 00 00 00 00
FF FF FF FF     00 00 00 00 00 00 00 00 00 00 00 00
```

Wzorzec wyszukiwany jest od początku slotu metodą `bytes.Index`.
Jeśli nie znaleziony → fallback: `MagicOffset = 0x15420 + 432`.

### 5.2. PlayerGameData — Statystyki

Wszystkie offsety **relatywne do `MagicOffset`** (ujemne = przed wzorcem):

| Pole | Offset od MagicOffset | Typ | Opis |
|---|---|---|---|
| `CharacterName` | `-0x11B` | `[16]uint16` UTF-16LE | Imię postaci (max 16 znaków) |
| `Vigor` | `-379` | u32 LE | Wytrzymałość |
| `Mind` | `-375` | u32 LE | Umysł |
| `Endurance` | `-371` | u32 LE | Wytrzymałość fizyczna |
| `Strength` | `-367` | u32 LE | Siła |
| `Dexterity` | `-363` | u32 LE | Zręczność |
| `Intelligence` | `-359` | u32 LE | Inteligencja |
| `Faith` | `-355` | u32 LE | Wiara |
| `Arcane` | `-351` | u32 LE | Tajemna |
| `Level` | `-335` | u32 LE | Poziom postaci |
| `Souls` | `-331` | u32 LE | Runy (złoto) |
| `Gender` | `-249` | u8 | 0=Male, 1=Female |
| `Class` | `-248` | u8 | Klasa startowa (0–9) |
| `ScadutreeBlessing` | `-187` | u8 | Poziom błogosławieństwa Scadutree (DLC, max 20) |
| `ShadowRealmBlessing` | `-186` | u8 | Poziom Spirit Ash Blessing (DLC, max 10) |

**Formuła poziomu:**
```
Level = Vigor + Mind + Endurance + Strength + Dexterity + Intelligence + Faith + Arcane - 79
```
(min Level = 1; base sum dla Level 1 to 80, stąd offset -79)

**SteamID w slocie (PC only):**
```
SteamID = slot.Data[0x280000 - 8 : 0x280000]  // u64 LE, ostatnie 8 bajtów slotu
```

### 5.3. GaItems — Tabela przedmiotów (Handle→ItemID)

Sekcja GaItems to tablica par `(Handle, ItemID)` przechowywana przed MagicOffset.
Skanowana od offsetu `0x20` do `MagicOffset`.

**Handle** to identyfikator instancji przedmiotu w grze. Górny nibble (4 bity) określa typ:

| Górny nibble | Typ | Rozmiar rekordu |
|---|---|---|
| `0x8` | Broń (Weapon) | **21 bajtów** |
| `0x9` | Zbroja (Armor) | **16 bajtów** |
| `0xA` | Talizman (Accessory) | 8 bajtów |
| `0xB` | Przedmiot (Item/Consumable) | 8 bajtów |
| `0xC` | Ash of War (AoW) | 8 bajtów |

**Rekord** (każdy zaczyna się od Handle):
```
[0]: Handle  (u32 LE) — typ + unikalny numer instancji
[4]: ItemID  (u32 LE) — ID przedmiotu z bazy danych gry
[8..N]: dodatkowe bajty (0x00) dla Weapon/Armor
```

**Skanowanie:**
```
curr = 0x20
while curr + 8 <= MagicOffset:
    handle = read_u32(data[curr])
    item_id = read_u32(data[curr+4])
    if handle != 0 and handle != 0xFFFFFFFF:
        gaMap[handle] = item_id
        record_size = get_record_size(handle)
        curr += record_size
        InventoryEnd = curr
    else:
        curr += 8
```

`InventoryEnd` — offset za ostatnim prawidłowym rekordem — jest bazą dla dalszych dynamicznych offsetów.

### 5.4. Dynamic Offsets Chain

Wyliczane w oparciu o `InventoryEnd`. Kolejność operacji:

```
PlayerDataOffset        = InventoryEnd + 0x1B0
SP_effect               = PlayerDataOffset + 0xD0
EquipedItemIndex        = SP_effect + 0x58
ActiveEquipedItems      = EquipedItemIndex + 0x1C
EquipedItemsID          = ActiveEquipedItems + 0x58
ActiveEquipedItemsGa    = EquipedItemsID + 0x58
InventoryHeld           = ActiveEquipedItemsGa + 0x9010
EquipedSpells           = InventoryHeld + 0x74
EquipedItems            = EquipedSpells + 0x8C
EquipedGestures         = EquipedItems + 0x18

projSize                = read_u32(data[EquipedGestures])  // dynamiczny rozmiar
EquipedProjectile       = EquipedGestures + projSize*8 + 4
// ⚠️ PS4: jeśli EquipedProjectile >= 0x280000, traktuj projSize=0
//    (garbage w danych PS4 może dać absurdalny projSize)
if EquipedProjectile >= len(data): EquipedProjectile = EquipedGestures + 4
EquipedArmaments        = EquipedProjectile + 0x9C
EquipePhysics           = EquipedArmaments + 0xC
FaceDataOffset          = EquipePhysics + 0x12F
StorageBoxOffset        = FaceDataOffset + 0x6010
```

**EventFlags chain (od StorageBoxOffset):**
```
gesturesOff             = StorageBoxOffset + 0x100
unlockedRegSz           = read_u32(data[gesturesOff])  // dynamiczny!
unlockedRegion          = gesturesOff + unlockedRegSz*4 + 4
// ⚠️ PS4: jeśli unlockedRegion >= 0x280000, traktuj unlockedRegSz=0
//    (ten sam problem co projSize — garbage bytes w PS4 saves)
if unlockedRegion <= 0 or unlockedRegion >= len(data): unlockedRegion = gesturesOff + 4
horse                   = unlockedRegion + 0x29
bloodStain              = horse + 0x4C
menuProfile             = bloodStain + 0x103C
gaItemsOther            = menuProfile + 0x1B588
tutorialData            = gaItemsOther + 0x40B
IngameTimerOffset       = tutorialData + 0x1A         // +0x1A = 26 bajtów
EventFlagsOffset        = IngameTimerOffset + 0x1C0000
```

> ⚠️ EventFlagsOffset **NIE jest stały** — zależy od rozmiaru listy odblokowanych regionów (`unlockedRegSz`). Musi być wyliczany przy każdym wczytaniu.

---

## 6. Inwentarz i Skrzynia

### 6.1. Inwentarz główny (Main Inventory)

```
Offset początku:  MagicOffset + 505
Common Items:     0xA80 (2688) rekordów
Key Items:        0x180 (384) rekordów  — po Common Items
Offset Key Items: (MagicOffset + 505) + 0xA80 * 12
```

Każdy rekord = 12 bajtów:
```
[0]: GaItemHandle  (u32 LE)
[4]: Quantity      (u32 LE)
[8]: Index         (u32 LE)  — numer porządkowy
```

**Pusty slot**: `GaItemHandle == 0` lub `GaItemHandle == 0xFFFFFFFF`.

### 6.2. Skrzynia (Storage Box)

```
Offset początku:  StorageBoxOffset + 4  (pomija 4-bajtowy header)
Pojemność:        2048 rekordów (stała)
Format:           Identyczny jak Main Inventory (12 bajtów/rekord)
Key Items:        Brak — skrzynia nie ma sekcji Key Items
```

> Przy odczycie zatrzymać się przy pierwszym pustym slocie (`handle == 0` lub `0xFFFFFFFF`), żeby uniknąć śmieci z niezainicjalizowanej pamięci.

### 6.3. Mapowanie Handle → Nazwa

Dla broni, zbroi i AoW: nazwa pochodzi z `GaMap[handle] = itemID`, a następnie z bazy danych po `itemID`.
Dla talizmanów i przedmiotów: `GaItemHandle` bezpośrednio to `itemID` (górny nibble to typ, nie instancja).

```
Handle 0x8xxxxxxx → Weapon → itemID = GaMap[handle]
Handle 0x9xxxxxxx → Armor  → itemID = GaMap[handle]
Handle 0xAxxxxxxx → Talisman → itemID = handle (bezpośrednio)
Handle 0xBxxxxxxx → Item    → itemID = handle (bezpośrednio)
Handle 0xCxxxxxxx → Ash of War → itemID = GaMap[handle]
```

---

## 7. Item IDs — Normalizacja i Infuse System

### 7.1. Kategorie ID (górne bity)

| Prefix (górne 4 bity) | Typ | Przykład |
|---|---|---|
| `0x0` | Broń (baza w DB) | `0x00C95000` = Dagger |
| `0x1` | Zbroja | `0x10000000`+ |
| `0x2` | Talizman | `0x20000000`+ |
| `0x4` | Przedmiot / konsumable | `0x40000000`+ |
| `0x8` | AoW (w slocie broń) | `0x80000000`+ |

### 7.2. Upgrade Level

Poziom ulepszenia jest **zakodowany w ID** — dodany jako offset do base ID:

```
FinalID = baseID + upgradeLevel   // np. Dagger +15 = 0x00C95000 + 15
```

Przy odczycie: `level = id - baseID` (gdzie baseID = `id & 0xFFFFFF00` dla ID z górnym bajtem zerowym, lub per-entry lookup w bazie).

### 7.3. Infuse Types

Typ infuzji też jest zakodowany w ID jako offset:

| Typ infuzji | Offset ID |
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
Przykład: Heavy Dagger +10 = 0x00C95000 + 100 + 10 = 0x00C9506E
```

**Warianty infuse w bazie danych**: baza gry zawiera oddzielne wpisy dla każdego wariantu infuzji (Heavy X, Keen X, …). Aby wyświetlać tylko bazowe bronie w UI, należy filtrować warianty: wpis jest wariantem jeśli `id - N×100` (N=1..12) istnieje w tej samej kategorii.

### 7.4. Spirit Ash Upgrade

Spirit Ashes (duchy) używają tego samego mechanizmu co broń — upgrade level dodawany do base ID, max +10.

---

## 8. Dodawanie przedmiotów

Proces dodania przedmiotu do slotu (in-memory, przed zapisem):

```
1. Ustal prefix (typ handle) na podstawie górnych bitów itemID.
2. Wygeneruj unikalny handle: prefix | 0x00010000, inkrementuj aż nie ma kolizji w GaMap.
3. Zapisz GaItem w slot.Data[InventoryEnd]:
       data[InventoryEnd+0] = handle (u32 LE)
       data[InventoryEnd+4] = finalItemID (u32 LE)
       data[InventoryEnd+8..N] = 0x00 (padding do record_size)
       InventoryEnd += record_size
4. Dla stackowalnych (Item, Talisman, AoW):
       Sprawdź czy handle już istnieje w Inventory.CommonItems.
       Jeśli tak → zwiększ Quantity i zaktualizuj data[offset + i*12 + 4].
       Jeśli nie → dodaj nowy rekord w pierwszym wolnym slocie.
5. Dla niestackowalnych (Weapon, Armor):
       Zawsze dodaj nowy rekord.
6. Inventory record (12 bajtów na slot):
       data[startOffset + idx*12 + 0] = GaItemHandle (u32 LE)
       data[startOffset + idx*12 + 4] = Quantity     (u32 LE)
       data[startOffset + idx*12 + 8] = Index        (u32 LE)
```

**startOffset** dla inwentarza: `MagicOffset + 505`.
**startOffset** dla skrzyni: `StorageBoxOffset + 4`.

---

## 9. UserData10 — Konto i Profile Summaries

### 9.1. Layout (0x60000 bajtów)

**PC (offset w bloku UserData10):**

| Offset | Typ | Opis |
|---|---|---|
| `0x00` | u64 LE | **SteamID** — 64-bitowy identyfikator Steam |
| `0x310` | `[10]u8` | **ActiveSlots** — 1=aktywny, 0=pusty (10 bajtów) |
| `0x31A` | `[10]×0x100` | **ProfileSummaries** — dane menu "Load Game" |

**PS4 (offset w bloku UserData10):**

| Offset | Typ | Opis |
|---|---|---|
| `0x300` | `[10]u8` | **ActiveSlots** |
| `0x30A` | `[10]×0x100` | **ProfileSummaries** |

> PS4 nie ma SteamID.

### 9.2. ProfileSummary (0x100 bajtów każda)

```
[0x00]: CharacterName [16]uint16  — UTF-16LE, 32 bajty
[0x20]: Level         uint32 LE   — poziom postaci
[0x24–0xFF]: padding/reserved
```

ProfileSummary jest wyświetlana w menu "Load Game" bez wczytywania pełnego slotu. Musi być zsynchronizowana z danymi slotu po każdej edycji.

---

## 10. Event Flags — Gracje, Bossy, Baseny

### 10.1. Struktura bitowa

Event Flags to bitowa tablica flag stanu świata (odblokowane gracje, pokonani bossowie, aktywne baseny przywoływania).

```
flags = slot.Data[EventFlagsOffset:]   // wyliczony dynamicznie — patrz §5.4
```

**Lokalizacja flagi dla danego ID:**

1. Jeśli ID istnieje w predefiniowanej tabeli `data.EventFlags` → użyj `info.Byte` i `info.Bit` z tabeli.
2. W przeciwnym razie (np. Sites of Grace, ID 0x11558–0x12CA0) → **formuła:**
```
byteIdx = id / 8
bitIdx  = 7 - (id % 8)
```

Flaga jest ustawiona jeśli: `flags[byteIdx] & (1 << bitIdx) != 0`

> Tabela `data.EventFlags` zawiera prekalkulowane wartości dla bossów, baseń, etc.
> Grace IDs **nie są** w tabeli — zawsze używają formuły.

### 10.2. Ustawianie flagi

```
set:   flags[byteIdx] |=  (1 << bitIdx)
clear: flags[byteIdx] &= ^(1 << bitIdx)
```

Modyfikacja odbywa się **in-place** w `slot.Data` — nie wymaga osobnego write-back.

---

## 11. Matchmaking Level

```
Offset: 0x154B3 w slot.Data (stały = PlayerGameDataOffset + 0x93 = 0x15420 + 0x93)
Typ:    u8
```

Określa maksymalny poziom broni gracza dla systemu matchmakingu PvP.
Obliczany jako maksymalny poziom ulepszenia ze wszystkich broni w inwentarzu.
Przy dodaniu broni wyższego poziomu należy zaktualizować jeśli `new_level > current`.

---

## 12. Zapis pliku (Write Flow)

### 12.1. Kolejność operacji

```
1. flushMetadata():
   - Zapisz SteamID do UserData10.Data[0x00] (PC)
   - Zapisz ActiveSlots do UserData10.Data[0x310] (PC) lub [0x300] (PS4)
   - Serializuj ProfileSummaries do UserData10.Data[0x31A] / [0x30A]

2. Zbuduj bufor:
   PC:
     write(header BND4)
     for i in 0..9:
         slotData = slot[i].Write()   // flush stats/name do slot.Data, zwróć Data
         md5 = md5(slotData)
         write(md5)
         write(slotData)
     ud10md5 = md5(UserData10.Data)
     write(ud10md5)
     write(UserData10.Data)
     write(UserData11)
   PS4:
     write(header PS4)
     for i in 0..9:
         write(slot[i].Write())       // bez MD5
     write(UserData10.Data)           // bez MD5
     write(UserData11)

3. Jeśli PC i Encrypted:
     finalData = IV + AES-CBC-Encrypt(key, IV, buffer)
   Inaczej:
     finalData = buffer

4. Atomowy zapis:
     WriteFile(path + ".tmp", finalData)
     Rename(path + ".tmp", path)       // atomowe na POSIX
```

### 12.2. Konwersja platform

**PS4 → PC:**
- Zamień header PS4 (0x70 b) na nowo wygenerowany BND4 header (§3.1).
- Wygeneruj losowe 16-bajtowe IV.
- Ustaw `Encrypted = true`.
- Dodaj prefiksy MD5 przed każdym slotem i UserData10.

**PC → PS4:**
- Zamień header BND4 (0x300 b) na stały PS4 header (§3.2).
- Usuń prefiksy MD5 (nie zapisuj ich).
- Ustaw `Encrypted = false`.

### 12.3. Backup przed zapisem

Jeśli plik docelowy **już istnieje** (nadpisanie):
```
backupPath = originalPath + "." + timestamp + ".bak"  // YYYYMMDD_HHMMSS
CopyFile(originalPath, backupPath)
PruneBackups: zachowaj max 10 ostatnich, usuń starsze
```

Jeśli plik nie istnieje (nowy plik) → pomiń backup.

---

## 13. Baza Danych Przedmiotów

### 13.1. ItemData (per przedmiot)

```go
type ItemData struct {
    Name         string
    Category     string   // granularna kategoria (np. "weapons", "helms")
    MaxInventory uint32   // max quantity w inwentarzu postaci
    MaxStorage   uint32   // max quantity w skrzyni
    MaxUpgrade   uint32   // 0=nieulepszalny, 10=boss/spirit ash, 25=zwykła broń
    IconPath     string   // ścieżka do ikony (np. "items/weapons/dagger.png")
}
```

### 13.2. Limity ilości (EAC Safety)

| Typ | MaxInventory | MaxStorage |
|---|---|---|
| Materiały crafting | 999 | 999 |
| Amunicja | 99 | 600 |
| Konsumable | 10–99 | 600 |
| Broń, Zbroja, Talizman, AoW | 1 | 1 |
| Key Items | 1 | 0 (nie do skrzyni) |

### 13.3. Kategorie

| Kategoria | Handle prefix | Klucz DB |
|---|---|---|
| weapons, bows, shields, staffs, seals | `0x8` | `data.Weapons`, `data.Bows`, ... |
| helms, chest, gauntlets, leggings | `0x9` | `data.Helms`, `data.Chest`, ... |
| talismans | `0xA` | `data.Talismans` |
| consumables, crafting, flasks, … | `0xB` | `data.Consumables`, ... |
| aows (Ashes of War) | `0xC` | `data.Aows` |
| ashes (Spirit Ashes) | `0x8` | `data.StandardAshes` |
| sorceries, incantations | `0xB` | `data.Sorceries`, ... |
| keyitems | `0xB` | `data.Keyitems` |

---

## 14. Bezpieczeństwo online (EAC)

Easy Anti-Cheat skanuje spójność save'a. Zasady bezpiecznej edycji:

1. **Level**: musi równać się `sum(attributes) - 79`. Nigdy nie ustawiaj wyższego niż wynika z atrybutów.
2. **Atrybuty**: każdy w zakresie 1–99.
3. **Bronie**: nie dodawaj broni "Cut Content" (przedmioty usunięte z gry, nieistniejące ID).
4. **Ilości**: nie przekraczaj MaxInventory / MaxStorage.
5. **DLC Scadutree**: nie ustawiaj `ScadutreeBlessing > 20` ani `ShadowRealmBlessing > 10`.
6. **Spójność DLC**: posiadanie wysokiego poziomu Scadutree bez fragmentów w ekwipunku może być wykryte.
7. **Matchmaking Level**: musi odpowiadać faktycznemu najwyższemu poziomowi broni w inwentarzu.

---

## 15. Konwencje binarne

- **Endianness**: wszystkie liczby **little-endian** (LE).
- **Stringi**: UTF-16LE, null-terminated (`uint16 = 0` jako terminator).
- **Flagi**: bitowe, odczyt/zapis przez maskowanie (`|=`, `&= ^`).
- **Adresy**: absolutne od początku `slot.Data` (0x280000 bajtów).
- **Offsety dynamiczne**: wyliczane przy każdym `Read()` — nie cache'ować między sesjami.
