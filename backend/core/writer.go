package core

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/oisis/EldenRing-SaveEditor/backend/db"
)

// upsertGaItemData ensures itemID is present in the GaItemData section.
// GaItemData tracks all weapons and AoW ever acquired; the game looks up weapon
// properties (reinforce_type) from this list on load. Adding a weapon without
// a corresponding GaItemData entry causes EXCEPTION_ACCESS_VIOLATION on load.
//
// Source: ER-Save-Editor src/vm/inventory/add_single.rs upsert_gaitem_data_list()
//
// Layout (at slot.GaItemDataOffset):
//   [0]   distinct_acquired_items_count (i32)
//   [4]   unk1 (i32) — preserve unchanged
//   [8+]  GaItem2 array: id(4) + unk(4) + reinforce_type(4) + unk1(4) per entry
func upsertGaItemData(slot *SaveSlot, itemID uint32) error {
	off := slot.GaItemDataOffset
	if off <= 0 {
		return nil // offset chain failed or not computed — skip silently
	}
	sa := NewSlotAccessor(slot.Data)

	// Read current count (first 4 bytes of GaItemData)
	if err := sa.CheckBounds(off, 4, "upsertGaItemData/count"); err != nil {
		return nil // non-fatal
	}
	count := int(int32(binary.LittleEndian.Uint32(slot.Data[off:])))
	if count < 0 || count >= GaItemDataMaxCount {
		return nil // corrupt or full — skip
	}

	// Scan existing entries for this itemID (only scan [0..count-1])
	arrayBase := off + GaItemDataArrayOff
	for i := 0; i < count; i++ {
		entryOff := arrayBase + i*GaItemDataEntryLen
		if err := sa.CheckBounds(entryOff, 4, "upsertGaItemData/scan"); err != nil {
			return nil
		}
		if binary.LittleEndian.Uint32(slot.Data[entryOff:]) == itemID {
			return nil // already present
		}
	}

	// Append new entry at position [count]
	newEntryOff := arrayBase + count*GaItemDataEntryLen
	if err := sa.CheckBounds(newEntryOff, GaItemDataEntryLen, "upsertGaItemData/write"); err != nil {
		return nil // non-fatal: no room
	}
	binary.LittleEndian.PutUint32(slot.Data[newEntryOff:], itemID)
	binary.LittleEndian.PutUint32(slot.Data[newEntryOff+4:], 0)  // unk
	binary.LittleEndian.PutUint32(slot.Data[newEntryOff+8:], 0)  // reinforce_type
	binary.LittleEndian.PutUint32(slot.Data[newEntryOff+12:], 0) // unk1

	// Write updated count
	binary.LittleEndian.PutUint32(slot.Data[off:], uint32(count+1))

	return nil
}

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
// forceStackable treats items as stackable (reuse existing GaMap handle) regardless of type.
// Used for arrows/bolts which have weapon-like IDs but are stackable in inventory.
func AddItemsToSlot(slot *SaveSlot, itemIDs []uint32, invQty, storageQty int, forceStackable bool) error {
	for _, id := range itemIDs {
		// 1. Convert item ID prefix to handle prefix and determine record size.
		// DB stores PC-style item IDs (0x00=weapon, 0x10=armor, 0x20=talisman, 0x40=goods, 0x80=AoW).
		// Handles always use GaItem prefixes (0x80=weapon, 0x90=armor, 0xA0=talisman, 0xB0=goods, 0xC0=AoW).
		handlePrefix := db.ItemIDToHandlePrefix(id)
		var recordSize int
		switch handlePrefix {
		case ItemTypeWeapon:
			recordSize = GaRecordWeapon
		case ItemTypeArmor:
			recordSize = GaRecordArmor
		case ItemTypeAccessory:
			recordSize = GaRecordAccessory
		case ItemTypeItem:
			recordSize = GaRecordItem
		case ItemTypeAow:
			recordSize = GaRecordAoW
		default:
			recordSize = GaRecordWeapon
		}

		// 2. Determine if item is stackable (handle=id pattern).
		// Stackable: talismans (0xA0), goods (0xB0). Handle = handlePrefix | lower bits of item ID.
		// Non-stackable: weapons (0x80), armor (0x90), AoW (0xC0). Handle = unique, GaMap indirection.
		// forceStackable: arrows — weapon-type but stackable; reuse GaMap handle if found.
		isStackable := handlePrefix == ItemTypeItem || handlePrefix == ItemTypeAccessory

		// 3. Search for existing handle in GaMap (for stackable reuse).
		handle := uint32(0)
		if isStackable || forceStackable {
			for h, i := range slot.GaMap {
				if i == id {
					handle = h
					break
				}
			}
		}

		if handle == 0 {
			if isStackable {
				// Stackable goods/talismans: handle = handlePrefix | lower bits of item ID.
				// The game reads these as: itemID = HandleToItemID(handle).
				handle = (id & 0x0FFFFFFF) | handlePrefix
			} else {
				// Weapons, armor, AoW, and arrows: generate unique handle with GaMap indirection.
				var err error
				handle, err = generateUniqueHandle(slot, handlePrefix)
				if err != nil {
					return err
				}
			}
			if err := writeGaItem(slot, handle, id, recordSize); err != nil {
				return err
			}
			slot.GaMap[handle] = id

			// Weapons and AoW must be registered in GaItemData — a separate section that
			// tracks all items ever acquired. The game looks up weapon properties (reinforce_type)
			// from this list on load; an absent entry causes EXCEPTION_ACCESS_VIOLATION.
			// Source: ER-Save-Editor upsert_gaitem_data_list() / upsert_projectile_list()
			if handlePrefix == ItemTypeWeapon || handlePrefix == ItemTypeAow {
				if err := upsertGaItemData(slot, id); err != nil {
					return err
				}
			}
		}

		// 4. Add to Inventory
		if invQty != 0 {
			qty := uint32(invQty)
			if err := addToInventory(slot, handle, qty, false); err != nil {
				return err
			}
		}

		// 5. Add to Storage
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
	gaLimit := slot.MagicOffset - DynPlayerData

	// Check BOTH constraints: GaItems must not overflow into Magic section,
	// AND must not exceed the physical buffer.
	if err := sa.CheckBounds(slot.InventoryEnd, size, "writeGaItem"); err != nil {
		return err
	}
	if slot.InventoryEnd+size > gaLimit {
		return fmt.Errorf("writeGaItem: no space in GaItems section "+
			"(InventoryEnd=0x%X + size=%d > gaLimit=0x%X)",
			slot.InventoryEnd, size, gaLimit)
	}

	// Write handle and itemID
	if err := sa.WriteU32(slot.InventoryEnd, handle); err != nil {
		return err
	}
	if err := sa.WriteU32(slot.InventoryEnd+4, itemID); err != nil {
		return err
	}

	// Write extra fields with correct sentinel defaults.
	// Source: ER-Save-Editor save_slot.rs GaItem::default() / GaItem::write()
	//
	// Weapon record (21 bytes total after handle+itemID: 13 extra bytes):
	//   [8-11]  unk2 = -1 (0xFFFFFFFF) — game reads as i32; 0 is a valid-looking pointer → crash
	//   [12-15] unk3 = -1 (0xFFFFFFFF) — same
	//   [16-19] aow_gaitem_handle = 0xFFFFFFFF — "no AoW attached"; 0 would resolve to empty handle
	//   [20]    unk5 = 0
	//
	// Armor record (16 bytes total: 8 extra bytes):
	//   [8-11]  unk2 = -1 (0xFFFFFFFF)
	//   [12-15] unk3 = -1 (0xFFFFFFFF)
	//
	// Writing zeros here (previous behavior) caused EXCEPTION_ACCESS_VIOLATION because the game
	// treats 0 as a valid GaItem handle / pointer rather than the null sentinel 0xFFFFFFFF.
	switch size {
	case GaRecordWeapon:
		if err := sa.WriteU32(slot.InventoryEnd+8, 0xFFFFFFFF); err != nil {
			return err
		}
		if err := sa.WriteU32(slot.InventoryEnd+12, 0xFFFFFFFF); err != nil {
			return err
		}
		if err := sa.WriteU32(slot.InventoryEnd+16, 0xFFFFFFFF); err != nil {
			return err
		}
		if err := sa.WriteU8(slot.InventoryEnd+20, 0); err != nil {
			return err
		}
	case GaRecordArmor:
		if err := sa.WriteU32(slot.InventoryEnd+8, 0xFFFFFFFF); err != nil {
			return err
		}
		if err := sa.WriteU32(slot.InventoryEnd+12, 0xFFFFFFFF); err != nil {
			return err
		}
	// GaRecordItem / GaRecordAccessory / GaRecordAoW: handle+itemID only (8 bytes), no extra fields.
	}
	slot.InventoryEnd += size

	// Fill the remaining pre-allocated empty slot region with clean 00000000|FFFFFFFF markers.
	//
	// Why this is necessary for weapon records (21B):
	//   After writing a 21B weapon at position P, the game's GaItem scanner reads
	//   the next entry at P+21. But the original pre-allocated empty slots sit on the
	//   8-byte grid (P+0, P+8, P+16, P+24 ...), so P+21 falls 5 bytes into the slot
	//   that started at P+16. The 4-byte handle read at P+21 spans the tail of the
	//   FFFFFFFF itemID (3×FF) and the first byte of the next slot handle (00), yielding
	//   0x00FFFFFF — an unknown type prefix. The game tries to dereference it as a
	//   pointer and crashes (EXCEPTION_ACCESS_VIOLATION reading 0xFFFFFFFFFFFFFFFF).
	//
	//   Writing 4 zeros at InventoryEnd (Fix #2, now removed) was insufficient:
	//   the scanner advances 8 bytes and lands at P+29, which is again misaligned.
	//   The two scanner grids {P+21, P+29, P+37, ...} and {P+0, P+8, P+16, ...}
	//   are offset by 5 and NEVER converge, so patching any finite number of bytes
	//   after the record does not help.
	//
	//   The only correct fix: rewrite all pre-allocated empty slots from the new
	//   InventoryEnd to gaLimit with a clean 00000000|FFFFFFFF pattern aligned to
	//   the game's NEW scanner grid. Every scanner step from InventoryEnd onward
	//   then hits a valid GaHandleEmpty regardless of the preceding record size.
	fillPos := slot.InventoryEnd
	for fillPos+GaRecordItem <= gaLimit {
		if err := sa.WriteU32(fillPos, GaHandleEmpty); err != nil {
			return err
		}
		if err := sa.WriteU32(fillPos+4, GaHandleInvalid); err != nil {
			return err
		}
		fillPos += GaRecordItem
	}

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
		// Compute safe next index (same logic as inventory path — see comment there).
		nextListId := slot.Storage.NextAcquisitionSortId
		maxExisting := uint32(InvEquipReservedMax)
		for _, item := range slot.Storage.CommonItems {
			if item.GaItemHandle != GaHandleEmpty && item.GaItemHandle != GaHandleInvalid && item.Index > maxExisting {
				maxExisting = item.Index
			}
		}
		if nextListId <= maxExisting {
			nextListId = maxExisting + 1
		}
		newItem := InventoryItem{GaItemHandle: handle, Quantity: qty, Index: nextListId}
		appendIdx := len(*items)
		*items = append(*items, newItem)
		off := startOffset + appendIdx*InvRecordLen
		if err := sa.CheckBounds(off, InvRecordLen, "addToInventory/storage-append"); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(slot.Data[off:], newItem.GaItemHandle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], newItem.Quantity)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], newItem.Index)

		// Update 4-byte storage item count header at StorageBoxOffset.
		newCount := uint32(len(*items))
		if err := sa.CheckBounds(slot.StorageBoxOffset, 4, "addToInventory/storage-header"); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(slot.Data[slot.StorageBoxOffset:], newCount)

		// Advance counter to nextListId+1 and write back (may have jumped over stale range).
		slot.Storage.NextAcquisitionSortId = nextListId + 1
		if slot.Storage.nextAcqSortIdOff > 0 {
			binary.LittleEndian.PutUint32(slot.Data[slot.Storage.nextAcqSortIdOff:], slot.Storage.NextAcquisitionSortId)
		}
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

		// Compute safe next index:
		//   1. Start from next_acquisition_sort_id stored in the save file.
		//   2. Clamp to be > InvEquipReservedMax (432) — CSGaItemIns[0..432] are reserved for
		//      equipment slots. If a new item lands in that range the game dereferences a wrong
		//      entry and crashes (EXCEPTION_ACCESS_VIOLATION).
		//   3. Clamp to be > max(existing item indices) — the counter can be stale (e.g., 242)
		//      while existing items already occupy 433-483. Using a stale value collides.
		nextListId := slot.Inventory.NextAcquisitionSortId
		maxExisting := uint32(InvEquipReservedMax)
		for _, item := range slot.Inventory.CommonItems {
			if item.GaItemHandle != GaHandleEmpty && item.GaItemHandle != GaHandleInvalid && item.Index > maxExisting {
				maxExisting = item.Index
			}
		}
		for _, item := range slot.Inventory.KeyItems {
			if item.GaItemHandle != GaHandleEmpty && item.GaItemHandle != GaHandleInvalid && item.Index > maxExisting {
				maxExisting = item.Index
			}
		}
		if nextListId <= maxExisting {
			nextListId = maxExisting + 1
		}

		(*items)[emptyIdx] = InventoryItem{GaItemHandle: handle, Quantity: qty, Index: nextListId}
		off := startOffset + emptyIdx*InvRecordLen
		if err := sa.CheckBounds(off, InvRecordLen, "addToInventory/inv-insert"); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(slot.Data[off:], handle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], qty)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], nextListId)

		// Advance counter to nextListId+1 and write back (may have jumped over stale range).
		slot.Inventory.NextAcquisitionSortId = nextListId + 1
		if slot.Inventory.nextAcqSortIdOff > 0 {
			binary.LittleEndian.PutUint32(slot.Data[slot.Inventory.nextAcqSortIdOff:], slot.Inventory.NextAcquisitionSortId)
		}
	}

	return nil
}
