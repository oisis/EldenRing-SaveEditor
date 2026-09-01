package saveengine

// SetWeaponInfusionResult reports one committed affinity change.
type SetWeaponInfusionResult struct {
	SaveSessionID  string `json:"saveSessionID"`
	SaveRevision   string `json:"saveRevision"`
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
	saveRevision, container, err := engine.setOwnedWeaponGameID(
		saveSessionID, characterID, ownedItemID, expectedRevision, expectedGameID, targetGameID,
		kindSetWeaponInfusion, 0)
	if err != nil {
		return SetWeaponInfusionResult{}, err
	}
	return SetWeaponInfusionResult{
		SaveSessionID: saveSessionID, SaveRevision: saveRevision, OwnedItemID: ownedItemID,
		CharacterID: characterID, Container: container, PreviousGameID: expectedGameID,
		GameID: targetGameID,
	}, nil
}
