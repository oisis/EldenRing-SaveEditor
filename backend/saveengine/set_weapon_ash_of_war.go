package saveengine

import "fmt"

const (
	noCustomAshOfWarHandle       uint32 = 0x00000000
	legacyNoCustomAshOfWarHandle uint32 = 0xFFFFFFFF
)

// SetWeaponAshOfWarResult reports one committed in-place Ash of War reference
// change. Handles stay private because they are save-format implementation
// details, not stable public identities.
type SetWeaponAshOfWarResult struct {
	SaveSessionID          string `json:"saveSessionID"`
	SaveRevision           string `json:"saveRevision"`
	WeaponOwnedItemID      string `json:"weaponOwnedItemID"`
	CharacterID            int    `json:"characterID"`
	Container              string `json:"container"`
	WeaponGameID           uint32 `json:"weaponGameID"`
	PreviousAshOfWarGameID uint32 `json:"previousAshOfWarGameID"`
	AshOfWarGameID         uint32 `json:"ashOfWarGameID"`
}

// SetWeaponAshOfWar mounts an existing free Ash of War GaItem or removes the
// current attachment. A zero targetAshOfWarGameID means removal. The operation
// never creates, removes, moves or repacks a GaItem record.
func (engine *Engine) SetWeaponAshOfWar(
	saveSessionID string,
	characterID int,
	weaponOwnedItemID string,
	expectedRevision string,
	expectedWeaponGameID uint32,
	targetAshOfWarGameID uint32,
) (SetWeaponAshOfWarResult, error) {
	if expectedWeaponGameID&gaItemHandleTypeMask != 0 {
		return SetWeaponAshOfWarResult{}, fmt.Errorf(
			"weapon game ID must use prefix 0; got 0x%08X", expectedWeaponGameID)
	}
	if targetAshOfWarGameID != 0 && targetAshOfWarGameID&gaItemHandleTypeMask != 0x80000000 {
		return SetWeaponAshOfWarResult{}, fmt.Errorf(
			"Ash of War game ID must use prefix 8; got 0x%08X", targetAshOfWarGameID)
	}
	if !isCanonicalRevision(expectedRevision) {
		return SetWeaponAshOfWarResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var container string
	var previousAshOfWarGameID uint32
	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetWeaponAshOfWar, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}
		flag, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		locator, err := loaded.session.resolveOwnedItemID(characterID, weaponOwnedItemID)
		if err != nil {
			return err
		}
		if locator.containerSection != InventorySectionCommon {
			return fmt.Errorf(
				"weaponOwnedItemID %q must address a common container section", weaponOwnedItemID)
		}
		container = locator.container
		records, err := readOwnedRecords(loaded, characterID, locator.container)
		if err != nil {
			return err
		}
		var target *ownedRecord
		for index := range records {
			record := &records[index]
			if record.containerSection == locator.containerSection &&
				record.physicalIndex == locator.physicalIndex &&
				record.ownedItemID == weaponOwnedItemID {
				target = record
				break
			}
		}
		if target == nil {
			return fmt.Errorf("weaponOwnedItemID %q no longer addresses a record of character %d",
				weaponOwnedItemID, characterID)
		}
		if target.gaItemHandle&gaItemHandleTypeMask != gaItemWeaponHandle || target.quantity == 0 {
			return fmt.Errorf(
				"weaponOwnedItemID %q is not a positive-quantity weapon record", weaponOwnedItemID)
		}

		var weaponAt int64
		var currentAshOfWarHandle uint32
		weaponMatches := 0
		aoWRecords := make([]gaItemRecord, 0)
		aoWByHandle := make(map[uint32]gaItemRecord)
		aoWRecordCounts := make(map[uint32]int)
		weaponReferences := make(map[uint32][]uint32)
		err = walkGaItemRecords(loaded.snapshot, loaded.session.platform, characterID,
			func(record gaItemRecord) error {
				switch record.handle & gaItemHandleTypeMask {
				case gaItemWeaponHandle:
					if record.gameID&gaItemHandleTypeMask != 0 {
						return fmt.Errorf(
							"weapon handle 0x%08X maps to non-weapon game ID 0x%08X",
							record.handle, record.gameID)
					}
					aoWHandle, err := loaded.snapshot.uint32At(record.at + 16)
					if err != nil {
						return fmt.Errorf(
							"cannot read Ash of War reference of weapon 0x%08X: %w",
							record.handle, err)
					}
					if !isNoCustomAshOfWarHandle(aoWHandle) {
						if aoWHandle&gaItemHandleTypeMask != gaItemAshOfWarHandle {
							return fmt.Errorf(
								"weapon 0x%08X carries invalid Ash of War handle 0x%08X",
								record.handle, aoWHandle)
						}
						weaponReferences[aoWHandle] = append(weaponReferences[aoWHandle], record.handle)
					}
					if record.handle == target.gaItemHandle {
						weaponMatches++
						weaponAt = record.at
						currentAshOfWarHandle = aoWHandle
						if record.gameID != expectedWeaponGameID {
							return fmt.Errorf(
								"weaponOwnedItemID %q now denotes item 0x%08X, not the expected 0x%08X",
								weaponOwnedItemID, record.gameID, expectedWeaponGameID)
						}
					}
				case gaItemAshOfWarHandle:
					if record.gameID&gaItemHandleTypeMask != 0x80000000 {
						return fmt.Errorf(
							"Ash of War handle 0x%08X maps to non-Ash-of-War game ID 0x%08X",
							record.handle, record.gameID)
					}
					aoWRecords = append(aoWRecords, record)
					aoWByHandle[record.handle] = record
					aoWRecordCounts[record.handle]++
				}
				return nil
			})
		if err != nil {
			return fmt.Errorf("cannot inspect GaItem records of character %d: %w", characterID, err)
		}
		if weaponMatches != 1 {
			return fmt.Errorf("weapon handle 0x%08X has %d GaItem records, want exactly 1",
				target.gaItemHandle, weaponMatches)
		}

		if !isNoCustomAshOfWarHandle(currentAshOfWarHandle) {
			if aoWRecordCounts[currentAshOfWarHandle] != 1 {
				return fmt.Errorf(
					"current Ash of War handle 0x%08X has %d GaItem records, want exactly 1",
					currentAshOfWarHandle, aoWRecordCounts[currentAshOfWarHandle])
			}
			users := weaponReferences[currentAshOfWarHandle]
			if len(users) != 1 || users[0] != target.gaItemHandle {
				return fmt.Errorf(
					"current Ash of War handle 0x%08X is referenced by %d weapon records, want only 0x%08X",
					currentAshOfWarHandle, len(users), target.gaItemHandle)
			}
			previousAshOfWarGameID = aoWByHandle[currentAshOfWarHandle].gameID
		}

		targetHandle := noCustomAshOfWarHandle
		if targetAshOfWarGameID != 0 {
			if previousAshOfWarGameID == targetAshOfWarGameID {
				targetHandle = currentAshOfWarHandle
			} else {
				for _, candidate := range aoWRecords {
					if candidate.gameID == targetAshOfWarGameID &&
						aoWRecordCounts[candidate.handle] == 1 &&
						len(weaponReferences[candidate.handle]) == 0 {
						targetHandle = candidate.handle
						break
					}
				}
				if targetHandle == noCustomAshOfWarHandle {
					return fmt.Errorf(
						"character %d has no unique free Ash of War GaItem for game ID 0x%08X; allocation is unsupported",
						characterID, targetAshOfWarGameID)
				}
			}
		}

		if targetHandle == currentAshOfWarHandle {
			return nil
		}
		if err := applyByteWrites(loaded.snapshot, []byteWrite{{
			at: weaponAt + 16, data: littleEndianUint32(targetHandle),
		}}); err != nil {
			return fmt.Errorf("cannot change Ash of War of weapon %q: %w", weaponOwnedItemID, err)
		}
		return nil
	})
	if err != nil {
		return SetWeaponAshOfWarResult{}, err
	}
	return SetWeaponAshOfWarResult{
		SaveSessionID: saveSessionID, SaveRevision: committed.SaveRevision,
		WeaponOwnedItemID: weaponOwnedItemID, CharacterID: characterID,
		Container: container, WeaponGameID: expectedWeaponGameID,
		PreviousAshOfWarGameID: previousAshOfWarGameID,
		AshOfWarGameID:         targetAshOfWarGameID,
	}, nil
}

func isNoCustomAshOfWarHandle(handle uint32) bool {
	return handle == noCustomAshOfWarHandle || handle == legacyNoCustomAshOfWarHandle
}
