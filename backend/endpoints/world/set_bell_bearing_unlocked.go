/*
Endpoint: SetBellBearingUnlocked
EndpointID: set_bell_bearing_unlocked
Purpose: Sets the handed-in state of a Bell Bearing.
How it works: The handler resolves one catalog Bell Bearing, validates its goods game ID and acquisition flag, and delegates one atomic flag-and-container mutation to SaveEngine.
Supported resource types: ItemDocument: BellBearing.
Input variables: saveSessionID, characterID, bellBearingKind, bellBearingKey, unlocked, expectedRevision.
GameCatalog variables read: item.gameID, item.family and the bell_bearing unlock eventFlagID, name and category.
Save variables processed: the acquisition event flag and, when unlocking, matching raw or goods-handle records in Inventory common/key and Storage common; the operation finishes with full success or rollback.
Implementation status: implemented
*/
package world

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetBellBearingUnlockedEndpointID is the stable backend identifier of SetBellBearingUnlocked.
const SetBellBearingUnlockedEndpointID = "set_bell_bearing_unlocked"

// SetBellBearingUnlockedDefinition describes the public mutation contract.
var SetBellBearingUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetBellBearingUnlocked",
	ID:                         SetBellBearingUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: BellBearing",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "bellBearingKind", "bellBearingKey", "unlocked", "expectedRevision"},
	Description:                "Sets the handed-in state of a Bell Bearing.",
})

// SetBellBearingUnlockedResult reports the committed state in public catalog
// terms without exposing the internal game ID, handle or event flag.
type SetBellBearingUnlockedResult struct {
	SaveSessionID   string              `json:"saveSessionID"`
	SaveRevision    string              `json:"saveRevision"`
	CharacterID     int                 `json:"characterID"`
	BellBearingKind schema.ResourceKind `json:"bellBearingKind"`
	BellBearingKey  string              `json:"bellBearingKey"`
	Unlocked        bool                `json:"unlocked"`
}

// SetBellBearingUnlocked sets or clears the handed-in state of one catalog Bell
// Bearing. Unlocking also consumes every matching physical copy from the
// confirmed Inventory and Storage sections; locking never creates an item.
func SetBellBearingUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	bellBearingKind string,
	bellBearingKey string,
	unlocked bool,
	expectedRevision string,
) (SetBellBearingUnlockedResult, error) {
	if engine == nil {
		return SetBellBearingUnlockedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetBellBearingUnlockedResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogBellBearings(gameCatalog)
	if err != nil {
		return SetBellBearingUnlockedResult{}, err
	}
	var matched declaredBellBearing
	found := false
	for _, bellBearing := range declared {
		if bellBearing.entry.Kind == schema.ResourceKind(bellBearingKind) &&
			bellBearing.entry.Key == bellBearingKey {
			matched = bellBearing
			found = true
			break
		}
	}
	if !found {
		resource, err := gameCatalog.ResourceByKindAndKey(
			schema.ResourceKind(bellBearingKind), bellBearingKey)
		if err != nil {
			return SetBellBearingUnlockedResult{}, err
		}
		if resource.Item == nil {
			return SetBellBearingUnlockedResult{}, fmt.Errorf(
				"resource kind %q key %q has no item document", bellBearingKind, bellBearingKey)
		}
		matched, found, err = declaredBellBearingFromResource(resource)
		if err != nil {
			return SetBellBearingUnlockedResult{}, err
		}
	}
	if !found {
		return SetBellBearingUnlockedResult{}, fmt.Errorf(
			"resource kind %q key %q declares no bell_bearing unlock",
			bellBearingKind, bellBearingKey)
	}
	resource, err := gameCatalog.ResourceByKindAndKey(
		matched.entry.Kind, matched.entry.Key)
	if err != nil {
		return SetBellBearingUnlockedResult{}, err
	}
	if !resource.Item.GameID.Known {
		return SetBellBearingUnlockedResult{}, fmt.Errorf(
			"bell bearing %q has no known game ID", bellBearingKey)
	}
	gameID := resource.Item.GameID.Value
	if gameID&0xF0000000 != 0x40000000 {
		return SetBellBearingUnlockedResult{}, fmt.Errorf(
			"bell bearing %q has game ID 0x%08X outside the goods family",
			bellBearingKey, gameID)
	}

	mutation, err := engine.SetBellBearingUnlocked(
		saveSessionID,
		characterID,
		matched.eventFlagID,
		gameID,
		unlocked,
		expectedRevision,
	)
	if err != nil {
		return SetBellBearingUnlockedResult{}, err
	}
	return SetBellBearingUnlockedResult{
		SaveSessionID:   mutation.SaveSessionID,
		SaveRevision:    mutation.SaveRevision,
		CharacterID:     mutation.CharacterID,
		BellBearingKind: matched.entry.Kind,
		BellBearingKey:  matched.entry.Key,
		Unlocked:        mutation.Unlocked,
	}, nil
}
