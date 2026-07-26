package core

import (
	"encoding/binary"
	"fmt"
)

// EquipmentSlotKind identifies a writable equipment slot within ChrAsmEquipment.
//
// Phase 7b.0 — backend-only foundation for weapon/ammo slots (0–9, hash 7),
// armor slots (12–15, hash 8), and the four player-visible talisman slots
// (17–20).
// The unknown slots 10/11/16 and EquippedGreatRune remain out of scope.
type EquipmentSlotKind int

const (
	EquipSlotLeftHandArmament1 EquipmentSlotKind = iota
	EquipSlotRightHandArmament1
	EquipSlotLeftHandArmament2
	EquipSlotRightHandArmament2
	EquipSlotLeftHandArmament3
	EquipSlotRightHandArmament3
	EquipSlotArrows1
	EquipSlotBolts1
	EquipSlotArrows2
	EquipSlotBolts2
	EquipSlotHead
	EquipSlotChest
	EquipSlotArms
	EquipSlotLegs
	EquipSlotTalisman1
	EquipSlotTalisman2
	EquipSlotTalisman3
	EquipSlotTalisman4
)

// equipmentSlotKindClass classifies what handle type a slot accepts.
type equipmentSlotKindClass int

const (
	slotClassWeapon   equipmentSlotKindClass = iota // accepts handle prefix 0x80 (ItemTypeWeapon)
	slotClassAmmo                                   // accepts handle prefix 0xB0 (ItemTypeItem / goods)
	slotClassArmor                                  // accepts handle prefix 0x90 (ItemTypeArmor)
	slotClassTalisman                               // accepts handle prefix 0xA0 (ItemTypeAccessory)
)

// equipmentSlotInfo maps a slot kind to its index in ChrAsmEquipment and its class.
type equipmentSlotInfo struct {
	index int                    // 0..21 within ChrAsmEquipment
	class equipmentSlotKindClass // expected handle class
}

var equipmentSlotTable = map[EquipmentSlotKind]equipmentSlotInfo{
	EquipSlotLeftHandArmament1:  {0, slotClassWeapon},
	EquipSlotRightHandArmament1: {1, slotClassWeapon},
	EquipSlotLeftHandArmament2:  {2, slotClassWeapon},
	EquipSlotRightHandArmament2: {3, slotClassWeapon},
	EquipSlotLeftHandArmament3:  {4, slotClassWeapon},
	EquipSlotRightHandArmament3: {5, slotClassWeapon},
	EquipSlotArrows1:            {6, slotClassAmmo},
	EquipSlotBolts1:             {7, slotClassAmmo},
	EquipSlotArrows2:            {8, slotClassAmmo},
	EquipSlotBolts2:             {9, slotClassAmmo},
	EquipSlotHead:               {12, slotClassArmor},
	EquipSlotChest:              {13, slotClassArmor},
	EquipSlotArms:               {14, slotClassArmor},
	EquipSlotLegs:               {15, slotClassArmor},
	EquipSlotTalisman1:          {17, slotClassTalisman},
	EquipSlotTalisman2:          {18, slotClassTalisman},
	EquipSlotTalisman3:          {19, slotClassTalisman},
	EquipSlotTalisman4:          {20, slotClassTalisman},
}

// EquipmentWrite is one entry in a WriteEquipment batch. Handle == 0 clears
// the slot using its native empty representation (Unarmed / bare armor for
// weapons and armor, invalid sentinels for ammunition and talismans).
type EquipmentWrite struct {
	Slot   EquipmentSlotKind
	Handle uint32
}

type nativeEquipmentWrite struct {
	equipIndexOff int
	itemIDOff     int
	handleOff     int
	equipIndex    uint32
	itemID        uint32
	handle        uint32
}

type resolvedEquipmentWrite struct {
	index   int
	class   equipmentSlotKindClass
	native  nativeEquipmentWrite
	dynamic uint32
}

// WriteEquipment applies a batch of equipment slot writes atomically.
//
// The game stores this state in four representations: an inventory-row index,
// a bare item ID, the exact inventory handle, and a direct item ID in the
// dynamic armaments block after acquired projectiles.
//
// Hash entries 7/8 are deliberately preserved. Their existing computed form
// does not match native T544/T547 values, so synthesizing them would be an
// unproven write. Slot MD5 is recalculated by SaveFile.SaveFile as usual.
//
// Concurrency: callers that share a SaveSlot across goroutines must hold the
// slot-level lock for the entire WriteEquipment call.
func (s *SaveSlot) WriteEquipment(writes []EquipmentWrite) error {
	if s == nil {
		return fmt.Errorf("WriteEquipment: nil slot")
	}
	if s.EquipItemsIDOffset <= 0 {
		return fmt.Errorf("WriteEquipment: EquipItemsIDOffset not parsed")
	}
	if s.EquipItemsIDOffset+equipmentStructLead+ChrAsmEquipmentSize > len(s.Data) {
		return fmt.Errorf("WriteEquipment: ChrAsmEquipment section out of bounds")
	}
	if len(writes) == 0 {
		return nil
	}
	armamentsOff, err := s.equippedArmamentsOffset()
	if err != nil {
		return fmt.Errorf("WriteEquipment: %w", err)
	}
	// Validate every write first and resolve every native representation before
	// mutating any of them.
	resolvedWrites := make([]resolvedEquipmentWrite, 0, len(writes))
	seenIndex := make(map[int]int, len(writes)) // index → position in writes for duplicate-detection diagnostics

	for i, w := range writes {
		info, ok := equipmentSlotTable[w.Slot]
		if !ok {
			return fmt.Errorf("WriteEquipment[%d]: unsupported slot kind %d (slots 10/11/16/21, spells, quick items, great rune, and unknown slots are out of scope)", i, int(w.Slot))
		}
		if prev, dup := seenIndex[info.index]; dup {
			return fmt.Errorf("WriteEquipment[%d]: slot index %d already written at writes[%d]", i, info.index, prev)
		}
		seenIndex[info.index] = i

		var native nativeEquipmentWrite
		var dynamic uint32
		var err error
		if info.class == slotClassTalisman {
			native, dynamic, err = s.resolveTalismanValues(info.index, w.Handle)
		} else {
			native, dynamic, err = s.resolveEquipmentValues(info.index, info.class, w.Handle)
		}
		if err != nil {
			return fmt.Errorf("WriteEquipment[%d]: %w", i, err)
		}
		resolvedWrites = append(resolvedWrites, resolvedEquipmentWrite{
			index:   info.index,
			class:   info.class,
			native:  native,
			dynamic: dynamic,
		})
	}

	if err := validateTalismanDuplicates(s, armamentsOff, resolvedWrites); err != nil {
		return fmt.Errorf("WriteEquipment: %w", err)
	}

	// All writes valid — perform the native updates.
	for _, r := range resolvedWrites {
		binary.LittleEndian.PutUint32(s.Data[r.native.equipIndexOff:], r.native.equipIndex)
		binary.LittleEndian.PutUint32(s.Data[r.native.itemIDOff:], r.native.itemID)
		binary.LittleEndian.PutUint32(s.Data[r.native.handleOff:], r.native.handle)
		binary.LittleEndian.PutUint32(s.Data[armamentsOff+r.index*4:], r.dynamic)
	}
	return nil
}

const (
	firstTalismanChrAsmIndex = 17
	talismanSlotCount        = 4
	inventoryEquipIndexBase  = 0x180
	equipmentStructLead      = 1
)

// equipmentRepresentationOffsets returns the three fixed player-data fields
// which accompany one dynamic equipped-armaments value:
//
//	EquipData: physical inventory row + 0x180
//	ChrAsm:    bare item ID (category prefix removed)
//	ChrAsm2:   exact inventory GaItem handle
//
// Each serialized structure has one leading byte before its 22 u32 equipment
// fields. T544/T547/T548 and the native T063/T064 ammunition saves establish
// the same +1 layout for weapons, ammunition, armor, and talismans.
func (s *SaveSlot) equipmentRepresentationOffsets(index int) (equipIndexOff, itemIDOff, handleOff int, err error) {
	if index < 0 || index >= ChrAsmFieldCount {
		return 0, 0, 0, fmt.Errorf("internal: equipment index %d out of range", index)
	}
	if s.MagicOffset < MinMagicOffset {
		return 0, 0, 0, fmt.Errorf("MagicOffset not parsed")
	}

	equipIndexBase := s.MagicOffset + DynSpEffect
	itemIDBase := equipIndexBase + DynEquipedItemIndex + DynActiveEquipedItems
	handleBase := itemIDBase + DynEquipedItemsID
	if handleBase != s.EquipItemsIDOffset {
		return 0, 0, 0, fmt.Errorf("equipment offset chain mismatch: ChrAsm2=0x%X EquipItemsIDOffset=0x%X", handleBase, s.EquipItemsIDOffset)
	}
	fieldOff := equipmentStructLead + index*4
	equipIndexOff = equipIndexBase + fieldOff
	itemIDOff = itemIDBase + fieldOff
	handleOff = handleBase + fieldOff
	for name, off := range map[string]int{
		"EquipData": equipIndexOff,
		"ChrAsm":    itemIDOff,
		"ChrAsm2":   handleOff,
	} {
		if off < 0 || off+4 > len(s.Data) {
			return 0, 0, 0, fmt.Errorf("%s equipment field out of bounds", name)
		}
	}
	return equipIndexOff, itemIDOff, handleOff, nil
}

func (s *SaveSlot) resolveTalismanValues(index int, handle uint32) (native nativeEquipmentWrite, dynamic uint32, err error) {
	native.equipIndexOff, native.itemIDOff, native.handleOff, err = s.equipmentRepresentationOffsets(index)
	if err != nil {
		return native, 0, err
	}
	if handle == 0 {
		native.equipIndex = GaHandleInvalid
		native.itemID = GaHandleInvalid
		native.handle = 0
		return native, GaHandleInvalid, nil
	}
	if handle == GaHandleInvalid {
		return native, 0, fmt.Errorf("handle 0xFFFFFFFF is invalid; use Handle=0 to clear a slot")
	}
	if handle&GaHandleTypeMask != ItemTypeAccessory {
		return native, 0, fmt.Errorf("handle 0x%08X is not a talisman handle", handle)
	}

	ordinal := index - firstTalismanChrAsmIndex
	unlocked := 1 + int(s.Player.TalismanSlots)
	if unlocked > talismanSlotCount {
		unlocked = talismanSlotCount
	}
	if ordinal >= unlocked {
		return native, 0, fmt.Errorf("talisman slot %d is locked (character has %d active slot(s))", ordinal+1, unlocked)
	}

	inventoryRow := -1
	for row, item := range s.Inventory.CommonItems {
		if item.GaItemHandle == handle && item.Quantity&0x7FFFFFFF != 0 {
			inventoryRow = row
			break
		}
	}
	if inventoryRow < 0 {
		return native, 0, fmt.Errorf("talisman handle 0x%08X is not present in inventory", handle)
	}

	bareItemID := handle & 0x0FFFFFFF
	if bareItemID == 0 {
		return native, 0, fmt.Errorf("talisman handle 0x%08X has an empty item ID", handle)
	}
	native.equipIndex = inventoryEquipIndexBase + uint32(inventoryRow)
	native.itemID = bareItemID
	native.handle = handle
	return native, 0x20000000 | bareItemID, nil
}

func validateTalismanDuplicates(s *SaveSlot, armamentsOff int, writes []resolvedEquipmentWrite) error {
	hasTalismanWrite := false
	for _, write := range writes {
		if write.class == slotClassTalisman {
			hasTalismanWrite = true
			break
		}
	}
	if !hasTalismanWrite {
		return nil
	}

	var final [talismanSlotCount]uint32
	for i := range final {
		final[i] = binary.LittleEndian.Uint32(s.Data[armamentsOff+(firstTalismanChrAsmIndex+i)*4:])
	}
	for _, write := range writes {
		if write.class == slotClassTalisman {
			final[write.index-firstTalismanChrAsmIndex] = write.dynamic
		}
	}

	seen := make(map[uint32]int, talismanSlotCount)
	unlocked := 1 + int(s.Player.TalismanSlots)
	if unlocked > talismanSlotCount {
		unlocked = talismanSlotCount
	}
	for i := 0; i < unlocked; i++ {
		raw := final[i]
		if raw == 0 || raw == GaHandleInvalid {
			continue
		}
		itemID := 0x20000000 | (raw & 0x0FFFFFFF)
		if previous, exists := seen[itemID]; exists {
			return fmt.Errorf("talisman 0x%08X cannot occupy slots %d and %d", itemID, previous+1, i+1)
		}
		seen[itemID] = i
	}
	return nil
}

const unarmedEquipmentItemID uint32 = 0x0001ADB0

var bareArmorItemIDBySlot = map[int]uint32{
	12: 0x10002710,
	13: 0x10002774,
	14: 0x100027D8,
	15: 0x1000283C,
}

func (s *SaveSlot) equippedArmamentsOffset() (int, error) {
	if s.EquippedSpellsOffset <= 0 {
		return 0, fmt.Errorf("EquippedSpellsOffset not parsed")
	}
	projHeaderOff := s.EquippedSpellsOffset + DynEquipedSpells + DynEquipedItems + DynEquipedGestures
	if projHeaderOff < 0 || projHeaderOff+4 > len(s.Data) {
		return 0, fmt.Errorf("projectile header out of bounds")
	}
	projCount := int(binary.LittleEndian.Uint32(s.Data[projHeaderOff:]))
	if projCount < 0 || projCount > MaxProjCount {
		return 0, fmt.Errorf("invalid projectile count %d", projCount)
	}
	armamentsOff := projHeaderOff + 4 + projCount*8
	if armamentsOff < projHeaderOff || armamentsOff+DynEquipedArmaments > len(s.Data) {
		return 0, fmt.Errorf("equipped armaments block out of bounds")
	}
	return armamentsOff, nil
}

func (s *SaveSlot) resolveEquipmentValues(index int, class equipmentSlotKindClass, handle uint32) (native nativeEquipmentWrite, dynamic uint32, err error) {
	native.equipIndexOff, native.itemIDOff, native.handleOff, err = s.equipmentRepresentationOffsets(index)
	if err != nil {
		return native, 0, err
	}
	if handle == 0 {
		switch class {
		case slotClassWeapon:
			handle, err = s.handleForItemID(unarmedEquipmentItemID, ItemTypeWeapon)
		case slotClassArmor:
			itemID, ok := bareArmorItemIDBySlot[index]
			if !ok {
				return native, 0, fmt.Errorf("no native empty armor item for slot %d", index)
			}
			handle, err = s.handleForItemID(itemID, ItemTypeArmor)
		case slotClassAmmo:
			native.equipIndex = GaHandleInvalid
			native.itemID = GaHandleInvalid
			native.handle = 0
			return native, GaHandleInvalid, nil
		default:
			return native, 0, fmt.Errorf("internal: unknown slot class %d", int(class))
		}
		if err != nil {
			return native, 0, err
		}
	}
	if handle == GaHandleInvalid {
		return native, 0, fmt.Errorf("handle 0xFFFFFFFF is invalid; use Handle=0 to clear a slot")
	}
	inventoryRow := s.inventoryRowForHandle(handle)
	if inventoryRow < 0 {
		return native, 0, fmt.Errorf("handle 0x%08X is not present in inventory", handle)
	}

	prefix := handle & GaHandleTypeMask
	switch class {
	case slotClassWeapon:
		switch prefix {
		case ItemTypeWeapon:
			// accepted below
		case ItemTypeAow:
			return native, 0, fmt.Errorf("handle 0x%08X has Ash of War prefix 0xC0; AoW equipping is out of scope for Phase 7b.0", handle)
		case ItemTypeArmor:
			return native, 0, fmt.Errorf("handle 0x%08X has armor prefix 0x90; cannot equip armor in a weapon slot", handle)
		case ItemTypeAccessory:
			return native, 0, fmt.Errorf("handle 0x%08X has talisman prefix 0xA0; cannot equip talisman in a weapon slot", handle)
		case ItemTypeItem:
			return native, 0, fmt.Errorf("handle 0x%08X has goods prefix 0xB0; cannot equip goods in a weapon slot", handle)
		default:
			return native, 0, fmt.Errorf("handle 0x%08X has unknown type prefix 0x%X for weapon slot", handle, prefix>>28)
		}
	case slotClassArmor:
		switch prefix {
		case ItemTypeArmor:
			// accepted below
		case ItemTypeWeapon:
			return native, 0, fmt.Errorf("handle 0x%08X has weapon prefix 0x80; cannot equip weapon in an armor slot", handle)
		case ItemTypeAow:
			return native, 0, fmt.Errorf("handle 0x%08X has Ash of War prefix 0xC0; cannot equip AoW in an armor slot", handle)
		case ItemTypeAccessory:
			return native, 0, fmt.Errorf("handle 0x%08X has talisman prefix 0xA0; cannot equip talisman in an armor slot", handle)
		case ItemTypeItem:
			return native, 0, fmt.Errorf("handle 0x%08X has goods prefix 0xB0; cannot equip goods in an armor slot", handle)
		default:
			return native, 0, fmt.Errorf("handle 0x%08X has unknown type prefix 0x%X for armor slot", handle, prefix>>28)
		}
	case slotClassAmmo:
		switch prefix {
		case ItemTypeWeapon, ItemTypeItem:
			// Native arrows/bolts use real GaItem records with an 0x80 handle.
			// accepted below
		case ItemTypeArmor:
			return native, 0, fmt.Errorf("handle 0x%08X has armor prefix 0x90; cannot equip armor in an ammo slot", handle)
		case ItemTypeAow:
			return native, 0, fmt.Errorf("handle 0x%08X has Ash of War prefix 0xC0; cannot equip AoW in an ammo slot", handle)
		case ItemTypeAccessory:
			return native, 0, fmt.Errorf("handle 0x%08X has talisman prefix 0xA0; cannot equip talisman in an ammo slot", handle)
		default:
			return native, 0, fmt.Errorf("handle 0x%08X has unknown type prefix 0x%X for ammo slot", handle, prefix>>28)
		}
	default:
		return native, 0, fmt.Errorf("internal: unknown slot class %d", int(class))
	}

	itemID, ok := s.GaMap[handle]
	if !ok || itemID == 0 || itemID == GaHandleInvalid {
		return native, 0, fmt.Errorf("handle 0x%08X not present in inventory (GaMap)", handle)
	}
	native.equipIndex = inventoryEquipIndexBase + uint32(inventoryRow)
	native.itemID = itemID & 0x0FFFFFFF
	native.handle = handle
	return native, itemID, nil
}

func (s *SaveSlot) inventoryRowForHandle(handle uint32) int {
	for row, item := range s.Inventory.CommonItems {
		if item.GaItemHandle == handle && item.Quantity&0x7FFFFFFF != 0 {
			return row
		}
	}
	return -1
}

func (s *SaveSlot) handleForItemID(itemID, expectedType uint32) (uint32, error) {
	for _, item := range s.Inventory.CommonItems {
		handle := item.GaItemHandle
		if item.Quantity&0x7FFFFFFF == 0 || handle&GaHandleTypeMask != expectedType {
			continue
		}
		if candidateID, ok := s.GaMap[handle]; ok && candidateID == itemID {
			return handle, nil
		}
	}
	return 0, fmt.Errorf("native empty item 0x%08X is not present in inventory/GaMap", itemID)
}
