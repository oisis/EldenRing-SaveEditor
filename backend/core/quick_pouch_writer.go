package core

import (
	"encoding/binary"
	"fmt"
)

// QuickPouchSlotKind identifies one writable Quick Item or Pouch slot.
// Quick Items occupy values 0..9 and Pouch slots occupy values 10..15.
type QuickPouchSlotKind int

const (
	QuickPouchSlotQuick1 QuickPouchSlotKind = iota
	QuickPouchSlotQuick2
	QuickPouchSlotQuick3
	QuickPouchSlotQuick4
	QuickPouchSlotQuick5
	QuickPouchSlotQuick6
	QuickPouchSlotQuick7
	QuickPouchSlotQuick8
	QuickPouchSlotQuick9
	QuickPouchSlotQuick10
	QuickPouchSlotPouch1
	QuickPouchSlotPouch2
	QuickPouchSlotPouch3
	QuickPouchSlotPouch4
	QuickPouchSlotPouch5
	QuickPouchSlotPouch6
)

const (
	quickPouchSlotCount      = equipItemDataQuickCount + equipItemDataPouchCount
	quickPouchDynamicQuick   = ChrAsmEquipmentSize
	quickPouchDynamicPouch   = 0x80
	quickPouchGoodsItemMask  = 0x0FFFFFFF
	quickPouchGoodsItemClass = 0x40000000
)

// QuickPouchWrite is one entry in an atomic Quick Item / Pouch write batch.
// Handle == 0 clears the slot using the native empty representation.
type QuickPouchWrite struct {
	Slot   QuickPouchSlotKind
	Handle uint32
}

type resolvedQuickPouchWrite struct {
	slotIndex  int
	pairOff    int
	dynamicOff int
	itemID     uint32
	equipIndex uint32
	handle     uint32
}

// WriteQuickPouch applies a Quick Item / Pouch batch atomically.
//
// Native saves keep two synchronized representations:
//   - EquipItemData: {goods handle, physical Inventory row + 0x180}
//   - equipped-armaments tail: direct GoodsParam item ID
//
// The active Quick Item index, Inventory quantities, and hash[9] are preserved.
// Existing native saves do not establish a stable hash[9] calculation, while
// SaveFile.SaveFile still recalculates the enclosing slot MD5.
func (s *SaveSlot) WriteQuickPouch(writes []QuickPouchWrite) error {
	if s == nil {
		return fmt.Errorf("WriteQuickPouch: nil slot")
	}
	if s.EquippedSpellsOffset <= 0 {
		return fmt.Errorf("WriteQuickPouch: EquippedSpellsOffset not parsed")
	}
	if len(writes) == 0 {
		return nil
	}

	pairBase := s.EquippedSpellsOffset + DynEquipedSpells
	if pairBase < 0 || pairBase+DynEquipedItems > len(s.Data) {
		return fmt.Errorf("WriteQuickPouch: EquipItemData section out of bounds")
	}
	armamentsOff, err := s.equippedArmamentsOffset()
	if err != nil {
		return fmt.Errorf("WriteQuickPouch: %w", err)
	}

	resolved := make([]resolvedQuickPouchWrite, 0, len(writes))
	seenSlots := make(map[int]int, len(writes))
	for i, write := range writes {
		slotIndex := int(write.Slot)
		if slotIndex < 0 || slotIndex >= quickPouchSlotCount {
			return fmt.Errorf("WriteQuickPouch[%d]: unsupported slot kind %d", i, slotIndex)
		}
		if previous, exists := seenSlots[slotIndex]; exists {
			return fmt.Errorf("WriteQuickPouch[%d]: slot %d already written at writes[%d]", i, slotIndex, previous)
		}
		seenSlots[slotIndex] = i

		pairOff, dynamicOff := quickPouchOffsets(pairBase, armamentsOff, slotIndex)
		if pairOff < 0 || pairOff+8 > len(s.Data) || dynamicOff < 0 || dynamicOff+4 > len(s.Data) {
			return fmt.Errorf("WriteQuickPouch[%d]: slot %d out of bounds", i, slotIndex)
		}

		entry := resolvedQuickPouchWrite{
			slotIndex:  slotIndex,
			pairOff:    pairOff,
			dynamicOff: dynamicOff,
			equipIndex: GaHandleInvalid,
			itemID:     GaHandleInvalid,
		}
		if write.Handle != 0 {
			if write.Handle == GaHandleInvalid {
				return fmt.Errorf("WriteQuickPouch[%d]: handle 0xFFFFFFFF is invalid; use Handle=0 to clear a slot", i)
			}
			if write.Handle&GaHandleTypeMask != ItemTypeItem {
				return fmt.Errorf("WriteQuickPouch[%d]: handle 0x%08X is not a goods handle", i, write.Handle)
			}
			inventoryRow := s.inventoryRowForHandle(write.Handle)
			if inventoryRow < 0 {
				return fmt.Errorf("WriteQuickPouch[%d]: handle 0x%08X is not present in Inventory", i, write.Handle)
			}
			entry.handle = write.Handle
			entry.equipIndex = inventoryEquipIndexBase + uint32(inventoryRow)
			entry.itemID = quickPouchGoodsItemClass | (write.Handle & quickPouchGoodsItemMask)
		}
		resolved = append(resolved, entry)
	}

	if err := validateQuickPouchDuplicates(s, resolved); err != nil {
		return fmt.Errorf("WriteQuickPouch: %w", err)
	}

	for _, write := range resolved {
		binary.LittleEndian.PutUint32(s.Data[write.pairOff:], write.handle)
		binary.LittleEndian.PutUint32(s.Data[write.pairOff+4:], write.equipIndex)
		binary.LittleEndian.PutUint32(s.Data[write.dynamicOff:], write.itemID)
	}
	return nil
}

func quickPouchOffsets(pairBase, armamentsOff, slotIndex int) (pairOff, dynamicOff int) {
	if slotIndex < equipItemDataQuickCount {
		return pairBase + slotIndex*8, armamentsOff + quickPouchDynamicQuick + slotIndex*4
	}
	pouchIndex := slotIndex - equipItemDataQuickCount
	return pairBase + equipItemDataPouchOff + pouchIndex*8, armamentsOff + quickPouchDynamicPouch + pouchIndex*4
}

func validateQuickPouchDuplicates(s *SaveSlot, writes []resolvedQuickPouchWrite) error {
	raw, err := s.ReadEquippedState()
	if err != nil {
		return err
	}
	var final [quickPouchSlotCount]uint32
	for i := range raw.QuickItems {
		final[i] = raw.QuickItems[i].ItemID
	}
	for i := range raw.Pouch {
		final[equipItemDataQuickCount+i] = raw.Pouch[i].ItemID
	}
	for _, write := range writes {
		final[write.slotIndex] = write.handle
	}
	if err := validateGoodsFamilyDuplicates("Quick Items", final[:equipItemDataQuickCount]); err != nil {
		return err
	}
	return validateGoodsFamilyDuplicates("Pouch", final[equipItemDataQuickCount:])
}

func validateGoodsFamilyDuplicates(family string, handles []uint32) error {
	seen := make(map[uint32]int, len(handles))
	for i, handle := range handles {
		if handle == 0 || handle == GaHandleInvalid {
			continue
		}
		if previous, exists := seen[handle]; exists {
			return fmt.Errorf("%s item 0x%08X cannot occupy slots %d and %d", family, handle, previous+1, i+1)
		}
		seen[handle] = i
	}
	return nil
}
