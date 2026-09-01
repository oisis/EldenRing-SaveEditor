package saveengine

// SetWeaponInfusionResult reports one committed affinity change.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetWeaponInfusionResult struct {
	MutationReceipt
	OwnedItemID    string `json:"ownedItemID"`
	CharacterID    int    `json:"characterID"`
	Container      string `json:"container"`
	PreviousGameID uint32 `json:"previousGameID"`
	GameID         uint32 `json:"gameID"`
}

// SetWeaponInfusion changes one weapon affinity while preserving its upgrade
// level, handle, container position and mounted Ash of War.
func (engine *Engine) SetWeaponInfusion(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	expectedRevision string,
	expectedGameID uint32,
	targetGameID uint32,
) (SetWeaponInfusionResult, error) {
	committed, container, err := engine.setOwnedWeaponGameID(
		saveSessionID, characterID, ownedItemID, expectedRevision, expectedGameID, targetGameID,
		kindSetWeaponInfusion, 0)
	if err != nil {
		return SetWeaponInfusionResult{}, err
	}
	return SetWeaponInfusionResult{
		MutationReceipt: committed, OwnedItemID: ownedItemID,
		CharacterID: characterID, Container: container, PreviousGameID: expectedGameID,
		GameID: targetGameID,
	}, nil
}
