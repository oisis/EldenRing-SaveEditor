package core

import (
	"encoding/binary"
	"fmt"
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
		switch id & GaHandleTypeMask {
		case ItemTypeWeapon:
			prefix = ItemTypeWeapon
			recordSize = GaRecordWeapon
		case ItemTypeArmor:
			prefix = ItemTypeArmor
			recordSize = GaRecordArmor
		case ItemTypeAccessory:
			prefix = ItemTypeAccessory
			recordSize = GaRecordAccessory
		case ItemTypeItem:
			prefix = ItemTypeItem
			recordSize = GaRecordItem
		case ItemTypeAow:
			prefix = ItemTypeAow
			recordSize = GaRecordAoW
		default:
			prefix = ItemTypeWeapon
			recordSize = GaRecordWeapon
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
				var err error
				handle, err = generateUniqueHandle(slot, prefix)
				if err != nil {
					return err
				}
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

func generateUniqueHandle(slot *SaveSlot, prefix uint32) (uint32, error) {
	h := prefix | GaHandleBase
	for i := 0; i < MaxHandleAttempts; i++ {
		if _, ok := slot.GaMap[h]; !ok {
			return h, nil
		}
		h++
	}
	return 0, fmt.Errorf("failed to generate unique handle after %d attempts (prefix 0x%X)",
		MaxHandleAttempts, prefix)
}

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

// RemoveItemFromSlot zeroes out inventory/storage slots for the given handle.
// Inventory: fixed pre-allocated array — zero the matching slot(s).
// Storage: dynamic list — zero the matching slot(s); game stops reading at handle==0.
// GaMap entry is removed only when the handle is absent from both lists after removal.
func RemoveItemFromSlot(slot *SaveSlot, handle uint32, fromInventory, fromStorage bool) error {
	sa := NewSlotAccessor(slot.Data)

	if fromInventory {
		invStart := slot.MagicOffset + InvStartFromMagic
		for i, item := range slot.Inventory.CommonItems {
			if item.GaItemHandle == handle {
				slot.Inventory.CommonItems[i] = InventoryItem{GaItemHandle: 0, Quantity: 0, Index: uint32(i)}
				off := invStart + i*InvRecordLen
				if err := sa.CheckBounds(off, InvRecordLen, "RemoveItemFromSlot/common"); err != nil {
					return err
				}
				binary.LittleEndian.PutUint32(slot.Data[off:], 0)
				binary.LittleEndian.PutUint32(slot.Data[off+4:], 0)
				binary.LittleEndian.PutUint32(slot.Data[off+8:], uint32(i))
			}
		}
		for i, item := range slot.Inventory.KeyItems {
			if item.GaItemHandle == handle {
				keyStart := invStart + CommonItemCount*InvRecordLen
				slot.Inventory.KeyItems[i] = InventoryItem{GaItemHandle: 0, Quantity: 0, Index: uint32(i)}
				off := keyStart + i*InvRecordLen
				if err := sa.CheckBounds(off, InvRecordLen, "RemoveItemFromSlot/key"); err != nil {
					return err
				}
				binary.LittleEndian.PutUint32(slot.Data[off:], 0)
				binary.LittleEndian.PutUint32(slot.Data[off+4:], 0)
				binary.LittleEndian.PutUint32(slot.Data[off+8:], uint32(i))
			}
		}
	}
	if fromStorage {
		storageStart := slot.StorageBoxOffset + StorageHeaderSkip
		for i, item := range slot.Storage.CommonItems {
			if item.GaItemHandle == handle {
				slot.Storage.CommonItems[i] = InventoryItem{GaItemHandle: 0, Quantity: 0, Index: 0}
				off := storageStart + i*InvRecordLen
				if err := sa.CheckBounds(off, InvRecordLen, "RemoveItemFromSlot/storage"); err != nil {
					return err
				}
				binary.LittleEndian.PutUint32(slot.Data[off:], 0)
				binary.LittleEndian.PutUint32(slot.Data[off+4:], 0)
				binary.LittleEndian.PutUint32(slot.Data[off+8:], 0)
			}
		}
	}
	// Remove from GaMap only if the handle is now absent from both lists.
	stillPresent := false
	for _, item := range slot.Inventory.CommonItems {
		if item.GaItemHandle == handle {
			stillPresent = true
			break
		}
	}
	if !stillPresent {
		for _, item := range slot.Inventory.KeyItems {
			if item.GaItemHandle == handle {
				stillPresent = true
				break
			}
		}
	}
	if !stillPresent {
		for _, item := range slot.Storage.CommonItems {
			if item.GaItemHandle == handle {
				stillPresent = true
				break
			}
		}
	}
	if !stillPresent {
		delete(slot.GaMap, handle)
	}
	return nil
}

func addToInventory(slot *SaveSlot, handle uint32, qty uint32, isStorage bool) error {
	sa := NewSlotAccessor(slot.Data)
	var items *[]InventoryItem
	var startOffset int
	var maxItems int

	if isStorage {
		items = &slot.Storage.CommonItems
		startOffset = slot.StorageBoxOffset + StorageHeaderSkip
		maxItems = StorageItemCount
	} else {
		items = &slot.Inventory.CommonItems
		startOffset = slot.MagicOffset + InvStartFromMagic
		maxItems = CommonItemCount
	}

	// Check if already in inventory (for stackable items).
	// SET quantity to the desired value (not ADD) — qty represents the target total,
	// not a delta. Prevents 10 existing + 99 max = 109 instead of 99.
	for i, item := range *items {
		if item.GaItemHandle == handle {
			(*items)[i].Quantity = qty
			off := startOffset + i*InvRecordLen + 4
			if err := sa.CheckBounds(off, 4, "addToInventory/update"); err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(slot.Data[off:], qty)
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
		off := startOffset + int(newIdx)*InvRecordLen
		if err := sa.CheckBounds(off, InvRecordLen, "addToInventory/storage-append"); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(slot.Data[off:], newItem.GaItemHandle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], newItem.Quantity)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], newItem.Index)
	} else {
		// Inventory is fully pre-allocated — find first empty slot (handle == 0 or 0xFFFFFFFF)
		emptyIdx := -1
		for i, item := range *items {
			if item.GaItemHandle == GaHandleEmpty || item.GaItemHandle == GaHandleInvalid {
				emptyIdx = i
				break
			}
		}
		if emptyIdx < 0 {
			return io.ErrShortBuffer // All slots occupied
		}
		(*items)[emptyIdx] = InventoryItem{GaItemHandle: handle, Quantity: qty, Index: uint32(emptyIdx)}
		off := startOffset + emptyIdx*InvRecordLen
		if err := sa.CheckBounds(off, InvRecordLen, "addToInventory/inv-insert"); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(slot.Data[off:], handle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], qty)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], uint32(emptyIdx))
	}

	return nil
}
