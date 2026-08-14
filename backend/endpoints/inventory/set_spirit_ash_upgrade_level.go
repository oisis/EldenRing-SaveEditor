/*
Endpoint: SetSpiritAshUpgradeLevel
EndpointID: set_spirit_ash_upgrade_level
Purpose: Sets the upgrade level of one owned Spirit Ash.
How it works: The handler resolves the opaque owned record, asks GameCatalog for the exact stored grave- or ghost-glovewort variant, then delegates one atomic mutation to SaveEngine.
Supported resource types: ItemDocument: spirit_ash with a known and enabled grave- or ghost-glovewort upgrade capability.
Input variables: saveSessionID, characterID, ownedItemID, upgradeLevel, expectedRevision.
GameCatalog variables read: item.gameID, item.family, item.capabilities.upgrade and stored upgrade variants.
Save variables processed: one Inventory or Storage common handle, the target GaItemData entry, and matching Quick Items or Pouch references when the record is in Inventory; quantity, acquisition index and unrelated bytes stay unchanged.
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

const SetSpiritAshUpgradeLevelEndpointID = "set_spirit_ash_upgrade_level"

var SetSpiritAshUpgradeLevelDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetSpiritAshUpgradeLevel",
	ID:                         SetSpiritAshUpgradeLevelEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: spirit_ash with known enabled grave- or ghost-glovewort upgrade capability",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "ownedItemID", "upgradeLevel", "expectedRevision"},
	Description:                "Sets the upgrade level of one owned Spirit Ash.",
})

type SetSpiritAshUpgradeLevelResult = saveengine.SetSpiritAshUpgradeLevelResult

func SetSpiritAshUpgradeLevel(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	upgradeLevel uint8,
	expectedRevision string,
) (SetSpiritAshUpgradeLevelResult, error) {
	if engine == nil {
		return SetSpiritAshUpgradeLevelResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetSpiritAshUpgradeLevelResult{}, errors.New("game catalog is not available")
	}

	owned, err := engine.GetOwnedItem(saveSessionID, characterID, ownedItemID)
	if err != nil {
		return SetSpiritAshUpgradeLevelResult{}, err
	}
	gameIDs, err := engine.ResolveGaItemIDs(
		saveSessionID, characterID, []uint32{owned.GaItemHandle})
	if err != nil {
		return SetSpiritAshUpgradeLevelResult{}, err
	}
	currentGameID := gameIDs[0]
	targetGameID, err := gameCatalog.SpiritAshUpgradeTarget(currentGameID, upgradeLevel)
	if err != nil {
		return SetSpiritAshUpgradeLevelResult{}, fmt.Errorf("owned item %q: %w", ownedItemID, err)
	}
	return engine.SetSpiritAshUpgradeLevel(
		saveSessionID, characterID, ownedItemID, upgradeLevel, expectedRevision,
		currentGameID, targetGameID)
}
