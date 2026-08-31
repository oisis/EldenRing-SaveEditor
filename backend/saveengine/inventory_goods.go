package saveengine

import (
	"errors"
	"fmt"
)

// InventoryGoodsPresence is one revision-coherent InventoryHeld membership
// read. Presence contains one entry for every distinct requested goods game ID.
type InventoryGoodsPresence struct {
	SaveSessionID string          `json:"saveSessionID"`
	SaveRevision  string          `json:"saveRevision"`
	CharacterID   int             `json:"characterID"`
	Active        bool            `json:"active"`
	Presence      map[uint32]bool `json:"presence"`
}

// GetInventoryGoodsPresence reports which requested goods game IDs have a
// positive-quantity record in common or key InventoryHeld. It accepts both
// native representations confirmed for goods: the handle-encoded 0xB form and
// the direct 0x4 game ID used by game-placed key items.
func (engine *Engine) GetInventoryGoodsPresence(
	saveSessionID string,
	characterID int,
	gameIDs []uint32,
) (InventoryGoodsPresence, error) {
	if saveSessionID == "" {
		return InventoryGoodsPresence{}, errors.New("saveSessionID is required")
	}

	byHandle := make(map[uint32]uint32, len(gameIDs)*2)
	present := make(map[uint32]bool, len(gameIDs))
	for _, gameID := range gameIDs {
		if gameID&gaItemHandleTypeMask != 0x40000000 {
			return InventoryGoodsPresence{}, fmt.Errorf(
				"goods game ID must use prefix 4; got 0x%08X", gameID)
		}
		present[gameID] = false
		byHandle[gameID] = gameID
		byHandle[gaItemGoodsHandle|(gameID&^gaItemHandleTypeMask)] = gameID
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return InventoryGoodsPresence{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return InventoryGoodsPresence{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}
	result := InventoryGoodsPresence{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
		Presence:      present,
	}
	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return InventoryGoodsPresence{}, fmt.Errorf(
			"cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue || len(gameIDs) == 0 {
		return result, nil
	}
	result.Active = true

	records, err := readInventoryRecords(loaded, characterID)
	if err != nil {
		return InventoryGoodsPresence{}, err
	}
	for _, record := range records {
		if record.Quantity == 0 {
			continue
		}
		if gameID, requested := byHandle[record.GaItemHandle]; requested {
			present[gameID] = true
		}
	}
	return result, nil
}
