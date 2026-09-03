package saveengine

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

const (
	itemOrderReservedIndexMax uint32 = 432
	itemOrderUnsafeIndex      uint32 = 10000
)

// SetInventoryOrderResult reports one committed supported Inventory order.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetInventoryOrderResult struct {
	MutationReceipt
	CharacterID        int      `json:"characterID"`
	GameIDs            []uint32 `json:"gameIDs"`
	AcquisitionIndices []uint32 `json:"acquisitionIndices"`
}

type inventoryOrderEntry struct {
	physicalIndex int
	gameID        uint32
	acquisition   uint32
}

// SetInventoryOrder atomically replaces the logical order of every supported
// Inventory common record. classifyGameID identifies the immutable catalog
// subset owned by the endpoint; SaveEngine owns identities and binary layout.
func (engine *Engine) SetInventoryOrder(
	saveSessionID string,
	characterID int,
	orderedOwnedItemIDs []string,
	expectedRevision string,
	classifyGameID func(uint32) (bool, error),
) (SetInventoryOrderResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetInventoryOrderResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if len(orderedOwnedItemIDs) == 0 {
		return SetInventoryOrderResult{}, fmt.Errorf("orderedOwnedItemIDs must not be empty")
	}
	if len(orderedOwnedItemIDs) > inventoryHeldCommonRecords {
		return SetInventoryOrderResult{}, fmt.Errorf(
			"orderedOwnedItemIDs contains %d records, want at most %d",
			len(orderedOwnedItemIDs), inventoryHeldCommonRecords)
	}
	if classifyGameID == nil {
		return SetInventoryOrderResult{}, fmt.Errorf("Inventory order classifier is required")
	}

	var gameIDs, acquisitionIndices []uint32
	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetInventoryOrder, characterID, func(loaded *loadedSave) error {
		if err := requireActiveCharacterAt(loaded, characterID, expectedRevision); err != nil {
			return err
		}
		var err error
		gameIDs, acquisitionIndices, err = applyInventoryOrder(
			loaded, characterID, orderedOwnedItemIDs, classifyGameID)
		return err
	})
	if err != nil {
		return SetInventoryOrderResult{}, err
	}

	return SetInventoryOrderResult{
		MutationReceipt:    committed,
		CharacterID:        characterID,
		GameIDs:            gameIDs,
		AcquisitionIndices: acquisitionIndices,
	}, nil
}

func planItemOrderIndices(
	storedNext uint32,
	count int,
	retainedBuckets map[uint32]struct{},
) ([]uint32, error) {
	base := uint64(storedNext)
	if base <= uint64(itemOrderReservedIndexMax) {
		base = uint64(itemOrderReservedIndexMax) + 2
	}
	if base%2 != 0 {
		base++
	}
	for bucket := range retainedBuckets {
		after := (uint64(bucket) + 1) * 2
		if after > base {
			base = after
		}
	}

	last := base + uint64(count-1)*2
	if last >= uint64(itemOrderUnsafeIndex) {
		return nil, fmt.Errorf(
			"item order would assign acquisition index %d, want at most %d",
			last, itemOrderUnsafeIndex-1)
	}

	indices := make([]uint32, count)
	for position := range indices {
		indices[position] = uint32(base) + uint32(position)*2
	}
	return indices, nil
}
