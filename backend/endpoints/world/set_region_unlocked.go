/*
Endpoint: SetRegionUnlocked
EndpointID: set_region_unlocked
Purpose: Sets the unlock state of a region.
How it works: The runtime handler validates the complete request and expected revision, resolves the region through the shared catalog resolver, passes the private RegionID to SaveEngine under expectedRevision control, and returns the public kind and key receipt.
Supported resource types: RegionDocument
Input variables: saveSessionID, characterID, regionKind, regionKey, unlocked, expectedRevision
GameCatalog variables read: resource kind and key plus region ID of every region.
Save variables processed: the UnlockedRegions list of the requested character; SaveEngine rebuilds the slot, validates expectedRevision and finishes with full success or rollback.
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

// SetRegionUnlockedEndpointID is the stable backend identifier of SetRegionUnlocked.
const SetRegionUnlockedEndpointID = "set_region_unlocked"

// SetRegionUnlockedDefinition describes the public mutation contract.
var SetRegionUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetRegionUnlocked",
	ID:                         SetRegionUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "RegionDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "regionKind", "regionKey", "unlocked", "expectedRevision"},
	Description:                "Sets the unlock state of a region.",
})

// SetRegionUnlockedResult reports the committed state in public catalog terms.
type SetRegionUnlockedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	RegionKind    schema.ResourceKind `json:"regionKind"`
	RegionKey     string              `json:"regionKey"`
	Unlocked      bool                `json:"unlocked"`
}

// SetRegionUnlocked unlocks or locks one curated region in a character slot of
// an existing save session.
func SetRegionUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	regionKind string,
	regionKey string,
	unlocked bool,
	expectedRevision string,
) (SetRegionUnlockedResult, error) {
	if engine == nil {
		return SetRegionUnlockedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetRegionUnlockedResult{}, errors.New("game catalog is not available")
	}
	if regionKind != string(schema.ResourceKindRegion) {
		return SetRegionUnlockedResult{}, fmt.Errorf(
			"resource kind %q is not %q", regionKind, schema.ResourceKindRegion)
	}

	declared, err := catalogRegions(gameCatalog)
	if err != nil {
		return SetRegionUnlockedResult{}, err
	}
	var matched declaredRegion
	found := false
	for _, region := range declared {
		if region.entry.Key == regionKey {
			matched = region
			found = true
			break
		}
	}
	if !found {
		return SetRegionUnlockedResult{}, fmt.Errorf(
			"unknown resource key %q in kind %q", regionKey, regionKind)
	}

	mutation, err := engine.SetRegionUnlocked(
		saveSessionID,
		characterID,
		matched.regionID,
		unlocked,
		expectedRevision,
	)
	if err != nil {
		return SetRegionUnlockedResult{}, err
	}
	return SetRegionUnlockedResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		RegionKind:    matched.entry.Kind,
		RegionKey:     matched.entry.Key,
		Unlocked:      mutation.Unlocked,
	}, nil
}
