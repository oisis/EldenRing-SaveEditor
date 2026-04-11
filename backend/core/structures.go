package core

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// MagicPattern matches the 192-byte pattern used in the Python editor for reliability.
// First block: 0x00 + 0xFFFFFFFF + 12 zeros (17 bytes)
// Subsequent blocks: 0xFFFFFFFF + 12 zeros (16 bytes each)
var MagicPattern = []byte{
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

const (
	ItemTypeWeapon    = 0x80000000
	ItemTypeArmor     = 0x90000000
	ItemTypeAccessory = 0xA0000000
	ItemTypeItem      = 0xB0000000
	ItemTypeAow       = 0xC0000000
)

type GaItem struct {
	Handle uint32
	ItemID uint32
}

type InventoryItem struct {
	GaItemHandle uint32
	Quantity     uint32
	Index        uint32
}

type EquipInventoryData struct {
	CommonItems           []InventoryItem
	KeyItems              []InventoryItem
	NextEquipIndex        uint32
	NextAcquisitionSortId uint32
	nextEquipIndexOff     int // absolute byte offset in slot.Data (for write-back)
	nextAcqSortIdOff      int // absolute byte offset in slot.Data (for write-back)
}

// Clone returns a deep copy of EquipInventoryData, including unexported offset fields.
func (e *EquipInventoryData) Clone() EquipInventoryData {
	c := EquipInventoryData{
		NextEquipIndex:        e.NextEquipIndex,
		NextAcquisitionSortId: e.NextAcquisitionSortId,
		nextEquipIndexOff:     e.nextEquipIndexOff,
		nextAcqSortIdOff:      e.nextAcqSortIdOff,
	}
	if e.CommonItems != nil {
		c.CommonItems = make([]InventoryItem, len(e.CommonItems))
		copy(c.CommonItems, e.CommonItems)
	}
	if e.KeyItems != nil {
		c.KeyItems = make([]InventoryItem, len(e.KeyItems))
		copy(c.KeyItems, e.KeyItems)
	}
	return c
}

func (e *EquipInventoryData) Read(r *Reader, commonCount, keyCount int) error {
	e.CommonItems = make([]InventoryItem, commonCount)
	var err error
	for i := 0; i < commonCount; i++ {
		if e.CommonItems[i].GaItemHandle, err = r.ReadU32(); err != nil {
			return fmt.Errorf("common[%d].handle: %w", i, err)
		}
		if e.CommonItems[i].Quantity, err = r.ReadU32(); err != nil {
			return fmt.Errorf("common[%d].quantity: %w", i, err)
		}
		if e.CommonItems[i].Index, err = r.ReadU32(); err != nil {
			return fmt.Errorf("common[%d].index: %w", i, err)
		}
	}
	if _, err = r.ReadU32(); err != nil { // skip key_count header
		return fmt.Errorf("key_count header: %w", err)
	}
	e.KeyItems = make([]InventoryItem, keyCount)
	for i := 0; i < keyCount; i++ {
		if e.KeyItems[i].GaItemHandle, err = r.ReadU32(); err != nil {
			return fmt.Errorf("key[%d].handle: %w", i, err)
		}
		if e.KeyItems[i].Quantity, err = r.ReadU32(); err != nil {
			return fmt.Errorf("key[%d].quantity: %w", i, err)
		}
		if e.KeyItems[i].Index, err = r.ReadU32(); err != nil {
			return fmt.Errorf("key[%d].index: %w", i, err)
		}
	}
	// Trailing counters — record offsets for write-back
	e.nextEquipIndexOff = r.Pos()
	if e.NextEquipIndex, err = r.ReadU32(); err != nil {
		return fmt.Errorf("NextEquipIndex: %w", err)
	}
	e.nextAcqSortIdOff = r.Pos()
	if e.NextAcquisitionSortId, err = r.ReadU32(); err != nil {
		return fmt.Errorf("NextAcquisitionSortId: %w", err)
	}
	return nil
}

type PlayerGameData struct {
	Level               uint32
	Vigor               uint32
	Mind                uint32
	Endurance           uint32
	Strength            uint32
	Dexterity           uint32
	Intelligence        uint32
	Faith               uint32
	Arcane              uint32
	Souls               uint32
	CharacterName       [16]uint16
	Gender              uint8
	Class               uint8
	ScadutreeBlessing   uint8
	ShadowRealmBlessing uint8
}

type SaveSlot struct {
	Data      []byte
	Version   uint32 // slot format version (offset 0x00); 0 = empty slot
	Player    PlayerGameData
	GaMap     map[uint32]uint32
	Inventory EquipInventoryData
	Storage   EquipInventoryData
	SteamID   uint64
	Warnings  []string // non-fatal issues detected during parsing

	MagicOffset      int
	InventoryEnd     int
	EventFlagsOffset int

	// Dynamic offsets from Python logic
	PlayerDataOffset   int
	FaceDataOffset     int
	StorageBoxOffset   int
	IngameTimerOffset  int
	GaItemDataOffset   int // start of GaItemData section (distinct_acquired_items_count header)
}

func (s *SaveSlot) Read(r *Reader, platform string) error {
	var err error
	s.Data, err = r.ReadBytes(SlotSize)
	if err != nil {
		return err
	}

	// 0. Read slot version (offset 0x00). Version 0 = empty/unused slot.
	s.Version = binary.LittleEndian.Uint32(s.Data[0:4])

	// 1. Find primary anchor
	s.MagicOffset = NewReader(s.Data).FindPattern(MagicPattern)
	if s.MagicOffset == -1 {
		s.MagicOffset = FallbackMagicBase
		s.Warnings = append(s.Warnings,
			fmt.Sprintf("MagicPattern not found, using fallback offset 0x%X", FallbackMagicBase))
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

	// 5. Cross-validate offset chain
	if err := s.validateOffsetChain(); err != nil {
		return fmt.Errorf("offset validation: %w", err)
	}

	// 6. Map inventory
	if err := s.mapInventory(); err != nil {
		return fmt.Errorf("mapInventory: %w", err)
	}

	// NOTE: Per-slot SteamID is at a dynamic offset within the sequential parsing chain
	// (after BaseVersion, before PS5Activity), NOT at the fixed SlotSize-8 address.
	// SlotSize-8 falls inside the PlayerGameDataHash region. The authoritative SteamID
	// is read from UserData10 by the save_manager and propagated to slots from there.
	return nil
}

func (s *SaveSlot) calculateDynamicOffsets() error {
	sa := NewSlotAccessor(s.Data)

	s.PlayerDataOffset = s.MagicOffset

	spEffect := s.PlayerDataOffset + DynSpEffect
	equipedItemIndex := spEffect + DynEquipedItemIndex
	activeEquipedItems := equipedItemIndex + DynActiveEquipedItems
	equipedItemsID := activeEquipedItems + DynEquipedItemsID
	activeEquipedItemsGa := equipedItemsID + DynActiveEquipedItemsGa
	inventoryHeld := activeEquipedItemsGa + DynInventoryHeld
	equipedSpells := inventoryHeld + DynEquipedSpells
	equipedItems := equipedSpells + DynEquipedItems
	equipedGestures := equipedItems + DynEquipedGestures

	// Dynamic field #1: acquired_projectiles header.
	// The u32 at equipedGestures is the byte-size of projectile data that follows.
	// Reference editors (ER-Save-Editor, er-save-manager) skip this section by reading
	// the 4-byte header and advancing past it. The actual projectile data size is embedded
	// in the header value but we only need to skip the 4-byte header itself — the projectile
	// data is already accounted for in the fixed offsets that follow.
	if err := sa.CheckBounds(equipedGestures, 4, "projHeader"); err != nil {
		return err
	}
	equipedProjectile := equipedGestures + 4

	equipedArmaments := equipedProjectile + DynEquipedArmaments
	equipePhysics := equipedArmaments + DynEquipePhysics
	s.FaceDataOffset = equipePhysics + DynFaceData
	s.StorageBoxOffset = s.FaceDataOffset + DynStorageBox

	// EventFlags offset chain
	gesturesOff := s.StorageBoxOffset + DynStorageToGestures
	if err := sa.CheckBounds(gesturesOff, 4, "gesturesOff"); err != nil {
		s.Warnings = append(s.Warnings, "EventFlags chain unreachable: "+err.Error())
		s.Warnings = append(s.Warnings, sa.Warnings...)
		return nil // non-fatal — event flags are optional for basic editing
	}

	// Dynamic field #2: unlocked_regions header.
	// Same pattern as projSize — the u32 is a byte-size/count value but we only need
	// to skip the 4-byte header. The region data is accounted for in fixed offsets.
	if err := sa.CheckBounds(gesturesOff, 4, "unlockedRegHeader"); err != nil {
		return err
	}
	unlockedRegion := gesturesOff + 4

	horse := unlockedRegion + DynHorse
	bloodStain := horse + DynBloodStain
	menuProfile := bloodStain + DynMenuProfile
	gaItemsOther := menuProfile + DynGaItemsOther
	s.GaItemDataOffset = gaItemsOther // GaItemData (ga_item_data) starts here — see Rust save_slot.rs read sequence
	tutorialData := gaItemsOther + DynTutorialData
	s.IngameTimerOffset = tutorialData + DynIngameTimer
	s.EventFlagsOffset = s.IngameTimerOffset + DynEventFlags

	// Collect SlotAccessor warnings
	s.Warnings = append(s.Warnings, sa.Warnings...)
	return nil
}

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
		{"InventoryEnd", s.InventoryEnd, GaItemsStart, s.MagicOffset - DynPlayerData + 1},
		{"PlayerDataOffset", s.PlayerDataOffset, s.MagicOffset, SlotSize},
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
		s.MagicOffset <= s.PlayerDataOffset &&
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

// ValidateSlotIntegrity performs write-ahead validation on a slot before saving.
// It re-checks the offset chain, inventory bounds, data length and stat sanity
// to prevent writing a corrupted save file.
func ValidateSlotIntegrity(slot *SaveSlot) error {
	// 1. Data length must equal SlotSize
	if len(slot.Data) != SlotSize {
		return fmt.Errorf("slot data length %d (0x%X) != expected SlotSize %d (0x%X)",
			len(slot.Data), len(slot.Data), SlotSize, SlotSize)
	}

	// 2. Offset chain re-validation
	if err := slot.validateOffsetChain(); err != nil {
		return fmt.Errorf("offset chain invalid: %w", err)
	}

	// 3. Inventory bounds: invStart and storageStart must be within slot.Data
	invStart := slot.MagicOffset + InvStartFromMagic
	if invStart < 0 || invStart >= SlotSize {
		return fmt.Errorf("inventory start offset 0x%X out of bounds [0, 0x%X)",
			invStart, SlotSize)
	}
	storageStart := slot.StorageBoxOffset + StorageHeaderSkip
	if storageStart < 0 || storageStart >= SlotSize {
		return fmt.Errorf("storage start offset 0x%X out of bounds [0, 0x%X)",
			storageStart, SlotSize)
	}

	// 4. Stat sanity: Level must be > 0, attributes 1–99
	if slot.Player.Level == 0 || slot.Player.Level > 713 {
		return fmt.Errorf("Level %d out of valid range [1, 713]", slot.Player.Level)
	}
	attrs := []struct {
		name string
		val  uint32
	}{
		{"Vigor", slot.Player.Vigor},
		{"Mind", slot.Player.Mind},
		{"Endurance", slot.Player.Endurance},
		{"Strength", slot.Player.Strength},
		{"Dexterity", slot.Player.Dexterity},
		{"Intelligence", slot.Player.Intelligence},
		{"Faith", slot.Player.Faith},
		{"Arcane", slot.Player.Arcane},
	}
	for _, a := range attrs {
		if a.val < 1 || a.val > 99 {
			return fmt.Errorf("%s = %d out of valid range [1, 99]", a.name, a.val)
		}
	}

	return nil
}

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

func (s *SaveSlot) scanGaItems(start int) {
	s.GaMap = make(map[uint32]uint32)
	curr := start

	gaLimit := s.MagicOffset - DynPlayerData // writable GaItems region ends 0x1B0 before Magic
	if gaLimit < start {
		gaLimit = start
	}

	// Determine expected GaItem count from slot version.
	// Reference: ER-Save-Editor reads exactly 5118 (version ≤ 81) or 5120 (version > 81).
	maxEntries := GaItemCountNew
	if s.Version > 0 && s.Version <= GaItemVersionBreak {
		maxEntries = GaItemCountOld
	}

	lastEnd := start
	entriesRead := 0
	for curr+GaRecordItem <= gaLimit && entriesRead < maxEntries {
		handle := binary.LittleEndian.Uint32(s.Data[curr:])
		itemID := binary.LittleEndian.Uint32(s.Data[curr+4:])

		if handle != GaHandleEmpty && handle != GaHandleInvalid {
			// Validate type prefix — only known types are valid GaItem records.
			// An unknown prefix (e.g. 0xFFFF0000 from scanner misalignment) must
			// be treated as a stop condition, not a valid item.
			typeBits := handle & GaHandleTypeMask
			switch typeBits {
			case ItemTypeWeapon:
				s.GaMap[handle] = itemID
				curr += GaRecordWeapon
				lastEnd = curr
			case ItemTypeArmor:
				s.GaMap[handle] = itemID
				curr += GaRecordArmor
				lastEnd = curr
			case ItemTypeAccessory, ItemTypeItem, ItemTypeAow:
				s.GaMap[handle] = itemID
				curr += GaRecordItem
				lastEnd = curr
			default:
				// Unknown type prefix — stop scanning (corrupted/misaligned region).
				curr = gaLimit
			}
		} else {
			curr += GaRecordItem
		}
		entriesRead++
	}
	s.InventoryEnd = lastEnd
}

func (e *EquipInventoryData) ReadStorage(r *Reader, count int) error {
	e.CommonItems = []InventoryItem{}
	for i := 0; i < count; i++ {
		handle, err := r.ReadU32()
		if err != nil {
			return fmt.Errorf("storage[%d].handle: %w", i, err)
		}
		quantity, err := r.ReadU32()
		if err != nil {
			return fmt.Errorf("storage[%d].quantity: %w", i, err)
		}
		index, err := r.ReadU32()
		if err != nil {
			return fmt.Errorf("storage[%d].index: %w", i, err)
		}

		// Skip empty/invalid entries but continue reading — storage can have sparse gaps
		// after item removal. Breaking on first empty would lose items after the gap.
		if handle == GaHandleEmpty || handle == GaHandleInvalid {
			continue
		}

		e.CommonItems = append(e.CommonItems, InventoryItem{
			GaItemHandle: handle,
			Quantity:     quantity,
			Index:        index,
		})
	}
	e.KeyItems = []InventoryItem{}
	return nil
}

func (s *SaveSlot) mapInventory() error {
	// Main Inventory
	invStart := s.MagicOffset + InvStartFromMagic
	if invStart+InvSafetyMargin < len(s.Data) {
		ir := NewReader(s.Data)
		ir.Seek(int64(invStart), 0)
		if err := s.Inventory.Read(ir, CommonItemCount, KeyItemCount); err != nil {
			return fmt.Errorf("inventory read: %w", err)
		}
	}

	// Storage Box
	storageStart := s.StorageBoxOffset + StorageHeaderSkip
	if storageStart+StorageSafetyMarg < len(s.Data) {
		sr := NewReader(s.Data)
		sr.Seek(int64(storageStart), 0)
		if err := s.Storage.ReadStorage(sr, StorageItemCount); err != nil {
			return fmt.Errorf("storage read: %w", err)
		}

		// Read storage trailing counters from fixed position
		// Layout: StorageCommonCount×12 + key_count(4) + StorageKeyCount×12 + next_equip_index(4) + next_acq_sort_id(4)
		storageNextEquipOff := storageStart + StorageNextEquipIdxRel
		storageNextAcqOff := storageStart + StorageNextAcqSortRel
		if storageNextAcqOff+4 <= len(s.Data) {
			s.Storage.nextEquipIndexOff = storageNextEquipOff
			s.Storage.NextEquipIndex = binary.LittleEndian.Uint32(s.Data[storageNextEquipOff:])
			s.Storage.nextAcqSortIdOff = storageNextAcqOff
			s.Storage.NextAcquisitionSortId = binary.LittleEndian.Uint32(s.Data[storageNextAcqOff:])
		}
	}
	return nil
}

func (s *SaveSlot) Write(platform string) []byte {
	sa := NewSlotAccessor(s.Data)
	mo := s.MagicOffset

	// Errors in Write are programming bugs (offsets already validated in Read),
	// so we ignore errors here. If any fails, it means Read() had a bug.
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

	// NOTE: Per-slot SteamID is NOT written here. The offset is at a dynamic position within
	// the sequential data chain (after BaseVersion, before PS5Activity), NOT at SlotSize-8.
	// SlotSize-8 falls inside the PlayerGameDataHash region (last 0x80 bytes). Writing there
	// corrupts hash data. The primary SteamID is stored in UserData10 and flushed by
	// flushMetadata() — that is the authoritative source the game uses.

	// NOTE: CSPlayerGameDataHash (last 0x80 bytes) is intentionally NOT recomputed here.
	// All reference editors (ER-Save-Editor, er-save-manager, Final.py) treat this region
	// as read-only — they preserve the original bytes from the save file. The game does not
	// validate this hash on load. Recomputing it with a wrong algorithm corrupts those bytes
	// and causes EXCEPTION_ACCESS_VIOLATION (the game uses these offsets for equipment lookup).

	return s.Data
}

type ProfileSummary struct {
	CharacterName [16]uint16
	Level         uint32
}

func (p *ProfileSummary) Read(r *Reader) error {
	start := r.Pos()
	for i := 0; i < 16; i++ {
		p.CharacterName[i], _ = r.ReadU16()
	}
	p.Level, _ = r.ReadU32()
	r.Seek(int64(start+0x100), 0)
	return nil
}

func (p *ProfileSummary) Serialize(data []byte, offset int) {
	for i := 0; i < 16; i++ {
		binary.LittleEndian.PutUint16(data[offset+i*2:], p.CharacterName[i])
	}
	binary.LittleEndian.PutUint32(data[offset+32:], p.Level)
}

type CSMenuSystemSaveLoad struct {
	Data []byte
}

func (c *CSMenuSystemSaveLoad) Read(r *Reader) {
	c.Data, _ = r.ReadBytes(0x60000)
}

func UTF16ToString(u16 []uint16) string {
	for i, v := range u16 {
		if v == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}
