/*
Endpoint: SetWeaponAshOfWar
EndpointID: set_weapon_ash_of_war
Purpose: Mounts, changes, or removes an Ash of War after validating all relations and effects on related instances.
How it works: The handler resolves the owned weapon, validates its custom-Ash capability and the exact compatible Ash resource, then asks SaveEngine to attach the first existing free copy or to write the canonical no-custom sentinel.
Supported resource types: ItemDocument: Weapon; mounting additionally requires known custom-Ash capability and a compatible AshOfWar already represented by a free GaItem copy.
Input variables: saveSessionID, characterID, weaponOwnedItemID, ashOfWarKind, ashOfWarKey, expectedRevision.
GameCatalog variables read: the current weapon identity, item.family, item.gameID, item.capabilities.ashOfWarMount, item.ashOfWar.compatibilityMask and the compatible_with_aow relation.
Save variables processed: one owned common weapon record, its unique GaItem record and the existing AoW GaItem references; success changes only the weapon's four-byte AoW handle field.
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

// SetWeaponAshOfWarEndpointID is the stable backend identifier of SetWeaponAshOfWar.
const SetWeaponAshOfWarEndpointID = "set_weapon_ash_of_war"

// SetWeaponAshOfWarDefinition describes the public mutation contract.
var SetWeaponAshOfWarDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetWeaponAshOfWar",
	ID:                         SetWeaponAshOfWarEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Weapon; mounting requires custom Ash capability and a compatible existing AshOfWar copy",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "weaponOwnedItemID", "ashOfWarKind", "ashOfWarKey", "expectedRevision"},
	Description:                "Mounts, changes, or removes an Ash of War by changing one validated weapon reference without allocating or repacking GaItems.",
})

// SetWeaponAshOfWarResult is the SaveEngine receipt. Zero in either Ash of War
// game-ID field means that side of the committed change had no custom Ash.
type SetWeaponAshOfWarResult = saveengine.SetWeaponAshOfWarResult

// SetWeaponAshOfWar mounts one compatible Ash of War or removes the custom Ash
// when both resource selectors are nil. Exactly one nil selector is invalid.
func SetWeaponAshOfWar(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	weaponOwnedItemID string,
	ashOfWarKind *string,
	ashOfWarKey *string,
	expectedRevision string,
) (SetWeaponAshOfWarResult, error) {
	if engine == nil {
		return SetWeaponAshOfWarResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetWeaponAshOfWarResult{}, errors.New("game catalog is not available")
	}
	if (ashOfWarKind == nil) != (ashOfWarKey == nil) {
		return SetWeaponAshOfWarResult{}, errors.New(
			"ashOfWarKind and ashOfWarKey must either both identify a resource or both be null")
	}

	owned, err := engine.GetOwnedItem(saveSessionID, characterID, weaponOwnedItemID)
	if err != nil {
		return SetWeaponAshOfWarResult{}, err
	}
	gameIDs, err := engine.ResolveGaItemIDs(
		saveSessionID, characterID, []uint32{owned.GaItemHandle})
	if err != nil {
		return SetWeaponAshOfWarResult{}, err
	}
	weaponGameID := gameIDs[0]
	weaponView, found := gameCatalog.ItemViewByGameID(weaponGameID)
	if !found || weaponView.Resource.Item == nil {
		return SetWeaponAshOfWarResult{}, fmt.Errorf(
			"owned weapon game ID 0x%08X is not found in game catalog", weaponGameID)
	}
	weapon := weaponView.Resource.Item
	if !weapon.Family.Known || weapon.Family.Value != schema.ItemFamilyWeapon {
		return SetWeaponAshOfWarResult{}, fmt.Errorf(
			"owned item %q resolves to family %q, want %q",
			weaponOwnedItemID, weapon.Family.Value, schema.ItemFamilyWeapon)
	}
	targetAshOfWarGameID := uint32(0)
	if ashOfWarKind != nil {
		mount := weapon.Capabilities.AshOfWarMount
		if !mount.Known {
			return SetWeaponAshOfWarResult{}, fmt.Errorf(
				"weapon kind %q key %q has an unknown Ash of War mount capability",
				weaponView.Resource.Kind, weaponView.Resource.Key)
		}
		if !mount.Enabled {
			return SetWeaponAshOfWarResult{}, fmt.Errorf(
				"weapon kind %q key %q cannot mount a custom Ash of War",
				weaponView.Resource.Kind, weaponView.Resource.Key)
		}
		if mount.Rules == nil || mount.Rules.Mode != schema.AshOfWarMountModeCustom {
			return SetWeaponAshOfWarResult{}, fmt.Errorf(
				"weapon kind %q key %q has no supported custom Ash of War mount rules",
				weaponView.Resource.Kind, weaponView.Resource.Key)
		}
		ashResource, err := gameCatalog.ResourceByKindAndKey(
			schema.ResourceKind(*ashOfWarKind), *ashOfWarKey)
		if err != nil {
			return SetWeaponAshOfWarResult{}, err
		}
		if ashResource.Item == nil || !ashResource.Item.Family.Known ||
			ashResource.Item.Family.Value != schema.ItemFamilyAshOfWar ||
			ashResource.Item.AshOfWar == nil {
			return SetWeaponAshOfWarResult{}, fmt.Errorf(
				"resource kind %q key %q is not a confirmed Ash of War",
				*ashOfWarKind, *ashOfWarKey)
		}
		if !ashResource.Item.GameID.Known ||
			ashResource.Item.GameID.Value&0xF0000000 != 0x80000000 {
			return SetWeaponAshOfWarResult{}, fmt.Errorf(
				"Ash of War kind %q key %q has no valid game ID",
				*ashOfWarKind, *ashOfWarKey)
		}
		if !ashResource.Item.AshOfWar.CompatibilityMask.Known {
			return SetWeaponAshOfWarResult{}, fmt.Errorf(
				"Ash of War kind %q key %q has unknown weapon compatibility",
				*ashOfWarKind, *ashOfWarKey)
		}
		compatible := false
		for _, relation := range weaponView.OutgoingRelations {
			if relation.Kind == schema.RelationCompatibleWithAshOfWar &&
				relation.To == ashResource.Ref() {
				compatible = true
				break
			}
		}
		if !compatible {
			return SetWeaponAshOfWarResult{}, fmt.Errorf(
				"Ash of War kind %q key %q is not compatible with weapon kind %q key %q",
				*ashOfWarKind, *ashOfWarKey,
				weaponView.Resource.Kind, weaponView.Resource.Key)
		}
		targetAshOfWarGameID = ashResource.Item.GameID.Value
	}

	return engine.SetWeaponAshOfWar(
		saveSessionID, characterID, weaponOwnedItemID, expectedRevision,
		weaponGameID, targetAshOfWarGameID)
}
