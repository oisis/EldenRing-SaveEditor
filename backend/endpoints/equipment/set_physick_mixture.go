/*
Endpoint: SetPhysickMixture
EndpointID: set_physick_mixture
Purpose: Atomically sets both entries of the Flask of Wondrous Physick mixture.
How it works: The runtime handler resolves exactly two nullable catalog resource references, validates every non-empty resource as a goods item with a confirmed Physick equipment capability, and delegates one atomic two-field mutation to SaveEngine under expectedRevision control.
Supported resource types: ItemDocument of family goods with capability equipment allowing slot physick.
Input variables: saveSessionID, characterID, crystalTearResources, expectedRevision.
GameCatalog variables read: item.family, item.gameID and item.capabilities.equipment.
Save variables processed: the two active Crystal Tear identifiers in EquipPhysicsData and the Inventory common/key records needed to prove ownership of the selected tears and the Flask of Wondrous Physick; SaveEngine validates the complete plan and finishes with full success or rollback.
Implementation status: implemented
*/
package equipment

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetPhysickMixtureEndpointID is the stable backend identifier of SetPhysickMixture.
const (
	SetPhysickMixtureEndpointID = "set_physick_mixture"
	physickGameIDTypeMask       = uint32(0xF0000000)
	physickGoodsGameIDPrefix    = uint32(0x40000000)
)

// SetPhysickMixtureDefinition describes the public mutation contract.
var SetPhysickMixtureDefinition = contract.MustDefine(contract.Definition{
	Name:                   "SetPhysickMixture",
	ID:                     SetPhysickMixtureEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "ItemDocument of family goods with capability equipment allowing slot physick",
	SupportedResourceVariables: []string{
		"saveSessionID", "characterID", "crystalTearResources", "expectedRevision",
	},
	Description: "Atomically sets both entries of the Flask of Wondrous Physick mixture.",
})

// SetPhysickMixtureResult reports the committed mixture in public catalog
// terms. A nil entry is one empty Physick position.
type SetPhysickMixtureResult struct {
	SaveSessionID        string                 `json:"saveSessionID"`
	SaveRevision         string                 `json:"saveRevision"`
	CharacterID          int                    `json:"characterID"`
	CrystalTearResources [2]*schema.ResourceRef `json:"crystalTearResources"`
}

// SetPhysickMixture replaces both Physick positions of one active character.
// crystalTearResources must contain exactly two entries; nil clears that exact
// position, without left-packing the other one.
func SetPhysickMixture(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	crystalTearResources []*schema.ResourceRef,
	expectedRevision string,
) (SetPhysickMixtureResult, error) {
	if engine == nil {
		return SetPhysickMixtureResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetPhysickMixtureResult{}, errors.New("game catalog is not available")
	}
	if len(crystalTearResources) != 2 {
		return SetPhysickMixtureResult{}, fmt.Errorf(
			"crystalTearResources must contain exactly 2 positions; got %d",
			len(crystalTearResources))
	}

	tears := [2]uint32{saveengine.PhysickEmptyTearID, saveengine.PhysickEmptyTearID}
	var resolved [2]*schema.ResourceRef
	for index, declared := range crystalTearResources {
		if declared == nil {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(declared.Kind, declared.Key)
		if err != nil {
			return SetPhysickMixtureResult{}, fmt.Errorf(
				"crystalTearResources[%d]: %w", index, err)
		}
		gameID, err := physickTearGameID(resource)
		if err != nil {
			return SetPhysickMixtureResult{}, fmt.Errorf(
				"crystalTearResources[%d]: %w", index, err)
		}
		tears[index] = gameID
		ref := resource.Ref()
		resolved[index] = &ref
	}

	mutation, err := engine.SetPhysickMixture(
		saveSessionID, characterID, tears, expectedRevision)
	if err != nil {
		return SetPhysickMixtureResult{}, err
	}
	return SetPhysickMixtureResult{
		SaveSessionID:        mutation.SaveSessionID,
		SaveRevision:         mutation.SaveRevision,
		CharacterID:          mutation.CharacterID,
		CrystalTearResources: resolved,
	}, nil
}

func physickTearGameID(resource schema.Resource) (uint32, error) {
	if resource.Item == nil {
		return 0, fmt.Errorf(
			"resource kind %q key %q has no item document", resource.Kind, resource.Key)
	}
	item := resource.Item
	if !item.Family.Known {
		return 0, fmt.Errorf(
			"resource kind %q key %q has no known item family", resource.Kind, resource.Key)
	}
	if item.Family.Value != schema.ItemFamilyGoods {
		return 0, fmt.Errorf(
			"resource kind %q key %q has item family %q, want %q",
			resource.Kind, resource.Key, item.Family.Value, schema.ItemFamilyGoods)
	}
	if !item.GameID.Known {
		return 0, fmt.Errorf(
			"resource kind %q key %q has no known game ID", resource.Kind, resource.Key)
	}
	if item.GameID.Value == saveengine.PhysickEmptyTearID ||
		item.GameID.Value&physickGameIDTypeMask != physickGoodsGameIDPrefix {
		return 0, fmt.Errorf(
			"resource kind %q key %q has unsupported Physick game ID 0x%08X",
			resource.Kind, resource.Key, item.GameID.Value)
	}

	equipment := item.Capabilities.Equipment
	if !equipment.Known || !equipment.Enabled || equipment.Rules == nil {
		return 0, fmt.Errorf(
			"resource kind %q key %q has no confirmed Physick equipment capability",
			resource.Kind, resource.Key)
	}
	for _, slot := range equipment.Rules.AllowedSlots {
		if slot == schema.EquipmentSlotPhysick {
			return item.GameID.Value, nil
		}
	}
	return 0, fmt.Errorf(
		"resource kind %q key %q cannot be equipped in the Physick slot",
		resource.Kind, resource.Key)
}
