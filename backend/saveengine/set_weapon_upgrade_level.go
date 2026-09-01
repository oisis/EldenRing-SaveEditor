package saveengine

// SetWeaponUpgradeLevelResult reports one committed upgrade-level change.
// OwnedItemID identifies the record that was changed but is stale after the
// returned revision advances, like every other owned-item mutation receipt.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetWeaponUpgradeLevelResult struct {
	MutationReceipt
	OwnedItemID    string `json:"ownedItemID"`
	CharacterID    int    `json:"characterID"`
	Container      string `json:"container"`
	PreviousGameID uint32 `json:"previousGameID"`
	GameID         uint32 `json:"gameID"`
	UpgradeLevel   uint8  `json:"upgradeLevel"`
}

// SetWeaponUpgradeLevel changes the exact save-side game ID of one existing
// weapon record while preserving its base weapon and affinity.
func (engine *Engine) SetWeaponUpgradeLevel(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	upgradeLevel uint8,
	expectedRevision string,
	expectedGameID uint32,
	targetGameID uint32,
	matchmakingLevel uint8,
) (SetWeaponUpgradeLevelResult, error) {
	committed, container, err := engine.setOwnedWeaponGameID(
		saveSessionID, characterID, ownedItemID, expectedRevision, expectedGameID, targetGameID,
		kindSetWeaponUpgradeLevel, matchmakingLevel)
	if err != nil {
		return SetWeaponUpgradeLevelResult{}, err
	}
	return SetWeaponUpgradeLevelResult{
		MutationReceipt: committed, OwnedItemID: ownedItemID,
		CharacterID: characterID, Container: container, PreviousGameID: expectedGameID,
		GameID: targetGameID, UpgradeLevel: upgradeLevel,
	}, nil
}
