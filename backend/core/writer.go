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
// reinforceTypeFromItemID extracts the reinforce_type (upgrade level) from a weapon item ID.
// Weapon IDs encode upgrade level as: baseID + infuseOffset + upgradeLevel
// where infuseOffset is a multiple of 100 (0=Standard, 100=Heavy, ..., 1200=Occult)
// and upgradeLevel is 0-25 (normal weapons) or 0-10 (boss/unique weapons).
// The reinforce_type stored in GaItemData equals the upgrade level.
// Source: ER-Save-Editor upsert_gaitem_data_list() uses item's reinforce_type_id from regulation.
func reinforceTypeFromItemID(itemID uint32) uint32 {
	return itemID % 100
}

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
	binary.LittleEndian.PutUint32(slot.Data[newEntryOff+4:], 0)                          // unk
	binary.LittleEndian.PutUint32(slot.Data[newEntryOff+8:], reinforceTypeFromItemID(itemID)) // reinforce_type
	binary.LittleEndian.PutUint32(slot.Data[newEntryOff+12:], 0)                         // unk1

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
				// The game resolves these DIRECTLY from the handle (itemID = HandleToItemID(handle)).
				// They are NEVER stored in the GaItems array — real saves have zero 0xA0/0xB0
				// entries in GaItems. Writing a GaItem for stackable items displaces empty fill
				// entries, causing the game to read past the fixed-count GaItem boundary → crash.
				handle = (id & 0x0FFFFFFF) | handlePrefix
				slot.GaMap[handle] = id
			} else {
				// Weapons, armor, AoW, and arrows: generate unique handle with GaMap indirection.
				var err error
				handle, err = generateUniqueHandle(slot, handlePrefix)
				if err != nil {
					return err
				}
				if err := writeGaItem(slot, handle, id, recordSize); err != nil {
					return err
				}
				slot.GaMap[handle] = id

				// Weapons and AoW must be registered in GaItemData — a separate section that
				// tracks all items ever acquired. The game looks up weapon properties (reinforce_type)
				// from this list on load; an absent entry causes EXCEPTION_ACCESS_VIOLATION.
				// Source: ER-Save-Editor upsert_gaitem_data_list() / upsert_projectile_list()
				//
				// Exception: arrows/bolts have weapon-type handles (0x80xxxxxx) but belong in the
				// projectile list (EquipProjectileData), NOT in GaItemData. The Rust reference editor
				// routes them via upsert_projectile_list() instead. Since we don't yet parse/write the
				// projectile section, we skip GaItemData registration for arrows — the game handles
				// projectile registration on its own when the item is first used.
				if (handlePrefix == ItemTypeWeapon && !db.IsArrowID(id)) || handlePrefix == ItemTypeAow {
					if err := upsertGaItemData(slot, id); err != nil {
						return err
					}
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
	// Extract part_id (byte 2, bits 16-23) from existing handles in GaMap.
	// Real handles: 0xTTPPCCCC where TT=type, PP=part_id (always 0x80 on real saves), CCCC=counter.
	// Reference: ER-Save-Editor generates handle = type | (part_gaitem_handle << 16) | counter
	partID := uint32(0x80) // default
	maxCounter := uint32(0)
	for h := range slot.GaMap {
		if h&GaHandleTypeMask == prefix {
			p := (h >> 16) & 0xFF
			if p != 0 {
				partID = p
			}
			counter := h & 0xFFFF
			if counter > maxCounter {
				maxCounter = counter
			}
		}
	}
	// Start from maxCounter+1 to avoid collisions
	h := prefix | (partID << 16) | (maxCounter + 1)
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
	gaLimit := slot.MagicOffset - DynPlayerData + 1

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

	// For records larger than 8 bytes (weapon=21, armor=16), the extra bytes
	// push trailing empty entries past gaLimit, causing the game to read into
	// PlayerGameData (crash). The game reads a FIXED count (5120) of entries.
	//
	// Fix (matching Final.py): shift ALL data after GaItems section backward
	// by (size-8) bytes, effectively removing empty bytes from the END of the
	// GaItems region. Then update MagicOffset and all dependent offsets.
	// This maintains: GaItems section size = constant = 5120 entries worth.
	if size > GaRecordItem {
		// The GaItem array has a fixed entry count (5120/5118). Adding a larger-than-8B
		// record at InventoryEnd consumes (size-8) extra bytes. To maintain the fixed
		// entry count, we shift ALL data from gaLimit onward RIGHT by (size-8) bytes.
		// This EXPANDS the GaItems section and moves MagicOffset (and everything after) right.
		// The slot's padding at the end absorbs the expansion.
		extraBytes := size - GaRecordItem

		// Shift everything from gaLimit to end of slot RIGHT by extraBytes
		// (last extraBytes of slot padding are lost — they were zeros anyway)
		copy(slot.Data[gaLimit+extraBytes:SlotSize], slot.Data[gaLimit:SlotSize-extraBytes])

		// Update all offsets (they reference positions at/after gaLimit, which shifted right)
		slot.MagicOffset += extraBytes
		slot.PlayerDataOffset += extraBytes
		slot.FaceDataOffset += extraBytes
		slot.StorageBoxOffset += extraBytes
		slot.GaItemDataOffset += extraBytes
		slot.IngameTimerOffset += extraBytes
		if slot.EventFlagsOffset > 0 {
			slot.EventFlagsOffset += extraBytes
		}
		if slot.Inventory.nextEquipIndexOff >= gaLimit {
			slot.Inventory.nextEquipIndexOff += extraBytes
		}
		if slot.Inventory.nextAcqSortIdOff >= gaLimit {
			slot.Inventory.nextAcqSortIdOff += extraBytes
		}
		if slot.Storage.nextEquipIndexOff >= gaLimit {
			slot.Storage.nextEquipIndexOff += extraBytes
		}
		if slot.Storage.nextAcqSortIdOff >= gaLimit {
			slot.Storage.nextAcqSortIdOff += extraBytes
		}
		// gaLimit itself increased (recalculate for fill below)
		gaLimit += extraBytes
	}

	// Rewrite empty fill from InventoryEnd to current gaLimit
	newGaLimit := slot.MagicOffset - DynPlayerData + 1
	fillPos := slot.InventoryEnd
	for fillPos+GaRecordItem <= newGaLimit {
		sa.WriteU32(fillPos, GaHandleEmpty)
		sa.WriteU32(fillPos+4, GaHandleInvalid)
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
		// Scan the physical pre-allocated storage array (1920 slots) to find and zero the handle.
		// Cannot use in-memory list index because ReadStorage skips empty slots (sparse).
		storageStart := slot.StorageBoxOffset + StorageHeaderSkip
		removed := 0
		for i := 0; i < StorageCommonCount; i++ {
			off := storageStart + i*InvRecordLen
			if err := sa.CheckBounds(off, InvRecordLen, "RemoveItemFromSlot/storage"); err != nil {
				break
			}
			h := binary.LittleEndian.Uint32(slot.Data[off:])
			if h == handle {
				binary.LittleEndian.PutUint32(slot.Data[off:], 0)
				binary.LittleEndian.PutUint32(slot.Data[off+4:], 0)
				binary.LittleEndian.PutUint32(slot.Data[off+8:], 0)
				removed++
			}
		}
		// Decrement common_inventory_items_distinct_count header
		if removed > 0 {
			countOff := slot.StorageBoxOffset
			if err := sa.CheckBounds(countOff, 4, "RemoveItemFromSlot/storage-count"); err == nil {
				currentCount := binary.LittleEndian.Uint32(slot.Data[countOff:])
				if currentCount >= uint32(removed) {
					binary.LittleEndian.PutUint32(slot.Data[countOff:], currentCount-uint32(removed))
				}
			}
		}
		// Update in-memory list
		for i, item := range slot.Storage.CommonItems {
			if item.GaItemHandle == handle {
				slot.Storage.CommonItems[i] = InventoryItem{GaItemHandle: 0, Quantity: 0, Index: 0}
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

	if isStorage {
		items = &slot.Storage.CommonItems
		startOffset = slot.StorageBoxOffset + StorageHeaderSkip
	} else {
		items = &slot.Inventory.CommonItems
		startOffset = slot.MagicOffset + InvStartFromMagic
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
		// Storage is pre-allocated (StorageCommonCount=1920 slots), same as held inventory.
		// Find first empty slot by scanning the binary data directly (the in-memory list
		// only contains non-empty items due to ReadStorage skipping gaps).
		storageCapacity := StorageCommonCount
		emptyIdx := -1
		for i := 0; i < storageCapacity; i++ {
			off := startOffset + i*InvRecordLen
			if off+InvRecordLen > len(slot.Data) {
				break
			}
			h := binary.LittleEndian.Uint32(slot.Data[off:])
			if h == GaHandleEmpty || h == GaHandleInvalid {
				emptyIdx = i
				break
			}
		}
		if emptyIdx < 0 {
			return io.ErrShortBuffer // All storage slots occupied
		}

		// Use next_equip_index as the Index value (matching Rust ER-Save-Editor behavior).
		// Clamp to be > InvEquipReservedMax and > max existing index to prevent collisions.
		nextListId := slot.Storage.NextEquipIndex
		if nextListId <= InvEquipReservedMax {
			nextListId = InvEquipReservedMax + 1
		}
		for i := 0; i < storageCapacity; i++ {
			off := startOffset + i*InvRecordLen
			if off+InvRecordLen > len(slot.Data) {
				break
			}
			h := binary.LittleEndian.Uint32(slot.Data[off:])
			if h == GaHandleEmpty || h == GaHandleInvalid {
				continue
			}
			typeBits := h & GaHandleTypeMask
			if typeBits != ItemTypeWeapon && typeBits != ItemTypeArmor &&
				typeBits != ItemTypeAccessory && typeBits != ItemTypeItem && typeBits != ItemTypeAow {
				continue
			}
			idx := binary.LittleEndian.Uint32(slot.Data[off+8:])
			if idx > InvEquipReservedMax && idx < 50000 && idx >= nextListId {
				nextListId = idx + 1
			}
		}

		newItem := InventoryItem{GaItemHandle: handle, Quantity: qty, Index: nextListId}
		off := startOffset + emptyIdx*InvRecordLen
		if err := sa.CheckBounds(off, InvRecordLen, "addToInventory/storage-insert"); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(slot.Data[off:], newItem.GaItemHandle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], newItem.Quantity)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], newItem.Index)

		// Advance BOTH counters and write back (matching Rust ER-Save-Editor).
		slot.Storage.NextEquipIndex = nextListId + 1
		slot.Storage.NextAcquisitionSortId = nextListId + 1
		if slot.Storage.nextEquipIndexOff > 0 {
			binary.LittleEndian.PutUint32(slot.Data[slot.Storage.nextEquipIndexOff:], slot.Storage.NextEquipIndex)
		}
		if slot.Storage.nextAcqSortIdOff > 0 {
			binary.LittleEndian.PutUint32(slot.Data[slot.Storage.nextAcqSortIdOff:], slot.Storage.NextAcquisitionSortId)
		}

		// Update common_inventory_items_distinct_count header.
		// The game uses this count to determine how many storage items to load.
		// Without this update, added items are invisible in-game (count stays 0).
		// Source: Rust ER-Save-Editor add_to_storage_common_items() increments common_item_count.
		countOff := slot.StorageBoxOffset // header is at StorageBoxOffset (before items)
		if err := sa.CheckBounds(countOff, 4, "addToInventory/storage-count"); err == nil {
			currentCount := binary.LittleEndian.Uint32(slot.Data[countOff:])
			binary.LittleEndian.PutUint32(slot.Data[countOff:], currentCount+1)
		}

		// Update in-memory list
		*items = append(*items, newItem)
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

		// Use next_equip_index as the Index value (matching Rust ER-Save-Editor).
		// The game uses this as CSGaItemIns index; items with index >= next_equip_index
		// are considered invalid and cause EXCEPTION_ACCESS_VIOLATION.
		// Clamp to be > InvEquipReservedMax (432) and > max existing index.
		nextListId := slot.Inventory.NextEquipIndex
		if nextListId <= InvEquipReservedMax {
			nextListId = InvEquipReservedMax + 1
		}
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

		// Advance BOTH counters and write back (matching Rust ER-Save-Editor).
		// The game requires next_equip_index > all item indices; without this update
		// the game considers new items invalid → crash.
		slot.Inventory.NextEquipIndex = nextListId + 1
		slot.Inventory.NextAcquisitionSortId = nextListId + 1
		if slot.Inventory.nextEquipIndexOff > 0 {
			binary.LittleEndian.PutUint32(slot.Data[slot.Inventory.nextEquipIndexOff:], slot.Inventory.NextEquipIndex)
		}
		if slot.Inventory.nextAcqSortIdOff > 0 {
			binary.LittleEndian.PutUint32(slot.Data[slot.Inventory.nextAcqSortIdOff:], slot.Inventory.NextAcquisitionSortId)
		}
	}

	return nil
}
