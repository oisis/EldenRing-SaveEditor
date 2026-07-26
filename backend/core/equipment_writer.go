package core

import (
	"encoding/binary"
	"fmt"
)

// EquipmentSlotKind identifies a writable equipment slot within ChrAsmEquipment.
//
// Phase 7b.0 — backend-only foundation for weapon/ammo slots (0–9, hash 7) and
// armor slots (12–15, hash 8).
// Talismans are deliberately read-only until their native write contract has
// been established by dedicated Deck tests.
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
)

// equipmentSlotKindClass classifies what handle type a slot accepts.
type equipmentSlotKindClass int

const (
	slotClassWeapon equipmentSlotKindClass = iota // accepts handle prefix 0x80 (ItemTypeWeapon)
	slotClassAmmo                                 // accepts handle prefix 0xB0 (ItemTypeItem / goods)
	slotClassArmor                                // accepts handle prefix 0x90 (ItemTypeArmor)
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
}

// EquipmentWrite is one entry in a WriteEquipment batch. Handle == 0 clears
// the slot (writes 0xFFFFFFFF).
type EquipmentWrite struct {
	Slot   EquipmentSlotKind
	Handle uint32
}

// WriteEquipment applies a batch of equipment slot writes atomically.
//
// The game stores this state in multiple representations: an inventory-index
// block, encoded inventory handles in the fixed EquipItemsID header, and direct
// item IDs in the dynamic armaments block after acquired projectiles.
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
	if s.EquipItemsIDOffset+ChrAsmEquipmentSize > len(s.Data) {
		return fmt.Errorf("WriteEquipment: ChrAsmEquipment section out of bounds")
	}
	if len(writes) == 0 {
		return nil
	}
	armamentsOff, err := s.equippedArmamentsOffset()
	if err != nil {
		return fmt.Errorf("WriteEquipment: %w", err)
	}
	// Validate every write first; record both native representations before
	// mutating either one.
	type resolved struct {
		index   int
		header  uint32
		dynamic uint32
	}
	resolvedWrites := make([]resolved, 0, len(writes))
	seenIndex := make(map[int]int, len(writes)) // index → position in writes for duplicate-detection diagnostics

	for i, w := range writes {
		info, ok := equipmentSlotTable[w.Slot]
		if !ok {
			return fmt.Errorf("WriteEquipment[%d]: unsupported slot kind %d (slots 10/11/16, spells, quick items, great rune, and unknown slots are out of scope)", i, int(w.Slot))
		}
		if prev, dup := seenIndex[info.index]; dup {
			return fmt.Errorf("WriteEquipment[%d]: slot index %d already written at writes[%d]", i, info.index, prev)
		}
		seenIndex[info.index] = i

		currentHeader := binary.LittleEndian.Uint32(s.Data[s.EquipItemsIDOffset+info.index*4:])
		header, dynamic, err := s.resolveEquipmentValues(info.index, info.class, w.Handle, currentHeader)
		if err != nil {
			return fmt.Errorf("WriteEquipment[%d]: %w", i, err)
		}
		resolvedWrites = append(resolvedWrites, resolved{index: info.index, header: header, dynamic: dynamic})
	}

	// All writes valid — perform the paired updates.
	for _, r := range resolvedWrites {
		binary.LittleEndian.PutUint32(s.Data[s.EquipItemsIDOffset+r.index*4:], r.header)
		binary.LittleEndian.PutUint32(s.Data[armamentsOff+r.index*4:], r.dynamic)
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

func (s *SaveSlot) resolveEquipmentValues(index int, class equipmentSlotKindClass, handle, currentHeader uint32) (header, dynamic uint32, err error) {
	if handle == 0 {
		return s.emptyEquipmentValues(index, class, currentHeader)
	}
	if handle == GaHandleInvalid {
		return 0, 0, fmt.Errorf("handle 0xFFFFFFFF is invalid; use Handle=0 to clear a slot")
	}
	if !s.hasInventoryHandle(handle) {
		return 0, 0, fmt.Errorf("handle 0x%08X is not present in inventory", handle)
	}

	prefix := handle & GaHandleTypeMask
	switch class {
	case slotClassWeapon:
		switch prefix {
		case ItemTypeWeapon:
			// accepted below
		case ItemTypeAow:
			return 0, 0, fmt.Errorf("handle 0x%08X has Ash of War prefix 0xC0; AoW equipping is out of scope for Phase 7b.0", handle)
		case ItemTypeArmor:
			return 0, 0, fmt.Errorf("handle 0x%08X has armor prefix 0x90; cannot equip armor in a weapon slot", handle)
		case ItemTypeAccessory:
			return 0, 0, fmt.Errorf("handle 0x%08X has talisman prefix 0xA0; cannot equip talisman in a weapon slot", handle)
		case ItemTypeItem:
			return 0, 0, fmt.Errorf("handle 0x%08X has goods prefix 0xB0; cannot equip goods in a weapon slot", handle)
		default:
			return 0, 0, fmt.Errorf("handle 0x%08X has unknown type prefix 0x%X for weapon slot", handle, prefix>>28)
		}
	case slotClassArmor:
		switch prefix {
		case ItemTypeArmor:
			// accepted below
		case ItemTypeWeapon:
			return 0, 0, fmt.Errorf("handle 0x%08X has weapon prefix 0x80; cannot equip weapon in an armor slot", handle)
		case ItemTypeAow:
			return 0, 0, fmt.Errorf("handle 0x%08X has Ash of War prefix 0xC0; cannot equip AoW in an armor slot", handle)
		case ItemTypeAccessory:
			return 0, 0, fmt.Errorf("handle 0x%08X has talisman prefix 0xA0; cannot equip talisman in an armor slot", handle)
		case ItemTypeItem:
			return 0, 0, fmt.Errorf("handle 0x%08X has goods prefix 0xB0; cannot equip goods in an armor slot", handle)
		default:
			return 0, 0, fmt.Errorf("handle 0x%08X has unknown type prefix 0x%X for armor slot", handle, prefix>>28)
		}
	case slotClassAmmo:
		switch prefix {
		case ItemTypeWeapon, ItemTypeItem:
			// Native arrows/bolts use real GaItem records with an 0x80 handle.
			// accepted below
		case ItemTypeArmor:
			return 0, 0, fmt.Errorf("handle 0x%08X has armor prefix 0x90; cannot equip armor in an ammo slot", handle)
		case ItemTypeAow:
			return 0, 0, fmt.Errorf("handle 0x%08X has Ash of War prefix 0xC0; cannot equip AoW in an ammo slot", handle)
		case ItemTypeAccessory:
			return 0, 0, fmt.Errorf("handle 0x%08X has talisman prefix 0xA0; cannot equip talisman in an ammo slot", handle)
		default:
			return 0, 0, fmt.Errorf("handle 0x%08X has unknown type prefix 0x%X for ammo slot", handle, prefix>>28)
		}
	default:
		return 0, 0, fmt.Errorf("internal: unknown slot class %d", int(class))
	}

	itemID, ok := s.GaMap[handle]
	if !ok || itemID == 0 || itemID == GaHandleInvalid {
		return 0, 0, fmt.Errorf("handle 0x%08X not present in inventory (GaMap)", handle)
	}
	return ((handle & 0x00FFFFFF) << 8) | (currentHeader & 0xFF), itemID, nil
}

func (s *SaveSlot) hasInventoryHandle(handle uint32) bool {
	for _, item := range s.Inventory.CommonItems {
		if item.GaItemHandle == handle && item.Quantity&0x7FFFFFFF != 0 {
			return true
		}
	}
	return false
}

func (s *SaveSlot) emptyEquipmentValues(index int, class equipmentSlotKindClass, currentHeader uint32) (header, dynamic uint32, err error) {
	switch class {
	case slotClassWeapon:
		handle, err := s.handleForItemID(unarmedEquipmentItemID, ItemTypeWeapon)
		if err != nil {
			return 0, 0, err
		}
		return ((handle & 0x00FFFFFF) << 8) | (currentHeader & 0xFF), unarmedEquipmentItemID, nil
	case slotClassArmor:
		itemID, ok := bareArmorItemIDBySlot[index]
		if !ok {
			return 0, 0, fmt.Errorf("no native empty armor item for slot %d", index)
		}
		handle, err := s.handleForItemID(itemID, ItemTypeArmor)
		if err != nil {
			return 0, 0, err
		}
		return ((handle & 0x00FFFFFF) << 8) | (currentHeader & 0xFF), itemID, nil
	case slotClassAmmo:
		if index == 6 || index == 8 {
			return 0x00000080, GaHandleInvalid, nil
		}
		return 0, GaHandleInvalid, nil
	}
	return 0, 0, fmt.Errorf("internal: unknown slot class %d", int(class))
}

func (s *SaveSlot) handleForItemID(itemID, expectedType uint32) (uint32, error) {
	var selected uint32
	for handle, candidateID := range s.GaMap {
		if candidateID != itemID || handle&GaHandleTypeMask != expectedType {
			continue
		}
		if selected == 0 || handle < selected {
			selected = handle
		}
	}
	if selected == 0 {
		return 0, fmt.Errorf("native empty item 0x%08X is not present in GaMap", itemID)
	}
	return selected, nil
}
