package saveengine

// ps4StorageSlotBounds reports where the slot data of one PS4 character slot
// starts and where it ends. It is the only PS4-specific part of the storage
// getter: the PS4 container stores slots without the MD5 prefix the PC container
// uses, so the data begins at the block offset, and the Storage Box layout
// inside the slot is identical on both platforms.
func ps4StorageSlotBounds(characterID int) (int64, int64) {
	base := ps4SlotDataOffset + int64(characterID)*ps4SlotSize
	return base, base + ps4SlotSize
}
