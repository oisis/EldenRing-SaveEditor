package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Rebuild slot layout constants.
//
// A character slot payload is a fixed characterSlotDataSize block (0x280000 bytes).
// The UnlockedRegions section is variable-length (uint32 count + N × uint32 region IDs).
// When the region list changes length, all subsequent sections shift by delta bytes.
//
// Every post-region section is sequentially walked and verified before any byte
// is copied. The tail padding between post-region sections and the fixed end-of-slot
// DLC block absorbs the shift, while the fixed end-of-slot DLC and PlayerGameDataHash
// blocks are preserved in their confirmed positions.
const (
	worldHeadSectionSize   = eventFlagHorseSize + eventFlagBloodStainSize // 117 bytes
	trailingFixedBlockSize = 12 + 12 + 16 + 8 + 32 + 50                   // Weather + Time + BaseVersion + SteamID + PS5Activity + DLC
	playerGameDataHashSize = 128
	slotFixedDlcSize       = int64(50)
	slotFixedHashSize      = int64(128)
	slotFixedDlcOffset     = characterSlotDataSize - slotFixedHashSize - slotFixedDlcSize // 0x27FF4E
	slotFixedHashOffset    = characterSlotDataSize - slotFixedHashSize                    // 0x27FF80
)

// rebuildSlotWithRegions serializes a single active character slot into a fresh
// 0x280000-byte buffer with the supplied regionIDs.
//
// It reads the snapshot directly through the codec and returns the new slot
// payload without mutating the snapshot, incrementing revisions, modifying undo
// history or changing dirty state.
//
// All offsets and lengths are computed using int64. The layout is rediscovered
// dynamically on every invocation without relying on cached values.
func rebuildSlotWithRegions(loaded *loadedSave, characterID int, regionIDs []uint32) ([]byte, error) {
	if loaded == nil || loaded.snapshot == nil {
		return nil, errors.New("loaded save snapshot is required")
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return nil, fmt.Errorf(
			"characterID %d is outside the range 0..%d", characterID, characterSlotCount-1)
	}
	if len(regionIDs) > regionMaxCount {
		return nil, fmt.Errorf(
			"requested %d unlocked regions, want at most %d", len(regionIDs), regionMaxCount)
	}

	userDataBase := userData10Base(loaded.session.platform)
	flag, err := loaded.snapshot.readAt(
		userDataBase+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return nil, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue {
		return nil, fmt.Errorf("character %d is not active", characterID)
	}

	slotBase, slotEnd := eventFlagSlotBounds(loaded.session.platform, characterID)
	if slotEnd-slotBase != characterSlotDataSize {
		return nil, fmt.Errorf(
			"character %d slot size %d does not match expected 0x%X",
			characterID, slotEnd-slotBase, characterSlotDataSize)
	}
	if !loaded.snapshot.covers(slotBase, characterSlotDataSize) {
		return nil, fmt.Errorf(
			"character %d slot exceeds the save file bounds", characterID)
	}

	slotVersion, err := loaded.snapshot.uint32At(slotBase)
	if err != nil {
		return nil, fmt.Errorf("cannot read slot version of character %d: %w", characterID, err)
	}
	if slotVersion == 0 {
		return nil, fmt.Errorf("character %d declares no slot version", characterID)
	}

	countAt, origCount, _, err := unlockedRegionsBounds(loaded, characterID)
	if err != nil {
		return nil, err
	}
	preRegsLen := countAt - slotBase
	if preRegsLen < 0 || preRegsLen > characterSlotDataSize {
		return nil, fmt.Errorf("character %d invalid pre-regions length %d", characterID, preRegsLen)
	}
	origRegsEnd := countAt + 4 + origCount*regionRecordSize
	if origRegsEnd > slotEnd {
		return nil, fmt.Errorf("character %d unlocked regions end outside slot", characterID)
	}

	// Sequentially walk and validate the entire confirmed post-regions chain.
	pos := origRegsEnd

	if pos+worldHeadSectionSize > slotEnd {
		return nil, fmt.Errorf("world head of character %d lies outside slot", characterID)
	}
	pos += worldHeadSectionSize

	menuPayloadSize, err := eventFlagDeclaredValue(
		loaded, characterID, pos+4, slotEnd, "menu profile size", eventFlagMaxDynamicSize)
	if err != nil {
		return nil, err
	}
	pos += eventFlagDynamicHeaderSize + menuPayloadSize

	if pos+eventFlagTrophyEquipSize > slotEnd {
		return nil, fmt.Errorf("trophy equip of character %d lies outside slot", characterID)
	}
	pos += eventFlagTrophyEquipSize

	if pos+eventFlagGaItemGameDataSize > slotEnd {
		return nil, fmt.Errorf("gaitem game data of character %d lies outside slot", characterID)
	}
	pos += eventFlagGaItemGameDataSize

	tutPayloadSize, err := eventFlagDeclaredValue(
		loaded, characterID, pos+4, slotEnd, "tutorial size", eventFlagMaxDynamicSize)
	if err != nil {
		return nil, err
	}
	pos += eventFlagDynamicHeaderSize + tutPayloadSize

	if pos+eventFlagScalarsSize > slotEnd {
		return nil, fmt.Errorf("scalars of character %d lie outside slot", characterID)
	}
	pos += eventFlagScalarsSize

	eventFlagsSize := int64(eventFlagSectionSize + eventFlagTerminatorSize)
	if pos+eventFlagsSize > slotEnd {
		return nil, fmt.Errorf("event flags of character %d lie outside slot", characterID)
	}
	pos += eventFlagsSize

	for index, maximum := range worldBlockLimits {
		if pos+4 > slotEnd {
			return nil, fmt.Errorf(
				"world block %d of character %d lies outside slot", index, characterID)
		}
		declared, err := loaded.snapshot.uint32At(pos)
		if err != nil {
			return nil, fmt.Errorf(
				"cannot read world block %d size of character %d: %w", index, characterID, err)
		}
		size := int64(int32(declared))
		if size < 0 || size >= maximum {
			return nil, fmt.Errorf(
				"character %d declares a world block %d size of %d, want 0..%d",
				characterID, index, size, maximum-1)
		}
		pos += 4 + size
		if pos > slotEnd {
			return nil, fmt.Errorf(
				"world block %d payload of character %d exceeds slot", index, characterID)
		}
	}

	if pos+playerCoordinatesSize > slotEnd {
		return nil, fmt.Errorf("player coordinates of character %d lie outside slot", characterID)
	}
	pos += playerCoordinatesSize

	spawnSize := int64(spawnPointFixedSize)
	if slotVersion >= spawnPointTempVersion {
		spawnSize += 4
	}
	if slotVersion >= spawnPointGameManByteVersion {
		spawnSize++
	}
	if pos+spawnSize > slotEnd {
		return nil, fmt.Errorf("spawn point block of character %d lies outside slot", characterID)
	}
	pos += spawnSize

	if pos+netManSectionSize > slotEnd {
		return nil, fmt.Errorf("netman of character %d lies outside slot", characterID)
	}
	pos += netManSectionSize

	if pos+trailingFixedBlockSize > slotEnd {
		return nil, fmt.Errorf("trailing fixed block of character %d lies outside slot", characterID)
	}
	pos += trailingFixedBlockSize

	if pos+playerGameDataHashSize > slotEnd {
		return nil, fmt.Errorf("player game data hash of character %d lies outside slot", characterID)
	}
	pos += playerGameDataHashSize

	origPostSectionsEnd := pos
	postSectionsLen := origPostSectionsEnd - origRegsEnd

	fixedDlcAt := slotBase + slotFixedDlcOffset
	if origPostSectionsEnd > fixedDlcAt {
		return nil, fmt.Errorf(
			"post-regions sections of character %d extend past fixed DLC offset (0x%X > 0x%X)",
			characterID, origPostSectionsEnd-slotBase, slotFixedDlcOffset)
	}
	originalTailLen := fixedDlcAt - origPostSectionsEnd

	newRegsLen := int64(4 + len(regionIDs)*regionRecordSize)
	newPostSectionsStart := preRegsLen + newRegsLen
	newPostSectionsEnd := newPostSectionsStart + postSectionsLen

	if newPostSectionsEnd > slotFixedDlcOffset {
		return nil, fmt.Errorf(
			"rebuilt slot for character %d overflows available capacity (post sections end at 0x%X > DLC offset 0x%X)",
			characterID, newPostSectionsEnd, slotFixedDlcOffset)
	}

	origRegsLen := origRegsEnd - countAt
	delta := newRegsLen - origRegsLen

	freshSlot := make([]byte, characterSlotDataSize)

	// 1. Copy pre-regions verbatim.
	preBytes, err := loaded.snapshot.readAt(slotBase, int(preRegsLen))
	if err != nil {
		return nil, err
	}
	copy(freshSlot[:preRegsLen], preBytes)

	// 2. Serialize new UnlockedRegions.
	binary.LittleEndian.PutUint32(freshSlot[preRegsLen:], uint32(len(regionIDs)))
	for i, id := range regionIDs {
		binary.LittleEndian.PutUint32(freshSlot[preRegsLen+4+int64(i)*regionRecordSize:], id)
	}

	// 3. Copy post-regions contiguous block.
	postBytes, err := loaded.snapshot.readAt(origRegsEnd, int(postSectionsLen))
	if err != nil {
		return nil, err
	}
	copy(freshSlot[newPostSectionsStart:newPostSectionsEnd], postBytes)

	// 4. Handle tail padding and fixed end-of-slot structures.
	if delta == 0 {
		tailBytes, err := loaded.snapshot.readAt(origPostSectionsEnd, int(slotEnd-origPostSectionsEnd))
		if err != nil {
			return nil, err
		}
		copy(freshSlot[newPostSectionsEnd:], tailBytes)
	} else {
		if delta < 0 {
			// Shift all available tailRest to the left.
			if originalTailLen > 0 {
				tailBytes, err := loaded.snapshot.readAt(origPostSectionsEnd, int(originalTailLen))
				if err != nil {
					return nil, err
				}
				copy(freshSlot[newPostSectionsEnd:newPostSectionsEnd+originalTailLen], tailBytes)
			}
			// Newly freed space before fixed DLC remains zeroed from allocation.
		} else {
			// delta > 0: Shift kept prefix of tailRest to the right, truncating the last delta bytes.
			keptTailLen := originalTailLen - delta
			if keptTailLen > 0 {
				keptBytes, err := loaded.snapshot.readAt(origPostSectionsEnd, int(keptTailLen))
				if err != nil {
					return nil, err
				}
				copy(freshSlot[newPostSectionsEnd:newPostSectionsEnd+keptTailLen], keptBytes)
			}
		}

		// Fixed DLC section.
		dlcBytes, err := loaded.snapshot.readAt(fixedDlcAt, int(slotFixedDlcSize))
		if err != nil {
			return nil, err
		}
		copy(freshSlot[slotFixedDlcOffset:slotFixedDlcOffset+slotFixedDlcSize], dlcBytes)

		// Fixed Hash section.
		hashBytes, err := loaded.snapshot.readAt(slotBase+slotFixedHashOffset, int(slotFixedHashSize))
		if err != nil {
			return nil, err
		}
		copy(freshSlot[slotFixedHashOffset:slotFixedHashOffset+slotFixedHashSize], hashBytes)
	}

	return freshSlot, nil
}
