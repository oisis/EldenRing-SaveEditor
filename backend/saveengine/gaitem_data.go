package saveengine

import (
	"encoding/binary"
	"fmt"
)

// GaItemGameData — "GaItemData" — is the fixed block the event-flag reader
// already walks over on its way to the bitfield. This file is the only place
// that addresses the block itself, and it does exactly one thing: it inserts one
// active entry for an item ID the character does not own yet.
//
// The layout it depends on, stated here for this mutation alone:
//
//   - The whole section is eventFlagGaItemGameDataSize = 8 + 7000*16 bytes long
//     and its length never changes. It is preallocated by the game, so nothing
//     behind it ever moves and no section is repacked here.
//   - The first four bytes are the number of distinct active entries. The game
//     stores the count as a signed value, so anything at or above
//     gaItemDataMaxCount — which includes every negative int32 — is treated as
//     corrupt instead of followed.
//   - The four bytes behind the count are not interpreted and are never written.
//   - The active prefix is that many gaItemDataActiveEntrySize-byte records of
//     {item ID, 1}. Its capacity is gaItemDataMaxCount records, so the highest
//     byte this mutation can reach is 8 + 7000*8 — the first half of the block.
//     The second half and the whole unknown tail behind the active prefix stay
//     byte for byte as the game left them.
//   - Ash of War entries — item ID >> 28 == 8 — form one contiguous segment at
//     the end of the active prefix. This mutation adds ordinary entries only,
//     never an Ash of War one, so that segment is only ever shifted right as a
//     whole, with its order and its contents preserved.
//   - Inside the ordinary entries only the last ascending run is treated as
//     sorted. An insert is placed by lower bound inside that run, so an unsorted
//     legacy prefix in front of it is left exactly as it is rather than
//     "repaired".
//
// The rule above is the one SaveForge 1.5.8 and 1.6.8 applied to every ordinary
// family through upsertOrdinaryGaItemData, and it is reimplemented here rather
// than adopted: no legacy type, helper or package is used.
const (
	// gaItemDataArrayOffset is the distance from the first byte of the section to
	// the first active entry: the four-byte count and the four bytes behind it.
	gaItemDataArrayOffset = 8

	// gaItemDataActiveEntrySize is the stride of one active entry: the item ID
	// followed by the flag the game sets to 1.
	gaItemDataActiveEntrySize = 8

	// gaItemDataMaxCount is the capacity of the active prefix.
	gaItemDataMaxCount = 7000

	// gaItemDataAshOfWarPrefix is the high nibble of an Ash of War item ID.
	gaItemDataAshOfWarPrefix = 8
)

// planGaItemDataInsertion returns the writes that add one active entry for
// itemID, or no write at all when the section already carries it.
//
// It reads and validates everything it needs and changes no byte: the caller
// applies the returned writes together with the rest of its plan, so a rejected
// insert can never leave a half-written section behind.
//
// The caller must already hold Engine.mutex and must have established that the
// slot is active.
func planGaItemDataInsertion(loaded *loadedSave, characterID int, itemID uint32) ([]byteWrite, error) {
	sectionAt, slotEnd, err := eventFlagGaItemGameDataAt(loaded, characterID)
	if err != nil {
		return nil, err
	}
	if sectionAt+eventFlagGaItemGameDataSize > slotEnd {
		return nil, fmt.Errorf("GaItemData of character %d does not fit into its slot", characterID)
	}

	declared, err := loaded.snapshot.uint32At(sectionAt)
	if err != nil {
		return nil, fmt.Errorf("cannot read the GaItemData count of character %d: %w", characterID, err)
	}
	if declared >= gaItemDataMaxCount {
		return nil, fmt.Errorf(
			"character %d declares %d active GaItemData entries, want fewer than %d",
			characterID, int32(declared), gaItemDataMaxCount)
	}

	count := int(declared)
	arrayAt := sectionAt + gaItemDataArrayOffset
	active, err := loaded.snapshot.readAt(arrayAt, count*gaItemDataActiveEntrySize)
	if err != nil {
		return nil, fmt.Errorf("cannot read the GaItemData entries of character %d: %w", characterID, err)
	}
	entryID := func(index int) uint32 {
		return binary.LittleEndian.Uint32(active[index*gaItemDataActiveEntrySize:])
	}

	// An item ID the section already carries needs no second entry: the game
	// stores one active entry per distinct ID, and a further physical copy of an
	// owned ID never adds one.
	for index := 0; index < count; index++ {
		if entryID(index) == itemID {
			return nil, nil
		}
	}

	ordinaryEnd := count
	for index := 0; index < count; index++ {
		if entryID(index)>>28 == gaItemDataAshOfWarPrefix {
			ordinaryEnd = index
			break
		}
	}
	ordinaryStart := 0
	for index := ordinaryEnd - 1; index > 0; index-- {
		if entryID(index-1) > entryID(index) {
			ordinaryStart = index
			break
		}
	}
	insertIndex := ordinaryEnd
	for index := ordinaryStart; index < ordinaryEnd; index++ {
		if entryID(index) >= itemID {
			insertIndex = index
			break
		}
	}

	entry := make([]byte, gaItemDataActiveEntrySize)
	binary.LittleEndian.PutUint32(entry, itemID)
	binary.LittleEndian.PutUint32(entry[4:], 1)
	// The new entry and everything behind it inside the active prefix form one
	// contiguous write, so the shift and the insert cannot be applied apart from
	// each other. active is a private copy, so appending to it reaches no
	// snapshot byte.
	shifted := append(entry, active[insertIndex*gaItemDataActiveEntrySize:]...)

	return []byteWrite{
		{at: arrayAt + int64(insertIndex)*gaItemDataActiveEntrySize, data: shifted},
		{at: sectionAt, data: littleEndianUint32(declared + 1)},
	}, nil
}
