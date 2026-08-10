package saveengine

// pcInventorySlotDataSize is the size of the slot data itself on PC: the PC
// container stores every slot as a 0x10-byte MD5 prefix followed by the data,
// and the prefix is skipped, never parsed or verified.
const pcInventorySlotDataSize = pcSlotBlockSize - 0x10

// pcInventorySlotBounds reports where the slot data of one PC character slot
// starts and where it ends. It is the only PC-specific part of the inventory
// getter: the InventoryHeld layout inside the slot is identical on both
// platforms, so the containers differ in this base alone.
func pcInventorySlotBounds(characterID int) (int64, int64) {
	base := pcSlotDataOffset + int64(characterID)*pcSlotBlockSize
	return base, base + pcInventorySlotDataSize
}
