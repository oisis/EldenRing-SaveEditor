package core

import (
	"fmt"
)

// AddBulkItems adds a list of item IDs to the character's inventory
func (s *SaveSlot) AddBulkItems(itemIDs []uint32) int {
	addedCount := 0
	
	// Find existing items to avoid duplicates
	existingItems := make(map[uint32]bool)
	for _, item := range s.GaItems {
		if item.Handle != 0 {
			existingItems[item.ItemID] = true
		}
	}

	// Find the next available handle and slot
	nextHandle := s.findNextHandle()
	
	for _, id := range itemIDs {
		if existingItems[id] {
			continue
		}

		// Find an empty slot in GaItems (Handle == 0)
		slotIdx := s.findEmptySlot()
		if slotIdx == -1 {
			fmt.Println("Inventory full!")
			break
		}

		// Add the item
		s.GaItems[slotIdx] = GaItem{
			Handle: nextHandle,
			ItemID: id,
			Unk08:  0,
			Unk0C:  0,
			Unk10:  0,
		}
		
		nextHandle++
		addedCount++
	}

	return addedCount
}

func (s *SaveSlot) findNextHandle() uint32 {
	var maxHandle uint32 = 0x80000000 // Start in a safe range for added items
	for _, item := range s.GaItems {
		if item.Handle > maxHandle && item.Handle < 0x90000000 {
			maxHandle = item.Handle
		}
	}
	return maxHandle + 1
}

func (s *SaveSlot) findEmptySlot() int {
	for i, item := range s.GaItems {
		if item.Handle == 0 {
			return i
		}
	}
	return -1
}
