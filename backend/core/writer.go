package core

import (
	"encoding/binary"
	"io"
)

type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) WriteU8(v uint8) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *Writer) WriteU16(v uint16) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *Writer) WriteU32(v uint32) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *Writer) WriteI32(v int32) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *Writer) WriteU64(v uint64) error {
	return binary.Write(w.w, binary.LittleEndian, v)
}

func (w *Writer) WriteBytes(v []byte) error {
	_, err := w.w.Write(v)
	return err
}

// AddItemsToSlot adds multiple items to a specific save slot.
func AddItemsToSlot(slot *SaveSlot, itemIDs []uint32, upgradeLevel int, invMax, storageMax bool) error {
	for _, id := range itemIDs {
		// 1. Determine item type and handle prefix
		prefix := uint32(ItemTypeItem)
		recordSize := 8
		if (id & 0x80000000) != 0 {
			prefix = ItemTypeAow
			recordSize = 8
		} else if (id & 0x10000000) != 0 {
			prefix = ItemTypeArmor
			recordSize = 16
		} else if (id & 0x20000000) != 0 {
			prefix = ItemTypeAccessory
			recordSize = 8
		} else if (id & 0x0FFFFFFF) == id { // Weapons usually don't have high bits set in DB
			prefix = ItemTypeWeapon
			recordSize = 21
		}

		// Normalize ID for weapons (add upgrade level)
		finalID := id
		if prefix == ItemTypeWeapon && upgradeLevel > 0 {
			finalID += uint32(upgradeLevel)
		}

		// 2. Generate a unique handle
		// For stackable items, we might want to find existing one, but for simplicity
		// and to match Python logic, we'll just add a new one if it's a weapon/armor.
		// For others, we check if it already exists in GaMap.
		handle := uint32(0)
		if prefix == ItemTypeItem || prefix == ItemTypeAccessory || prefix == ItemTypeAow {
			for h, i := range slot.GaMap {
				if i == finalID {
					handle = h
					break
				}
			}
		}

		if handle == 0 {
			handle = generateUniqueHandle(slot, prefix)
			// Add to GaItems section in s.Data
			if err := writeGaItem(slot, handle, finalID, recordSize); err != nil {
				return err
			}
			slot.GaMap[handle] = finalID
		}

		// 3. Add to Inventory
		if invMax {
			if err := addToInventory(slot, handle, 1, false); err != nil {
				return err
			}
		}

		// 4. Add to Storage
		if storageMax {
			if err := addToInventory(slot, handle, 1, true); err != nil {
				return err
			}
		}
	}
	return nil
}

func generateUniqueHandle(slot *SaveSlot, prefix uint32) uint32 {
	// Start with a base handle and increment until unique
	// Python uses a similar approach or random, but sequential is safer for parity.
	h := prefix | 0x00010000
	for {
		if _, ok := slot.GaMap[h]; !ok {
			return h
		}
		h++
	}
}

func writeGaItem(slot *SaveSlot, handle, itemID uint32, size int) error {
	// Find the first empty space in GaItems section (usually after InventoryEnd)
	// For simplicity, we'll append at InventoryEnd if it doesn't exceed MagicOffset
	if slot.InventoryEnd+size >= slot.MagicOffset {
		return io.ErrShortBuffer // No space in GaItems section
	}

	binary.LittleEndian.PutUint32(slot.Data[slot.InventoryEnd:], handle)
	binary.LittleEndian.PutUint32(slot.Data[slot.InventoryEnd+4:], itemID)
	
	// Zero out the rest of the record if it's larger than 8 bytes
	for i := 8; i < size; i++ {
		slot.Data[slot.InventoryEnd+i] = 0
	}

	slot.InventoryEnd += size
	return nil
}

func addToInventory(slot *SaveSlot, handle uint32, qty uint32, isStorage bool) error {
	var items *[]InventoryItem
	var startOffset int
	var maxItems int

	if isStorage {
		items = &slot.Storage.CommonItems
		startOffset = slot.StorageBoxOffset + 4
		maxItems = 2048
	} else {
		items = &slot.Inventory.CommonItems
		startOffset = slot.MagicOffset + 505
		maxItems = 0xa80
	}

	// Check if already in inventory (for stackable items)
	for i, item := range *items {
		if item.GaItemHandle == handle {
			// Update quantity in memory and Data
			(*items)[i].Quantity += qty
			binary.LittleEndian.PutUint32(slot.Data[startOffset+i*12+4:], (*items)[i].Quantity)
			return nil
		}
	}

	// Find first empty slot
	if len(*items) >= maxItems {
		return io.ErrShortBuffer // Inventory full
	}

	newIdx := uint32(len(*items))
	newItem := InventoryItem{
		GaItemHandle: handle,
		Quantity:     qty,
		Index:        newIdx,
	}

	*items = append(*items, newItem)
	
	// Write to Data
	off := startOffset + int(newIdx)*12
	binary.LittleEndian.PutUint32(slot.Data[off:], newItem.GaItemHandle)
	binary.LittleEndian.PutUint32(slot.Data[off+4:], newItem.Quantity)
	binary.LittleEndian.PutUint32(slot.Data[off+8:], newItem.Index)

	return nil
}
