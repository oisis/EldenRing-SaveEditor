# REFACTOR.md — Phase 20: Offset Safety & Code Hardening

> **Branch:** `feature/phase20-offset-safety`
> **Status:** 📋 Planned
> **Cel:** Wyeliminować ryzyka uszkodzenia save file przez bounds-checked offset management,
> walidację krzyżową, error propagation zamiast silent fallback, oraz poprawę spójności UI.
> **Zasada:** Zero zmian w binarnym formacie save file. Refaktor dotyczy WYŁĄCZNIE logiki odczytu/zapisu/walidacji.

---

## Spis treści

1. [Kontekst i motywacja](#1-kontekst-i-motywacja)
2. [Etap A: Named Offset Constants](#2-etap-a-named-offset-constants)
3. [Etap B: SlotReader — bounds-checked access](#3-etap-b-slotreader--bounds-checked-access)
4. [Etap C: Error propagation w mapStats/calculateDynamicOffsets](#4-etap-c-error-propagation)
5. [Etap D: Cross-validation — validateOffsetChain](#5-etap-d-cross-validation)
6. [Etap E: Writer safety — writeGaItem/addToInventory/generateUniqueHandle](#6-etap-e-writer-safety)
7. [Etap F: SaveSlot.Warnings — pipeline do UI](#7-etap-f-warnings-pipeline)
8. [Etap G: Frontend hardening](#8-etap-g-frontend-hardening)
9. [Etap H: Testy jednostkowe](#9-etap-h-testy-jednostkowe)
10. [Checklist walidacji po każdym etapie](#10-checklist-walidacji)
11. [Pliki do modyfikacji — mapa zmian](#11-mapa-zmian)
12. [Etap I: SaveManager hardening](#12-etap-i-savemanager-hardening)
13. [Etap J: Database & event flags hardening](#13-etap-j-database--event-flags-hardening)
14. [Etap K: Frontend — performance & UI consistency](#14-etap-k-frontend--performance--ui-consistency)
15. [Anty-wzorce — czego NIE robić](#15-anty-wzorce)

---

## 1. Kontekst i motywacja

### Problem

Obecny system offsetów w `backend/core/structures.go` ma trzy fundamentalne słabości:

1. **Pojedynczy punkt awarii (MagicOffset)** — cały łańcuch offsetów (stats, inventory, storage,
   event flags) zwisa na jednym kotwicy znalezionym przez `FindPattern`. Jeśli pattern trafi źle
   (np. powtarza się w danych gracza), WSZYSTKIE offsety są błędne.

2. **Dynamiczne odczyty z niezaufanych danych bez sanity bounds** — `projSize` (linia 132) i
   `unlockedRegSz` (linia 147) są czytane z surowego save'a. Sprawdzamy jedynie czy wynikowy
   offset mieści się w buforze (`>= len(data)`), ale nie czy wartość jest rozsądna. Na PS4
   to powoduje problemy (garbage bytes dające absurdalne wartości).

3. **Brak walidacji krzyżowej** — po wyliczeniu offsetów nie sprawdzamy ich wzajemnej spójności.
   Nie wiemy czy `StorageBoxOffset > MagicOffset`, czy `EventFlagsOffset < 0x280000`.

### Konsekwencje

| Scenariusz | Teraz | Po refaktorze |
|---|---|---|
| MagicOffset za mały (< 400) | **panic** w `mapStats()` (ujemny index) | error z opisem |
| `projSize` = garbage na PS4 | Silent fallback, dane mogą być błędne | Warning + clamp z logiem |
| `writeGaItem` poza buforem | Buffer overflow → uszkodzony save | Error before write |
| `generateUniqueHandle` z pełnym GaMap | **Infinite loop** | Error po 10000 iteracji |
| Offset chain niemonotoniczny | Silent corruption | Error z nazwami offsetów |

---

## 2. Etap A: Named Offset Constants

### Cel
Jedno źródło prawdy dla WSZYSTKICH magicznych liczb rozrzuconych po `structures.go`, `writer.go`, `character_vm.go`.

### Nowy plik: `backend/core/offset_defs.go`

```go
package core

// SlotSize is the fixed size of each save slot in bytes (2,621,440 = 0x280000).
const SlotSize = 0x280000

// Offsets relative to MagicOffset (negative = before the pattern).
// Source: SPEC.md §5.2 PlayerGameData.
const (
    OffLevel               = -335
    OffVigor               = -379
    OffMind                = -375
    OffEndurance           = -371
    OffStrength            = -367
    OffDexterity           = -363
    OffIntelligence        = -359
    OffFaith               = -355
    OffArcane              = -351
    OffSouls               = -331
    OffGender              = -249
    OffClass               = -248
    OffScadutreeBlessing   = -187
    OffShadowRealmBlessing = -186
    OffCharacterName       = -0x11B // 16 x uint16 UTF-16LE

    // MagicOffset must be at least this value; otherwise negative stat offsets
    // would access memory before the start of the slot buffer.
    MinMagicOffset = 400 // abs(OffVigor) + margin
)

// GaItems section.
const (
    GaItemsStart = 0x20 // scan starts here
)

// GaItem record sizes by handle type prefix (upper nibble).
const (
    GaRecordWeapon    = 21
    GaRecordArmor     = 16
    GaRecordAccessory = 8
    GaRecordItem      = 8
    GaRecordAoW       = 8
)

// Inventory layout (relative to MagicOffset).
const (
    InvStartFromMagic = 505       // MagicOffset + 505
    CommonItemCount   = 0xA80     // 2688 common item slots
    KeyItemCount      = 0x180     // 384 key item slots
    StorageItemCount  = 2048      // storage box capacity
    InvRecordLen      = 12        // bytes per inventory record (handle + qty + index)
)

// Dynamic offset chain constants (relative to InventoryEnd).
// Source: SPEC.md §5.4.
const (
    DynPlayerData         = 0x1B0
    DynSpEffect           = 0xD0
    DynEquipedItemIndex   = 0x58
    DynActiveEquipedItems = 0x1C
    DynEquipedItemsID     = 0x58
    DynActiveEquipedItemsGa = 0x58
    DynInventoryHeld      = 0x9010
    DynEquipedSpells      = 0x74
    DynEquipedItems       = 0x8C
    DynEquipedGestures    = 0x18
    DynEquipedArmaments   = 0x9C
    DynEquipePhysics      = 0x0C
    DynFaceData           = 0x12F
    DynStorageBox         = 0x6010
    DynStorageToGestures  = 0x100
    DynHorse              = 0x29
    DynBloodStain         = 0x4C
    DynMenuProfile        = 0x103C
    DynGaItemsOther       = 0x1B588
    DynTutorialData       = 0x40B
    DynIngameTimer        = 0x1A
    DynEventFlags         = 0x1C0000
)

// Sanity limits for dynamic size reads from untrusted save data.
const (
    MaxProjSize       = 256  // max projectile slots (read from save, PS4 can have garbage)
    MaxUnlockedRegSz  = 1024 // max unlocked region entries
    MaxHandleAttempts = 10000 // max iterations for generateUniqueHandle
)
```

### Instrukcje implementacji

1. Utwórz `backend/core/offset_defs.go` z powyższą zawartością.
2. **NIE** usuwaj jeszcze starych magic numbers z `structures.go` — to zrobimy w Etapie C.
3. Sprawdź: `go build ./backend/core/` — musi się kompilować bez błędów.

### Weryfikacja zgodności z SPEC.md

Każda stała MUSI być zgodna z odpowiednim wpisem w `gemini/SPEC.md` §5.2 i §5.4.
W komentarzu przy każdej stałej podaj sekcję SPEC.md z której pochodzi.

---

## 3. Etap B: SlotReader — bounds-checked access

### Cel
Wrapper na `[]byte` który ZAWSZE sprawdza bounds przed odczytem/zapisem. Eliminuje WSZYSTKIE
potencjalne panic z index-out-of-range.

### Nowy plik: `backend/core/slot_access.go`

```go
package core

import (
    "encoding/binary"
    "fmt"
)

// SlotAccessor provides bounds-checked read/write access to a save slot's raw byte buffer.
// It collects non-fatal warnings (e.g. clamped dynamic sizes) separately from fatal errors.
type SlotAccessor struct {
    Data     []byte
    Warnings []string
}

func NewSlotAccessor(data []byte) *SlotAccessor {
    return &SlotAccessor{Data: data}
}

// ReadU32 reads a little-endian uint32 at the given offset with bounds checking.
func (sa *SlotAccessor) ReadU32(off int) (uint32, error) {
    if off < 0 || off+4 > len(sa.Data) {
        return 0, fmt.Errorf("ReadU32: offset %d (0x%X) out of bounds [0, %d)",
            off, off, len(sa.Data))
    }
    return binary.LittleEndian.Uint32(sa.Data[off:]), nil
}

// ReadU64 reads a little-endian uint64 at the given offset with bounds checking.
func (sa *SlotAccessor) ReadU64(off int) (uint64, error) {
    if off < 0 || off+8 > len(sa.Data) {
        return 0, fmt.Errorf("ReadU64: offset %d (0x%X) out of bounds [0, %d)",
            off, off, len(sa.Data))
    }
    return binary.LittleEndian.Uint64(sa.Data[off:]), nil
}

// ReadU16 reads a little-endian uint16 at the given offset with bounds checking.
func (sa *SlotAccessor) ReadU16(off int) (uint16, error) {
    if off < 0 || off+2 > len(sa.Data) {
        return 0, fmt.Errorf("ReadU16: offset %d (0x%X) out of bounds [0, %d)",
            off, off, len(sa.Data))
    }
    return binary.LittleEndian.Uint16(sa.Data[off:]), nil
}

// ReadU8 reads a single byte at the given offset with bounds checking.
func (sa *SlotAccessor) ReadU8(off int) (uint8, error) {
    if off < 0 || off >= len(sa.Data) {
        return 0, fmt.Errorf("ReadU8: offset %d (0x%X) out of bounds [0, %d)",
            off, off, len(sa.Data))
    }
    return sa.Data[off], nil
}

// WriteU32 writes a little-endian uint32 at the given offset with bounds checking.
func (sa *SlotAccessor) WriteU32(off int, val uint32) error {
    if off < 0 || off+4 > len(sa.Data) {
        return fmt.Errorf("WriteU32: offset %d (0x%X) out of bounds [0, %d)",
            off, off, len(sa.Data))
    }
    binary.LittleEndian.PutUint32(sa.Data[off:], val)
    return nil
}

// WriteU64 writes a little-endian uint64 at the given offset with bounds checking.
func (sa *SlotAccessor) WriteU64(off int, val uint64) error {
    if off < 0 || off+8 > len(sa.Data) {
        return fmt.Errorf("WriteU64: offset %d (0x%X) out of bounds [0, %d)",
            off, off, len(sa.Data))
    }
    binary.LittleEndian.PutUint64(sa.Data[off:], val)
    return nil
}

// WriteU16 writes a little-endian uint16 at the given offset with bounds checking.
func (sa *SlotAccessor) WriteU16(off int, val uint16) error {
    if off < 0 || off+2 > len(sa.Data) {
        return fmt.Errorf("WriteU16: offset %d (0x%X) out of bounds [0, %d)",
            off, off, len(sa.Data))
    }
    binary.LittleEndian.PutUint16(sa.Data[off:], val)
    return nil
}

// WriteU8 writes a single byte at the given offset with bounds checking.
func (sa *SlotAccessor) WriteU8(off int, val uint8) error {
    if off < 0 || off >= len(sa.Data) {
        return fmt.Errorf("WriteU8: offset %d (0x%X) out of bounds [0, %d)",
            off, off, len(sa.Data))
    }
    sa.Data[off] = val
    return nil
}

// ReadDynamicSize reads a uint32 size value from untrusted save data and clamps it
// to a sane maximum. Returns 0 (not error) when clamped, but appends a warning.
// This is the correct behavior for PS4 saves which often have garbage in size fields.
func (sa *SlotAccessor) ReadDynamicSize(off int, maxSize int, name string) (int, error) {
    raw, err := sa.ReadU32(off)
    if err != nil {
        return 0, fmt.Errorf("cannot read %s: %w", name, err)
    }
    size := int(raw)
    if size < 0 || size > maxSize {
        sa.Warnings = append(sa.Warnings,
            fmt.Sprintf("%s: raw value %d (0x%X) exceeds max %d, clamped to 0",
                name, size, size, maxSize))
        return 0, nil
    }
    return size, nil
}

// CheckBounds validates that a write of `size` bytes at `off` is safe.
func (sa *SlotAccessor) CheckBounds(off, size int, label string) error {
    if off < 0 || off+size > len(sa.Data) {
        return fmt.Errorf("%s: offset %d + size %d = %d exceeds buffer length %d",
            label, off, size, off+size, len(sa.Data))
    }
    return nil
}
```

### Instrukcje implementacji

1. Utwórz `backend/core/slot_access.go` z powyższą zawartością.
2. Sprawdź: `go build ./backend/core/`.
3. **NIE** refaktoruj jeszcze `structures.go` — to Etap C.

### Kluczowe decyzje projektowe

- `ReadDynamicSize` zwraca `(0, nil)` a nie error przy clamp — PS4 saves mają garbage
  w polach `projSize`/`unlockedRegSz` i to jest NORMALNE. Error zatrzymałby ładowanie
  normalnego PS4 save'a. Warning jest logowany.
- `Warnings []string` — zbierane per-slot, nie per-operację. Przekazywane do UI w Etapie F.
- `SlotAccessor` operuje na tym samym `[]byte` co `SaveSlot.Data` — zero kopii.

---

## 4. Etap C: Error propagation

### Cel
Zamienić `mapStats()` i `calculateDynamicOffsets()` z funkcji void na zwracające `error`.
Zastąpić bezpośrednie `binary.LittleEndian.*` wywołaniami `SlotAccessor`.
Zastąpić magic numbers stałymi z `offset_defs.go`.

### Modyfikacja: `backend/core/structures.go`

#### 4.1. Dodaj pole `Warnings` do `SaveSlot`

```go
type SaveSlot struct {
    // ... istniejące pola ...
    Warnings []string // non-fatal issues detected during parsing
}
```

#### 4.2. Refaktor `SaveSlot.Read()`

```go
func (s *SaveSlot) Read(r *Reader, platform string) error {
    var err error
    s.Data, err = r.ReadBytes(SlotSize)
    if err != nil {
        return err
    }

    // 1. Find primary anchor
    s.MagicOffset = NewReader(s.Data).FindPattern(MagicPattern)
    if s.MagicOffset == -1 {
        // Fallback — ale z warning, NIE cicho
        s.MagicOffset = 0x15420 + 432
        s.Warnings = append(s.Warnings,
            "MagicPattern not found, using fallback offset 0x15852")
    }
    if s.MagicOffset < MinMagicOffset {
        return fmt.Errorf("MagicOffset %d (0x%X) too small (min %d)",
            s.MagicOffset, s.MagicOffset, MinMagicOffset)
    }

    // 2. Read stats
    if err := s.mapStats(); err != nil {
        return fmt.Errorf("mapStats: %w", err)
    }

    // 3. Scan GaItems
    s.scanGaItems(GaItemsStart)

    // 4. Calculate dynamic offsets
    if err := s.calculateDynamicOffsets(); err != nil {
        return fmt.Errorf("dynamic offsets: %w", err)
    }

    // 5. Cross-validate (Etap D)
    if err := s.validateOffsetChain(); err != nil {
        return fmt.Errorf("offset validation: %w", err)
    }

    // 6. Map inventory
    s.mapInventory()

    if platform == "PC" {
        sa := NewSlotAccessor(s.Data)
        steamID, err := sa.ReadU64(SlotSize - 8)
        if err != nil {
            return fmt.Errorf("SteamID read: %w", err)
        }
        s.SteamID = steamID
    }
    return nil
}
```

#### 4.3. Refaktor `mapStats()` → zwraca `error`

```go
func (s *SaveSlot) mapStats() error {
    sa := NewSlotAccessor(s.Data)
    mo := s.MagicOffset
    var err error

    if s.Player.Level, err = sa.ReadU32(mo + OffLevel); err != nil {
        return fmt.Errorf("Level: %w", err)
    }
    if s.Player.Vigor, err = sa.ReadU32(mo + OffVigor); err != nil {
        return fmt.Errorf("Vigor: %w", err)
    }
    if s.Player.Mind, err = sa.ReadU32(mo + OffMind); err != nil {
        return fmt.Errorf("Mind: %w", err)
    }
    if s.Player.Endurance, err = sa.ReadU32(mo + OffEndurance); err != nil {
        return fmt.Errorf("Endurance: %w", err)
    }
    if s.Player.Strength, err = sa.ReadU32(mo + OffStrength); err != nil {
        return fmt.Errorf("Strength: %w", err)
    }
    if s.Player.Dexterity, err = sa.ReadU32(mo + OffDexterity); err != nil {
        return fmt.Errorf("Dexterity: %w", err)
    }
    if s.Player.Intelligence, err = sa.ReadU32(mo + OffIntelligence); err != nil {
        return fmt.Errorf("Intelligence: %w", err)
    }
    if s.Player.Faith, err = sa.ReadU32(mo + OffFaith); err != nil {
        return fmt.Errorf("Faith: %w", err)
    }
    if s.Player.Arcane, err = sa.ReadU32(mo + OffArcane); err != nil {
        return fmt.Errorf("Arcane: %w", err)
    }
    if s.Player.Souls, err = sa.ReadU32(mo + OffSouls); err != nil {
        return fmt.Errorf("Souls: %w", err)
    }
    if s.Player.Gender, err = sa.ReadU8(mo + OffGender); err != nil {
        return fmt.Errorf("Gender: %w", err)
    }
    if s.Player.Class, err = sa.ReadU8(mo + OffClass); err != nil {
        return fmt.Errorf("Class: %w", err)
    }
    if s.Player.ScadutreeBlessing, err = sa.ReadU8(mo + OffScadutreeBlessing); err != nil {
        return fmt.Errorf("ScadutreeBlessing: %w", err)
    }
    if s.Player.ShadowRealmBlessing, err = sa.ReadU8(mo + OffShadowRealmBlessing); err != nil {
        return fmt.Errorf("ShadowRealmBlessing: %w", err)
    }

    // Character name (16 x uint16 UTF-16LE)
    nameOff := mo + OffCharacterName
    for i := 0; i < 16; i++ {
        val, err := sa.ReadU16(nameOff + i*2)
        if err != nil {
            return fmt.Errorf("CharacterName[%d]: %w", i, err)
        }
        s.Player.CharacterName[i] = val
    }

    return nil
}
```

#### 4.4. Refaktor `calculateDynamicOffsets()` → zwraca `error`

```go
func (s *SaveSlot) calculateDynamicOffsets() error {
    sa := NewSlotAccessor(s.Data)

    s.PlayerDataOffset = s.InventoryEnd + DynPlayerData

    spEffect := s.PlayerDataOffset + DynSpEffect
    equipedItemIndex := spEffect + DynEquipedItemIndex
    activeEquipedItems := equipedItemIndex + DynActiveEquipedItems
    equipedItemsID := activeEquipedItems + DynEquipedItemsID
    activeEquipedItemsGa := equipedItemsID + DynActiveEquipedItemsGa
    inventoryHeld := activeEquipedItemsGa + DynInventoryHeld
    equipedSpells := inventoryHeld + DynEquipedSpells
    equipedItems := equipedSpells + DynEquipedItems
    equipedGestures := equipedItems + DynEquipedGestures

    // Dynamic read #1: projSize
    projSize, err := sa.ReadDynamicSize(equipedGestures, MaxProjSize, "projSize")
    if err != nil {
        return err
    }
    equipedProjectile := equipedGestures + projSize*8 + 4

    equipedArmaments := equipedProjectile + DynEquipedArmaments
    equipePhysics := equipedArmaments + DynEquipePhysics
    s.FaceDataOffset = equipePhysics + DynFaceData
    s.StorageBoxOffset = s.FaceDataOffset + DynStorageBox

    // EventFlags offset chain
    gesturesOff := s.StorageBoxOffset + DynStorageToGestures
    if err := sa.CheckBounds(gesturesOff, 4, "gesturesOff"); err != nil {
        s.Warnings = append(s.Warnings, "EventFlags chain unreachable: "+err.Error())
        return nil // non-fatal — event flags are optional for basic editing
    }

    // Dynamic read #2: unlockedRegSz
    unlockedRegSz, err := sa.ReadDynamicSize(gesturesOff, MaxUnlockedRegSz, "unlockedRegSz")
    if err != nil {
        return err
    }
    unlockedRegion := gesturesOff + unlockedRegSz*4 + 4

    horse := unlockedRegion + DynHorse
    bloodStain := horse + DynBloodStain
    menuProfile := bloodStain + DynMenuProfile
    gaItemsOther := menuProfile + DynGaItemsOther
    tutorialData := gaItemsOther + DynTutorialData
    s.IngameTimerOffset = tutorialData + DynIngameTimer
    s.EventFlagsOffset = s.IngameTimerOffset + DynEventFlags

    // Collect SlotAccessor warnings
    s.Warnings = append(s.Warnings, sa.Warnings...)
    return nil
}
```

#### 4.5. Refaktor `Write()` — użyj SlotAccessor

```go
func (s *SaveSlot) Write(platform string) []byte {
    sa := NewSlotAccessor(s.Data)
    mo := s.MagicOffset

    // Errors in Write are programming bugs (offsets already validated in Read),
    // so we use must-style helpers. If any fails, it means Read() had a bug.
    sa.WriteU32(mo+OffLevel, s.Player.Level)
    sa.WriteU32(mo+OffVigor, s.Player.Vigor)
    sa.WriteU32(mo+OffMind, s.Player.Mind)
    sa.WriteU32(mo+OffEndurance, s.Player.Endurance)
    sa.WriteU32(mo+OffStrength, s.Player.Strength)
    sa.WriteU32(mo+OffDexterity, s.Player.Dexterity)
    sa.WriteU32(mo+OffIntelligence, s.Player.Intelligence)
    sa.WriteU32(mo+OffFaith, s.Player.Faith)
    sa.WriteU32(mo+OffArcane, s.Player.Arcane)
    sa.WriteU32(mo+OffSouls, s.Player.Souls)
    sa.WriteU8(mo+OffGender, s.Player.Gender)
    sa.WriteU8(mo+OffClass, s.Player.Class)
    sa.WriteU8(mo+OffScadutreeBlessing, s.Player.ScadutreeBlessing)
    sa.WriteU8(mo+OffShadowRealmBlessing, s.Player.ShadowRealmBlessing)

    nameOff := mo + OffCharacterName
    for i := 0; i < 16; i++ {
        sa.WriteU16(nameOff+i*2, s.Player.CharacterName[i])
    }

    if platform == "PC" {
        sa.WriteU64(SlotSize-8, s.SteamID)
    }
    return s.Data
}
```

### WAŻNE — Kolejność refaktoru w pliku

1. Najpierw dodaj `Warnings` do struct `SaveSlot`.
2. Potem zmień sygnaturę `mapStats()` na `error` i ciało.
3. Potem zmień sygnaturę `calculateDynamicOffsets()` na `error` i ciało.
4. Potem zmień `Read()` aby obsłużyć nowe error returns.
5. Potem zmień `Write()`.
6. Na końcu `mapInventory()` — użyj stałych `InvStartFromMagic`, `CommonItemCount`, `KeyItemCount`.
7. Kompiluj po KAŻDYM kroku: `go build ./backend/core/`.

### Pliki wymagające aktualizacji po zmianie sygnatur

Po dodaniu `error` return do `mapStats()` i `calculateDynamicOffsets()`,
**NIE MA** innych plików do zmiany — obie funkcje są prywatne i wywoływane
WYŁĄCZNIE z `SaveSlot.Read()`.

Po dodaniu `Warnings []string` do `SaveSlot`, nie trzeba nic zmieniać w istniejących
plikach — pole jest nowe, domyślnie `nil`.

---

## 5. Etap D: Cross-validation

### Cel
Po wyliczeniu wszystkich offsetów, przed użyciem ich do odczytu inventory, walidujemy
że tworzą sensowny, monotoniczny łańcuch w obrębie bufora slotu.

### Nowa funkcja w `backend/core/structures.go`

```go
// validateOffsetChain verifies that all computed offsets are within bounds
// and in the expected monotonic order. Called after calculateDynamicOffsets().
func (s *SaveSlot) validateOffsetChain() error {
    type check struct {
        name   string
        offset int
        minVal int
        maxVal int
    }

    checks := []check{
        {"MagicOffset", s.MagicOffset, MinMagicOffset, SlotSize},
        {"InventoryEnd", s.InventoryEnd, GaItemsStart, s.MagicOffset},
        {"PlayerDataOffset", s.PlayerDataOffset, s.InventoryEnd, SlotSize},
        {"FaceDataOffset", s.FaceDataOffset, s.PlayerDataOffset, SlotSize},
        {"StorageBoxOffset", s.StorageBoxOffset, s.FaceDataOffset, SlotSize},
    }

    for _, c := range checks {
        if c.offset < c.minVal || c.offset >= c.maxVal {
            return fmt.Errorf("offset %s = 0x%X out of expected range [0x%X, 0x%X)",
                c.name, c.offset, c.minVal, c.maxVal)
        }
    }

    // Monotonicity: offsets MUST be strictly increasing in this order
    if !(s.InventoryEnd <= s.MagicOffset &&
        s.MagicOffset < s.PlayerDataOffset &&
        s.PlayerDataOffset < s.FaceDataOffset &&
        s.FaceDataOffset < s.StorageBoxOffset) {
        return fmt.Errorf("offset chain order violated: "+
            "InventoryEnd=0x%X MagicOffset=0x%X PlayerData=0x%X FaceData=0x%X StorageBox=0x%X",
            s.InventoryEnd, s.MagicOffset, s.PlayerDataOffset,
            s.FaceDataOffset, s.StorageBoxOffset)
    }

    // EventFlagsOffset is optional (may be 0 if chain was unreachable)
    if s.EventFlagsOffset > 0 && s.EventFlagsOffset >= SlotSize {
        s.Warnings = append(s.Warnings,
            fmt.Sprintf("EventFlagsOffset 0x%X >= SlotSize, event flags disabled",
                s.EventFlagsOffset))
        s.EventFlagsOffset = 0
    }

    return nil
}
```

### Kiedy NIE odrzucać slotu

- `EventFlagsOffset = 0` — slot jest poprawny, ale Sites of Grace nie będą edytowalne.
  UI powinien wyświetlić warning i zablokować zakładkę World Progress dla tego slotu.
- Fallback MagicOffset — slot może być poprawny, ale z warningiem.

---

## 6. Etap E: Writer safety

### Cel
Zabezpieczyć `writeGaItem()`, `addToInventory()`, `generateUniqueHandle()` w `backend/core/writer.go`.

### 6.1. `generateUniqueHandle()` — limit iteracji

```go
func generateUniqueHandle(slot *SaveSlot, prefix uint32) (uint32, error) {
    h := prefix | 0x00010000
    for i := 0; i < MaxHandleAttempts; i++ {
        if _, ok := slot.GaMap[h]; !ok {
            return h, nil
        }
        h++
    }
    return 0, fmt.Errorf("failed to generate unique handle after %d attempts (prefix 0x%X)",
        MaxHandleAttempts, prefix)
}
```

**UWAGA:** Zmiana sygnatury z `uint32` na `(uint32, error)`.
Wymaga aktualizacji callsite w `AddItemsToSlot()` (linia ~92):

```go
// Stare:
handle = generateUniqueHandle(slot, prefix)
// Nowe:
handle, err = generateUniqueHandle(slot, prefix)
if err != nil {
    return err
}
```

### 6.2. `writeGaItem()` — dodatkowy bounds check

```go
func writeGaItem(slot *SaveSlot, handle, itemID uint32, size int) error {
    sa := NewSlotAccessor(slot.Data)

    // Check BOTH constraints: GaItems must not overflow into Magic section,
    // AND must not exceed the physical buffer.
    if err := sa.CheckBounds(slot.InventoryEnd, size, "writeGaItem"); err != nil {
        return err
    }
    if slot.InventoryEnd+size >= slot.MagicOffset {
        return fmt.Errorf("writeGaItem: no space in GaItems section "+
            "(InventoryEnd=0x%X + size=%d >= MagicOffset=0x%X)",
            slot.InventoryEnd, size, slot.MagicOffset)
    }

    // Write handle and itemID
    if err := sa.WriteU32(slot.InventoryEnd, handle); err != nil {
        return err
    }
    if err := sa.WriteU32(slot.InventoryEnd+4, itemID); err != nil {
        return err
    }
    // Zero remaining bytes (weapon=13 extra, armor=8 extra, others=0)
    for i := 8; i < size; i++ {
        if err := sa.WriteU8(slot.InventoryEnd+i, 0); err != nil {
            return err
        }
    }
    slot.InventoryEnd += size
    return nil
}
```

### 6.3. `addToInventory()` — bounds check przed zapisem

W funkcji `addToInventory()`, przed zapisem do `slot.Data`:

```go
// Before writing to slot.Data, validate offset
off := startOffset + emptyIdx*InvRecordLen
sa := NewSlotAccessor(slot.Data)
if err := sa.CheckBounds(off, InvRecordLen, "addToInventory"); err != nil {
    return err
}
```

### 6.4. `RemoveItemFromSlot()` — analogiczne bounds checks

Każdy zapis do `slot.Data` w `RemoveItemFromSlot()` powinien przejść przez `sa.CheckBounds()`
lub `sa.WriteU32()`.

---

## 7. Etap F: Warnings pipeline do UI

### Cel
Użytkownik widzi żółty banner gdy save załadowany z warningami.

### 7.1. Backend: `GetCharacter` zwraca warnings

Opcja A (prosta) — dodaj pole `Warnings` do `CharacterViewModel`:

```go
// backend/vm/character_vm.go
type CharacterViewModel struct {
    // ... istniejące pola ...
    Warnings []string `json:"warnings"`
}
```

W `MapParsedSlotToVM()`:
```go
vm.Warnings = slot.Warnings
```

### 7.2. Frontend: banner w App.tsx

```tsx
{character?.warnings?.length > 0 && (
    <div className="mx-4 mt-2 p-3 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
        <p className="text-[10px] font-bold text-yellow-600 uppercase tracking-widest">
            Save loaded with warnings
        </p>
        <ul className="mt-1 text-[9px] text-yellow-600/80 list-disc list-inside">
            {character.warnings.map((w, i) => <li key={i}>{w}</li>)}
        </ul>
    </div>
)}
```

### 7.3. Frontend: disable World Progress gdy EventFlagsOffset = 0

W `App.tsx` tab click handler:
```tsx
if (tab === 'world progress' && !character?.eventFlagsAvailable) {
    // Show toast or disable tab
}
```

Wymaga dodania pola `eventFlagsAvailable bool` do `CharacterViewModel`.

---

## 8. Etap G: Frontend hardening

### 8.1. Error Boundary (`frontend/src/components/ErrorBoundary.tsx`)

```tsx
import { Component, ReactNode } from 'react';

interface Props { children: ReactNode; }
interface State { hasError: boolean; error: string; }

export class ErrorBoundary extends Component<Props, State> {
    state: State = { hasError: false, error: '' };

    static getDerivedStateFromError(error: Error): State {
        return { hasError: true, error: error.message };
    }

    render() {
        if (this.state.hasError) {
            return (
                <div className="flex flex-col items-center justify-center h-full p-10">
                    <p className="text-red-500 font-bold">Something went wrong</p>
                    <p className="text-xs text-muted-foreground mt-2">{this.state.error}</p>
                    <button
                        onClick={() => this.setState({ hasError: false, error: '' })}
                        className="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded-md text-xs"
                    >
                        Try Again
                    </button>
                </div>
            );
        }
        return this.props.children;
    }
}
```

Wrap w `main.tsx`:
```tsx
<ErrorBoundary>
    <App />
</ErrorBoundary>
```

### 8.2. `useMemo` w InventoryTab i DatabaseTab

**InventoryTab.tsx** — owrapuj `mergedOwnedItems`:
```tsx
const mergedOwnedItems = useMemo(() => {
    // ... istniejąca logika merge ...
}, [inventory, storage]);

const filteredOwnedItems = useMemo(() => {
    return sortItems(mergedOwnedItems.filter(/* ... */));
}, [mergedOwnedItems, search, category, sortCol, sortDir, showFlaggedItems]);
```

**DatabaseTab.tsx** — owrapuj `filteredItems`:
```tsx
const filteredItems = useMemo(() => {
    return dbItems.filter(/* ... */).sort(/* ... */);
}, [dbItems, search, sortCol, sortDir, showFlaggedItems]);
```

### 8.3. Fix `window.go.main.App.SaveCharacter` bypass

W `InventoryTab.tsx` linia ~109, zamień:
```tsx
// STARE (bypass):
await window.go.main.App.SaveCharacter(charIndex, char);

// NOWE (importowany binding):
import { SaveCharacter } from '../wailsjs/go/main/App';
// ...
await SaveCharacter(charIndex, char);
```

**UWAGA:** Po tej zmianie konieczne `wails generate module` jeśli `SaveCharacter`
nie istnieje jeszcze w auto-generowanych bindingach.

### 8.4. WorldProgressTab — nie połykaj błędów

```tsx
// STARE:
.catch(() => setLoading(false));

// NOWE:
.catch(err => {
    console.error("Failed to load graces:", err);
    setLoading(false);
    // Opcjonalnie: setError(String(err));
});
```

---

## 9. Etap H: Testy jednostkowe

### Nowy plik: `backend/core/slot_access_test.go`

```go
package core

import "testing"

func TestSlotAccessorReadU32OutOfBounds(t *testing.T) {
    sa := NewSlotAccessor(make([]byte, 10))
    _, err := sa.ReadU32(8) // needs 4 bytes at offset 8, but buffer is only 10
    if err == nil {
        t.Fatal("expected error for out-of-bounds read")
    }
}

func TestSlotAccessorReadU32Negative(t *testing.T) {
    sa := NewSlotAccessor(make([]byte, 100))
    _, err := sa.ReadU32(-1)
    if err == nil {
        t.Fatal("expected error for negative offset")
    }
}

func TestSlotAccessorReadDynamicSizeClamp(t *testing.T) {
    data := make([]byte, 100)
    binary.LittleEndian.PutUint32(data[0:], 99999) // absurd value
    sa := NewSlotAccessor(data)
    size, err := sa.ReadDynamicSize(0, 256, "test")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if size != 0 {
        t.Fatalf("expected clamped to 0, got %d", size)
    }
    if len(sa.Warnings) != 1 {
        t.Fatalf("expected 1 warning, got %d", len(sa.Warnings))
    }
}
```

### Nowy plik: `backend/core/offset_validation_test.go`

```go
package core

import "testing"

func TestValidateOffsetChain_Valid(t *testing.T) {
    s := &SaveSlot{
        Data:             make([]byte, SlotSize),
        MagicOffset:      0x15852,
        InventoryEnd:     0x10000,
        PlayerDataOffset: 0x15852 + 0x1B0, // > MagicOffset
        FaceDataOffset:   0x20000,
        StorageBoxOffset: 0x26010,
    }
    if err := s.validateOffsetChain(); err != nil {
        t.Fatalf("valid chain rejected: %v", err)
    }
}

func TestValidateOffsetChain_NonMonotonic(t *testing.T) {
    s := &SaveSlot{
        Data:             make([]byte, SlotSize),
        MagicOffset:      0x15852,
        InventoryEnd:     0x10000,
        PlayerDataOffset: 0x10000, // == InventoryEnd, not > MagicOffset → fail
        FaceDataOffset:   0x20000,
        StorageBoxOffset: 0x26010,
    }
    if err := s.validateOffsetChain(); err == nil {
        t.Fatal("non-monotonic chain should be rejected")
    }
}

func TestValidateOffsetChain_MagicTooSmall(t *testing.T) {
    s := &SaveSlot{
        Data:        make([]byte, SlotSize),
        MagicOffset: 100, // < MinMagicOffset
    }
    if err := s.validateOffsetChain(); err == nil {
        t.Fatal("small MagicOffset should be rejected")
    }
}
```

### Rozszerzenie istniejących round-trip testów

W `tests/roundtrip_test.go` po załadowaniu save'a dodaj:
```go
// Verify no warnings on known-good saves
for i, slot := range save.Slots {
    if len(slot.Warnings) > 0 {
        t.Errorf("Slot %d has unexpected warnings: %v", i, slot.Warnings)
    }
}
```

---

## 10. Checklist walidacji po każdym etapie

Po **KAŻDYM** etapie (A, B, C, D, E, F, G, H) wykonaj:

```bash
# 1. Kompilacja backendu
go build ./backend/...

# 2. Kompilacja całej aplikacji (sprawdza bindingsy)
go build ./...

# 3. Testy jednostkowe backendu
go test -v ./backend/core/...

# 4. Round-trip testy (PS4, PC, konwersje)
go test -v ./tests/roundtrip_test.go

# 5. TypeScript typecheck (po zmianach frontendu)
cd frontend && npx tsc --noEmit

# 6. Frontend lint (po zmianach frontendu)
cd frontend && npm run lint

# 7. Full build
make build
```

**ZASADA:** Jeśli którykolwiek krok fail — napraw PRZED przejściem do następnego etapu.

---

## 11. Mapa zmian — pliki do modyfikacji

| Etap | Plik | Operacja | Opis |
|---|---|---|---|
| A | `backend/core/offset_defs.go` | **NOWY** | Named constants |
| B | `backend/core/slot_access.go` | **NOWY** | SlotAccessor |
| C | `backend/core/structures.go` | MODIFY | Error propagation, use SlotAccessor + constants |
| D | `backend/core/structures.go` | MODIFY | Dodaj `validateOffsetChain()` |
| E | `backend/core/writer.go` | MODIFY | Bounds checks, generateUniqueHandle error |
| E | `app.go` | MODIFY | Handle nowy error z generateUniqueHandle (jeśli callsite zmieniony) |
| F | `backend/vm/character_vm.go` | MODIFY | Dodaj `Warnings` do CharacterViewModel |
| F | `frontend/src/components/App.tsx` | MODIFY | Warning banner |
| G | `frontend/src/components/ErrorBoundary.tsx` | **NOWY** | Error boundary |
| G | `frontend/src/main.tsx` | MODIFY | Wrap App w ErrorBoundary |
| G | `frontend/src/components/InventoryTab.tsx` | MODIFY | useMemo, fix window.go bypass |
| G | `frontend/src/components/DatabaseTab.tsx` | MODIFY | useMemo |
| G | `frontend/src/components/WorldProgressTab.tsx` | MODIFY | Error handling |
| H | `backend/core/slot_access_test.go` | **NOWY** | Unit testy SlotAccessor |
| H | `backend/core/offset_validation_test.go` | **NOWY** | Unit testy walidacji |
| H | `tests/roundtrip_test.go` | MODIFY | Dodaj warnings check |

| I | `backend/core/save_manager.go` | MODIFY | Min file size validation, ReadBytes error propagation, cross-platform atomic write |
| J | `backend/db/db.go` | MODIFY | Event flags bounds-check z error, global item index O(1) |
| J | `app.go` | MODIFY | Handle nowe error returns z GetEventFlag/SetEventFlag |
| K | `frontend/src/components/InventoryTab.tsx` | MODIFY | react-virtual, toast, shared UI components |
| K | `frontend/src/components/DatabaseTab.tsx` | MODIFY | react-virtual, toast, shared UI components |
| K | `frontend/src/components/ui/Card.tsx` | **NOWY** | Shared Card component |
| K | `frontend/src/components/ui/SectionHeader.tsx` | **NOWY** | Shared section header |
| K | `frontend/src/components/ui/ActionButton.tsx` | **NOWY** | Shared button component |
| K | `frontend/src/main.tsx` | MODIFY | Toast provider (react-hot-toast) |
| K | `frontend/package.json` | MODIFY | Dodaj @tanstack/react-virtual, react-hot-toast |

### Pliki które NIE powinny być modyfikowane

- `backend/core/reader.go` — niezależny streaming reader, nie wymaga zmian
- `backend/core/crypto.go` — nie dotyczy offsetów
- `backend/core/backup.go` — nie dotyczy offsetów
- `frontend/wailsjs/` — auto-generowane, nie edytuj

---

## 12. Etap I: SaveManager hardening

### Cel
Zabezpieczyć `save_manager.go` przed trzema scenariuszami: crash na za małym pliku,
cicha utrata danych przy atomic write na Windows, ignorowane errory z ReadBytes.

### 12.1. Walidacja minimalnego rozmiaru pliku w `LoadSave()`

```go
// backend/core/save_manager.go — na początku LoadSave(), po os.Open

const MinSaveFileSize = 10 * SlotSize // 10 slots × 0x280000 = ~25 MB minimum

func LoadSave(path string) (*SaveFile, error) {
    info, err := os.Stat(path)
    if err != nil {
        return nil, fmt.Errorf("cannot stat save file: %w", err)
    }
    if info.Size() < MinSaveFileSize {
        return nil, fmt.Errorf("file too small (%d bytes, minimum %d) — not a valid save",
            info.Size(), MinSaveFileSize)
    }
    // ... reszta LoadSave ...
}
```

### 12.2. Error propagation z ReadBytes

Znajdź WSZYSTKIE wywołania `r.ReadBytes()` z ignorowanym errorem (`_`) i zamień na propagację:

```go
// STARE:
padding, _ := r.ReadBytes(0x60000)

// NOWE:
padding, err := r.ReadBytes(0x60000)
if err != nil {
    return nil, fmt.Errorf("failed to read padding at offset 0x%X: %w", r.Offset(), err)
}
```

**UWAGA:** Przejrzyj cały `save_manager.go` — mogą być inne miejsca z `_`.

### 12.3. Atomic write cross-platform

`os.Rename()` na Windows NIE nadpisuje istniejącego pliku (zwraca error).
Dodatkowo, przy błędzie rename obecny kod kasuje `.tmp`, tracąc dane.

```go
func (sf *SaveFile) SaveFile(path string) error {
    tmpPath := path + ".tmp"

    // ... write to tmpPath ...

    // Cross-platform atomic rename
    if runtime.GOOS == "windows" {
        // Windows: remove target first, then rename
        if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
            // DON'T delete tmp — it contains the user's data!
            return fmt.Errorf("cannot remove old file for atomic write: %w (new data preserved in %s)", err, tmpPath)
        }
    }
    if err := os.Rename(tmpPath, path); err != nil {
        // DON'T delete tmp — it contains the user's data!
        return fmt.Errorf("rename failed: %w (new data preserved in %s)", err, tmpPath)
    }
    return nil
}
```

**KLUCZOWE:** NIGDY nie kasuj `.tmp` przy błędzie — to jedyna kopia danych użytkownika.
Zamiast tego zwróć error z informacją gdzie jest `.tmp`.

---

## 13. Etap J: Database & event flags hardening

### Cel
Zabezpieczyć `db.go` przed cichymi błędami i poprawić wydajność lookupów.

### 13.1. Event flags — bounds checking z error

```go
// backend/db/db.go

// GetEventFlag returns the flag value, or false + error if offset is out of bounds.
func GetEventFlag(flags []byte, flagID uint32) (bool, error) {
    byteIdx := flagID / 8
    bitIdx := flagID % 8
    if int(byteIdx) >= len(flags) {
        return false, fmt.Errorf("event flag %d (byte %d) out of bounds (flags len %d)",
            flagID, byteIdx, len(flags))
    }
    return (flags[byteIdx] & (1 << bitIdx)) != 0, nil
}

// SetEventFlag sets or clears a flag. Returns error if offset is out of bounds.
func SetEventFlag(flags []byte, flagID uint32, value bool) error {
    byteIdx := flagID / 8
    bitIdx := flagID % 8
    if int(byteIdx) >= len(flags) {
        return fmt.Errorf("event flag %d (byte %d) out of bounds (flags len %d)",
            flagID, byteIdx, len(flags))
    }
    if value {
        flags[byteIdx] |= 1 << bitIdx
    } else {
        flags[byteIdx] &^= 1 << bitIdx
    }
    return nil
}
```

**UWAGA:** Zmiana sygnatur wymaga aktualizacji callsites:
- `app.go: GetGraces()` — obsłuż error z `GetEventFlag`
- `app.go: SetGraceVisited()` — propaguj error z `SetEventFlag`

### 13.2. Global item index — O(1) lookup zamiast O(18×n)

```go
// backend/db/db.go

var globalItemIndex map[uint32]ItemEntry

func init() {
    // Build global index from all category maps at startup
    globalItemIndex = make(map[uint32]ItemEntry, 4000)
    allMaps := []map[uint32]ItemEntry{
        data.Weapons, data.RangedAndCatalysts, data.ArrowsAndBolts,
        data.Shields, data.Helms, data.Chest, data.Arms, data.Legs,
        data.Talismans, data.Aows, data.Ashes, data.Sorceries,
        data.Incantations, data.CraftingMaterials, data.BolsteringMaterials,
        data.Tools, data.KeyItems,
    }
    for _, m := range allMaps {
        for id, entry := range m {
            globalItemIndex[id] = entry
        }
    }
}

// GetItemData returns item data in O(1) via the global index.
func GetItemData(id uint32) (ItemEntry, bool) {
    entry, ok := globalItemIndex[id]
    return entry, ok
}
```

Zastąp wywołania starej `GetItemData()` (linear search) nową wersją.
`GetItemDataFuzzy()` może wewnętrznie korzystać z `globalItemIndex` — najpierw szuka exact match,
potem base ID (id z zamaskowanymi bitami upgrade).

---

## 14. Etap K: Frontend — performance & UI consistency

### Cel
Virtualizacja dużych tabel, shared UI components, ujednolicony system notyfikacji.

### 14.1. Table virtualization z `@tanstack/react-virtual`

```bash
cd frontend && npm install @tanstack/react-virtual
```

Zastosuj w `InventoryTab.tsx` i `DatabaseTab.tsx`:

```tsx
import { useVirtualizer } from '@tanstack/react-virtual';

// W komponencie:
const parentRef = useRef<HTMLDivElement>(null);
const rowVirtualizer = useVirtualizer({
    count: filteredItems.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 40, // row height in px
    overscan: 20,
});

// W JSX — zamiast mapowania WSZYSTKICH items:
<div ref={parentRef} className="overflow-auto" style={{ height: '600px' }}>
    <div style={{ height: `${rowVirtualizer.getTotalSize()}px`, position: 'relative' }}>
        {rowVirtualizer.getVirtualItems().map(virtualRow => {
            const item = filteredItems[virtualRow.index];
            return (
                <div
                    key={item.handle || item.id}
                    style={{
                        position: 'absolute',
                        top: 0,
                        transform: `translateY(${virtualRow.start}px)`,
                        width: '100%',
                        height: `${virtualRow.size}px`,
                    }}
                >
                    {/* ... istniejący wiersz tabeli ... */}
                </div>
            );
        })}
    </div>
</div>
```

**Efekt:** Zamiast 1000+ DOM nodes → ~40-60 widocznych + 20 overscan. Rendering 10-50x szybszy.

### 14.2. Toast/notification system

```bash
cd frontend && npm install react-hot-toast
```

```tsx
// frontend/src/main.tsx
import { Toaster } from 'react-hot-toast';

<ErrorBoundary>
    <App />
    <Toaster position="bottom-right" toastOptions={{
        style: { background: '#1a1a2e', color: '#e0e0e0', border: '1px solid #333' },
        duration: 4000,
    }} />
</ErrorBoundary>
```

Zastąp we WSZYSTKICH komponentach:
```tsx
// STARE:
alert("Error: " + err);
// lub:
.catch(() => setLoading(false));

// NOWE:
import toast from 'react-hot-toast';
toast.error("Failed to load: " + String(err));
```

### 14.3. Shared UI components

Utwórz `frontend/src/components/ui/` z trzema komponentami:

**`Card.tsx`:**
```tsx
interface CardProps {
    children: React.ReactNode;
    className?: string;
}
export function Card({ children, className = '' }: CardProps) {
    return (
        <div className={`bg-[#1a1a2e] border border-[#2a2a4a] rounded-xl p-6 ${className}`}>
            {children}
        </div>
    );
}
```

**`SectionHeader.tsx`:**
```tsx
interface SectionHeaderProps {
    title: string;
    accentColor?: string;
}
export function SectionHeader({ title, accentColor = 'bg-blue-500' }: SectionHeaderProps) {
    return (
        <div className="flex items-center gap-2 mb-4">
            <div className={`w-1 h-5 ${accentColor} rounded-full`} />
            <h3 className="text-sm font-semibold text-white uppercase tracking-wider">{title}</h3>
        </div>
    );
}
```

**`ActionButton.tsx`:**
```tsx
interface ActionButtonProps {
    children: React.ReactNode;
    onClick: () => void;
    variant?: 'primary' | 'danger' | 'ghost';
    disabled?: boolean;
    className?: string;
}
export function ActionButton({ children, onClick, variant = 'primary', disabled, className = '' }: ActionButtonProps) {
    const base = 'px-4 py-2 rounded-lg text-xs font-medium transition-all duration-200 disabled:opacity-40';
    const variants = {
        primary: 'bg-blue-600 hover:bg-blue-500 text-white',
        danger: 'bg-red-600/20 hover:bg-red-600/40 text-red-400 border border-red-500/30',
        ghost: 'bg-white/5 hover:bg-white/10 text-gray-300',
    };
    return (
        <button onClick={onClick} disabled={disabled} className={`${base} ${variants[variant]} ${className}`}>
            {children}
        </button>
    );
}
```

Następnie zamień hardcoded style w istniejących tabs na te komponenty.
**NIE rób tego hurtowo** — zamieniaj po jednym tabie, sprawdzając wizualnie po każdym.

---

## 15. Propozycje architektoniczne (future — Phase 21+)

Poniższe propozycje wykraczają poza scope Phase 20. Zapisane tu jako referencja dla przyszłych faz.

### 15.1. Write-ahead validation
Przed każdym zapisem do pliku (`SaveFile()`), wywołaj `validateSlotIntegrity()` sprawdzający
że cały łańcuch offsetów i dane inventory są spójne. Zapobiega zapisaniu uszkodzonego save'a.

### 15.2. Undo/redo system
Zachowaj kopię `slot.Data` przed edycją. Pozwala na "Revert" bez ponownego ładowania pliku.
Implementacja: deep copy `[]byte` + stack operacji w `App` struct.

### 15.3. Save file diffing
Przed zapisem porównaj wynikowy plik z oryginałem i pokaż userowi co się zmieniło
(np. "Changed: Level 50→99, Added: 3 items, Modified: 2 quantities").
Buduje zaufanie że save jest poprawny.

### 15.4. `updateItemsAndSync()` transactionality
`character_vm.go:updateItemsAndSync()` — jeśli zapis qty powiedzie się dla 3/5 items
a 4th fail, stan in-memory i binarny są niespójne. Brak rollbacku.
Propozycja: waliduj wszystkie offsety przed startem zapisu, lub operuj na kopii `slot.Data`.
Oznaczone jako `// TODO(phase21): migrate to SlotAccessor`.

---

## 16. Anty-wzorce — czego NIE robić

### ❌ NIE zmieniaj formatu binarnego save file
Ten refaktor dotyczy WYŁĄCZNIE logiki odczytu/zapisu. Bajty zapisane do pliku
muszą być identyczne jak przed refaktorem. Potwierdź round-trip testami.

### ❌ NIE usuwaj silent fallbacków dla PS4 bez zastąpienia
PS4 saves NAPRAWDĘ mają garbage w `projSize` i `unlockedRegSz`.
Zamiast usuwać fallback → zamień na `ReadDynamicSize()` z clamp + warning.
Fallback to poprawne zachowanie, brak raportowania — to bug.

### ❌ NIE dodawaj nowych zależności (bibliotek)
SlotAccessor i offset_defs używają wyłącznie stdlib. Nie dodawaj loggerów,
assertion libraries, itp.

### ❌ NIE rób Etapów C+D+E w jednym commicie
Każdy etap = osobny commit. Jeśli C zepsuje testy, łatwo znaleźć przyczynę.

### ❌ NIE zmieniaj kolejności operacji w Read()
`FindPattern → mapStats → scanGaItems → calculateDynamicOffsets → validateOffsetChain → mapInventory`
Ta kolejność jest krytyczna — każdy krok zależy od poprzedniego.

### ❌ NIE ignoruj errorów z SlotAccessor w Write()
W `Write()` errory oznaczają bug w logice (offsety były walidowane w `Read()`).
Użyj `_ = sa.WriteU32(...)` TYLKO jeśli jesteś 100% pewny że offset jest poprawny
(bo przeszedł walidację w Read). W razie wątpliwości — propaguj error.

### ❌ NIE dodawaj `SlotAccessor` do `SaveSlot` struct
`SlotAccessor` to tymczasowy helper tworzony na czas operacji.
NIE jest stanem slotu. Twórz go lokalnie: `sa := NewSlotAccessor(s.Data)`.

### ❌ NIE zmieniaj `character_vm.go` `updateItemsAndSync()` w tym refaktorze
Ta funkcja też ma hardcoded offsety, ale jest złożona i wymaga osobnego refaktoru.
Oznacz ją `// TODO(phase21): migrate to SlotAccessor` i zostaw na następną fazę.

---

## Przykładowa kolejność commitów

```
feat(core): add offset_defs.go with named constants (Etap A)
feat(core): add SlotAccessor for bounds-checked access (Etap B)
refactor(core): mapStats returns error, uses SlotAccessor (Etap C.1)
refactor(core): calculateDynamicOffsets returns error (Etap C.2)
refactor(core): Read() propagates errors from mapStats/dynOffsets (Etap C.3)
refactor(core): Write() uses SlotAccessor + named constants (Etap C.4)
feat(core): add validateOffsetChain cross-validation (Etap D)
fix(core): bounds checks in writeGaItem/addToInventory (Etap E.1)
fix(core): generateUniqueHandle returns error with max attempts (Etap E.2)
feat(vm): add Warnings field to CharacterViewModel (Etap F)
feat(ui): add ErrorBoundary component (Etap G.1)
perf(ui): useMemo for filtered item lists (Etap G.2)
fix(ui): import SaveCharacter from wailsjs instead of window.go (Etap G.3)
test(core): unit tests for SlotAccessor and offset validation (Etap H)
fix(core): validate min file size and propagate ReadBytes errors (Etap I.1)
fix(core): cross-platform atomic write, preserve .tmp on failure (Etap I.2)
fix(db): event flags bounds-check with error return (Etap J.1)
perf(db): global item index for O(1) lookups (Etap J.2)
perf(ui): table virtualization with @tanstack/react-virtual (Etap K.1)
refactor(ui): unified toast notifications, remove alert() (Etap K.2)
refactor(ui): shared Card, SectionHeader, ActionButton components (Etap K.3)
```
