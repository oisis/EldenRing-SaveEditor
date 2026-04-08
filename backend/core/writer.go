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
// invQty and storageQty control quantities: 0 = skip, -1 = use provided max from caller, >0 = exact qty.
// forceStackable treats items as stackable goods (handle=id, 21-byte record) regardless of ID prefix.
// Used for arrows/bolts which have weapon-like 0x8x... IDs but are stackable in inventory.
func AddItemsToSlot(slot *SaveSlot, itemIDs []uint32, invQty, storageQty int, forceStackable bool) error {
	for _, id := range itemIDs {
		// 1. Determine item type and handle prefix from upper nibble
		var prefix uint32
		var recordSize int
		switch id & 0xF0000000 {
		case ItemTypeWeapon:
			prefix = ItemTypeWeapon
			recordSize = 21
		case ItemTypeArmor:
			prefix = ItemTypeArmor
			recordSize = 16
		case ItemTypeAccessory:
			prefix = ItemTypeAccessory
			recordSize = 8
		case ItemTypeItem:
			prefix = ItemTypeItem
			recordSize = 8
		case ItemTypeAow:
			prefix = ItemTypeAow
			recordSize = 8
		default:
			prefix = ItemTypeWeapon
			recordSize = 21
		}

		// 2. Generate a unique handle.
		// For stackable items (goods, talismans, arrows), reuse existing handle if item already in GaMap.
		handle := uint32(0)
		if prefix == ItemTypeItem || prefix == ItemTypeAccessory || prefix == ItemTypeAow || forceStackable {
			for h, i := range slot.GaMap {
				if i == id {
					handle = h
					break
				}
			}
		}

		if handle == 0 {
			// For stackable goods (0xB0) and talismans (0xA0), the game convention
			// is handle == ID (no indirection). character_vm.go reads these back as
			// itemID = item.GaItemHandle, so the handle must equal the item ID.
			// Arrows (forceStackable) also use handle == ID for stacking.
			// Weapons, armor, and AoW use separate handle→ID indirection via GaMap.
			if prefix == ItemTypeItem || prefix == ItemTypeAccessory || forceStackable {
				handle = id
			} else {
				handle = generateUniqueHandle(slot, prefix)
			}
			if err := writeGaItem(slot, handle, id, recordSize); err != nil {
				return err
			}
			slot.GaMap[handle] = id
		}

		// 3. Add to Inventory
		if invQty != 0 {
			qty := uint32(invQty)
			if err := addToInventory(slot, handle, qty, false); err != nil {
				return err
			}
		}

		// 4. Add to Storage
		if storageQty != 0 {
			qty := uint32(storageQty)
			if err := addToInventory(slot, handle, qty, true); err != nil {
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

	// Check if already in inventory (for stackable items).
	// SET quantity to the desired value (not ADD) — qty represents the target total,
	// not a delta. Prevents 10 existing + 99 max = 109 instead of 99.
	for i, item := range *items {
		if item.GaItemHandle == handle {
			(*items)[i].Quantity = qty
			binary.LittleEndian.PutUint32(slot.Data[startOffset+i*12+4:], qty)
			return nil
		}
	}

	if isStorage {
		// Storage uses a dynamic list — append at current length if capacity allows
		if len(*items) >= maxItems {
			return io.ErrShortBuffer
		}
		newIdx := uint32(len(*items))
		newItem := InventoryItem{GaItemHandle: handle, Quantity: qty, Index: newIdx}
		*items = append(*items, newItem)
		off := startOffset + int(newIdx)*12
		binary.LittleEndian.PutUint32(slot.Data[off:], newItem.GaItemHandle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], newItem.Quantity)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], newItem.Index)
	} else {
		// Inventory is fully pre-allocated — find first empty slot (handle == 0 or 0xFFFFFFFF)
		emptyIdx := -1
		for i, item := range *items {
			if item.GaItemHandle == 0 || item.GaItemHandle == 0xFFFFFFFF {
				emptyIdx = i
				break
			}
		}
		if emptyIdx < 0 {
			return io.ErrShortBuffer // All slots occupied
		}
		(*items)[emptyIdx] = InventoryItem{GaItemHandle: handle, Quantity: qty, Index: uint32(emptyIdx)}
		off := startOffset + emptyIdx*12
		binary.LittleEndian.PutUint32(slot.Data[off:], handle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], qty)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], uint32(emptyIdx))
	}

	return nil
}
