/*
Endpoint: SetWeaponUpgradeLevel
EndpointID: set_weapon_upgrade_level
Purpose: Sets the upgrade level of one owned weapon while preserving its base weapon and affinity.
How it works: The handler resolves the opaque owned record, asks GameCatalog for the target ID under that weapon's confirmed upgrade model and limit, then delegates one atomic mutation to SaveEngine.
Supported resource types: ItemDocument: weapon with a known and enabled standard or somber upgrade capability.
Input variables: saveSessionID, characterID, ownedItemID, upgradeLevel, expectedRevision.
GameCatalog variables read: item.gameID, item.family and item.capabilities.upgrade including model and maxLevel; stored affinity variants provide the preserved upgrade anchor.
Save variables processed: one weapon GaItem ID, the target GaItemData entry, the durable matchmaking weapon level byte and both item-ID representations of every matching equipped hand slot; the handle, row, quantity, acquisition index and unrelated bytes stay unchanged.
Implementation status: implemented
*/
package inventory

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const SetWeaponUpgradeLevelEndpointID = "set_weapon_upgrade_level"

var SetWeaponUpgradeLevelDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetWeaponUpgradeLevel",
	ID:                         SetWeaponUpgradeLevelEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: weapon with known enabled standard or somber upgrade capability",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "ownedItemID", "upgradeLevel", "expectedRevision"},
	Description:                "Sets the upgrade level of one owned weapon while preserving its base weapon and affinity.",
})

type SetWeaponUpgradeLevelResult = saveengine.SetWeaponUpgradeLevelResult

func SetWeaponUpgradeLevel(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	upgradeLevel uint8,
	expectedRevision string,
) (SetWeaponUpgradeLevelResult, error) {
	if engine == nil {
		return SetWeaponUpgradeLevelResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetWeaponUpgradeLevelResult{}, errors.New("game catalog is not available")
	}

	owned, err := engine.GetOwnedItem(saveSessionID, characterID, ownedItemID)
	if err != nil {
		return SetWeaponUpgradeLevelResult{}, err
	}
	gameIDs, err := engine.ResolveGaItemIDs(saveSessionID, characterID, []uint32{owned.GaItemHandle})
	if err != nil {
		return SetWeaponUpgradeLevelResult{}, err
	}
	currentGameID := gameIDs[0]
	targetGameID, err := gameCatalog.WeaponUpgradeTarget(currentGameID, upgradeLevel)
	if err != nil {
		return SetWeaponUpgradeLevelResult{}, fmt.Errorf("owned item %q: %w", ownedItemID, err)
	}
	matchmakingLevel, err := gameCatalog.WeaponMatchmakingLevel(currentGameID, upgradeLevel)
	if err != nil {
		return SetWeaponUpgradeLevelResult{}, fmt.Errorf("owned item %q: %w", ownedItemID, err)
	}

	return engine.SetWeaponUpgradeLevel(
		saveSessionID, characterID, ownedItemID, upgradeLevel, expectedRevision,
		currentGameID, targetGameID, matchmakingLevel)
}
