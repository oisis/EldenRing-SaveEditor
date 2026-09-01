package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// PhysickEmptyTearID is the confirmed native value of one empty mixture
	// position. The endpoint uses it when a public resource position is null.
	PhysickEmptyTearID       = uint32(0xFFFFFFFF)
	physickFilledFlaskID     = uint32(0x400000FA)
	physickEmptyFlaskID      = uint32(0x400000FB)
	physickGameIDTypeMask    = uint32(0xF0000000)
	physickGoodsGameIDPrefix = uint32(0x40000000)
)

// SetPhysickMixtureResult reports one committed two-slot Physick assignment.
// Tears contains the exact save-side IDs written to the two positions;
// 0xFFFFFFFF is the native empty value.
type SetPhysickMixtureResult struct {
	SaveSessionID string    `json:"saveSessionID"`
	SaveRevision  string    `json:"saveRevision"`
	CharacterID   int       `json:"characterID"`
	Tears         [2]uint32 `json:"tears"`
}

// SetPhysickMixture atomically replaces both active Crystal Tear positions of
// one character. Every non-empty ID must be a distinct goods item the character
// carries with positive quantity in Inventory common or key, and the character
// must carry a Flask of Wondrous Physick. Storage is deliberately not consulted.
func (engine *Engine) SetPhysickMixture(
	saveSessionID string,
	characterID int,
	tears [2]uint32,
	expectedRevision string,
) (SetPhysickMixtureResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetPhysickMixtureResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	for index, tearID := range tears {
		if tearID == PhysickEmptyTearID {
			continue
		}
		if tearID&physickGameIDTypeMask != physickGoodsGameIDPrefix {
			return SetPhysickMixtureResult{}, fmt.Errorf(
				"physick tear %d has unsupported game ID 0x%08X; want a goods ID or 0xFFFFFFFF",
				index, tearID)
		}
	}
	if tears[0] != PhysickEmptyTearID && tears[0] == tears[1] {
		return SetPhysickMixtureResult{}, fmt.Errorf(
			"physick tear 0x%08X cannot occupy both positions", tears[0])
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetPhysickMixture, characterID, func(loaded *loadedSave) error {
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

		owned, hasFlask, err := ownedPhysickGoods(loaded, characterID)
		if err != nil {
			return err
		}
		if !hasFlask {
			return fmt.Errorf("character %d does not own a Flask of Wondrous Physick", characterID)
		}
		for index, tearID := range tears {
			if tearID == PhysickEmptyTearID {
				continue
			}
			if _, exists := owned[tearID]; !exists {
				return fmt.Errorf(
					"physick tear %d with game ID 0x%08X is not owned by character %d",
					index, tearID, characterID)
			}
		}

		blockAt, err := physickMixtureAt(loaded, characterID)
		if err != nil {
			return err
		}
		before, err := loaded.snapshot.readAt(blockAt, physickReadSize)
		if err != nil {
			return fmt.Errorf("cannot read physick mixture of character %d: %w", characterID, err)
		}

		after := make([]byte, physickReadSize)
		for index, tearID := range tears {
			binary.LittleEndian.PutUint32(after[index*4:], tearID)
		}
		if bytes.Equal(before, after) {
			return nil
		}
		if err := loaded.snapshot.writeAt(blockAt, after); err != nil {
			return fmt.Errorf("cannot write physick mixture of character %d: %w", characterID, err)
		}

		written, verifyErr := loaded.snapshot.readAt(blockAt, physickReadSize)
		if verifyErr == nil && bytes.Equal(written, after) {
			return nil
		}
		if rollback := loaded.snapshot.writeAt(blockAt, before); rollback != nil {
			return fmt.Errorf(
				"physick mixture of character %d could not be verified and could not be restored: %w",
				characterID, rollback)
		}
		return errors.New("physick mixture mutation could not be verified; the save is unchanged")
	})
	if err != nil {
		return SetPhysickMixtureResult{}, err
	}

	return SetPhysickMixtureResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  committed.SaveRevision,
		CharacterID:   characterID,
		Tears:         tears,
	}, nil
}

// ownedPhysickGoods returns every positive-quantity goods ID carried in the two
// InventoryHeld sections. It uses the Inventory reader's locator and decoder,
// but does not mint OwnedItemIDs while a mutation is still being validated.
func ownedPhysickGoods(loaded *loadedSave, characterID int) (map[uint32]struct{}, bool, error) {
	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return nil, false, err
	}
	section, err := loaded.snapshot.readAt(sectionAt, inventoryHeldSectionSize)
	if err != nil {
		return nil, false, fmt.Errorf("cannot read inventory of character %d: %w", characterID, err)
	}

	keyEnd := inventoryHeldSectionSize - inventoryHeldTrailingCounters
	records := appendInventoryRecords(
		make([]InventoryRecord, 0), section[:inventoryHeldCommonSize], InventorySectionCommon)
	records = appendInventoryRecords(records, section[inventoryHeldKeyAt:keyEnd], InventorySectionKey)

	owned := make(map[uint32]struct{})
	hasFlask := false
	for _, record := range records {
		if record.Quantity == 0 || record.GaItemHandle&gaItemHandleTypeMask != gaItemGoodsHandle {
			continue
		}
		gameID, err := resolveGaItemHandle(nil, record.GaItemHandle)
		if err != nil {
			return nil, false, fmt.Errorf(
				"cannot resolve inventory goods handle 0x%08X of character %d: %w",
				record.GaItemHandle, characterID, err)
		}
		owned[gameID] = struct{}{}
		if gameID == physickFilledFlaskID || gameID == physickEmptyFlaskID {
			hasFlask = true
		}
	}
	return owned, hasFlask, nil
}
