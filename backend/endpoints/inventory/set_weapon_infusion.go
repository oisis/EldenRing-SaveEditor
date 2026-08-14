/*
Endpoint: SetWeaponInfusion
EndpointID: set_weapon_infusion
Purpose: Sets the affinity of one owned weapon while preserving its upgrade level.
How it works: The handler resolves the opaque owned record, asks GameCatalog for the target affinity ID, then delegates one atomic mutation to SaveEngine.
Supported resource types: ItemDocument: weapon with a known and enabled infusion capability.
Input variables: saveSessionID, characterID, ownedItemID, affinity, expectedRevision.
GameCatalog variables read: item.gameID, item.family, item.variants and item.capabilities.infusion; upgrade rules prove that the current level remains valid under the target affinity.
Save variables processed: one weapon GaItem ID, the target GaItemData entry and both item-ID representations of every matching equipped hand slot; Ash of War and unrelated bytes stay unchanged.
Implementation status: implemented
*/
package inventory

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const SetWeaponInfusionEndpointID = "set_weapon_infusion"

var SetWeaponInfusionDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetWeaponInfusion",
	ID:                         SetWeaponInfusionEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: weapon with known enabled infusion capability",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "ownedItemID", "affinity", "expectedRevision"},
	Description:                "Sets the affinity of one owned weapon while preserving its upgrade level.",
})

type SetWeaponInfusionResult struct {
	SaveSessionID  string          `json:"saveSessionID"`
	SaveRevision   string          `json:"saveRevision"`
	OwnedItemID    string          `json:"ownedItemID"`
	CharacterID    int             `json:"characterID"`
	Container      string          `json:"container"`
	PreviousGameID uint32          `json:"previousGameID"`
	GameID         uint32          `json:"gameID"`
	Affinity       schema.Affinity `json:"affinity"`
	UpgradeLevel   uint8           `json:"upgradeLevel"`
}

func SetWeaponInfusion(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	affinity schema.Affinity,
	expectedRevision string,
) (SetWeaponInfusionResult, error) {
	if engine == nil {
		return SetWeaponInfusionResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetWeaponInfusionResult{}, errors.New("game catalog is not available")
	}

	owned, err := engine.GetOwnedItem(saveSessionID, characterID, ownedItemID)
	if err != nil {
		return SetWeaponInfusionResult{}, err
	}
	gameIDs, err := engine.ResolveGaItemIDs(saveSessionID, characterID, []uint32{owned.GaItemHandle})
	if err != nil {
		return SetWeaponInfusionResult{}, err
	}
	currentGameID := gameIDs[0]
	targetGameID, upgradeLevel, err := gameCatalog.WeaponInfusionTarget(
		currentGameID, affinity)
	if err != nil {
		return SetWeaponInfusionResult{}, fmt.Errorf("owned item %q: %w", ownedItemID, err)
	}

	mutation, err := engine.SetWeaponInfusion(
		saveSessionID, characterID, ownedItemID, expectedRevision, currentGameID, targetGameID)
	if err != nil {
		return SetWeaponInfusionResult{}, err
	}
	return SetWeaponInfusionResult{
		SaveSessionID: mutation.SaveSessionID, SaveRevision: mutation.SaveRevision,
		OwnedItemID: mutation.OwnedItemID, CharacterID: mutation.CharacterID,
		Container: mutation.Container, PreviousGameID: mutation.PreviousGameID,
		GameID: mutation.GameID, Affinity: affinity, UpgradeLevel: upgradeLevel,
	}, nil
}
