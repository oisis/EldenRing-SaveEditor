/*
Endpoint: SetMapRegionRevealed
EndpointID: set_map_region_revealed
Purpose: Sets the visibility of the specified map region without providing general access to raw flags.
How it works: The handler validates the whole curated map region table through the shared resolver, indexes the Map Fragment items by the visibility flag their single item.unlocks entry of kind "map" declares, resolves the requested resource by its exact kind and key, and delegates one atomic event-flag-and-Inventory mutation to SaveEngine under expectedRevision control.
Supported resource types: MapRegionDocument, plus the goods ItemDocument that declares the matching map unlock when the region has a Map Fragment.
Input variables: saveSessionID, characterID, mapRegionKind, mapRegionKey, revealed, expectedRevision.
GameCatalog variables read: resource kind and key plus mapRegion.visibleEventFlagID of every map region, and item.gameID, item.family and item.unlocks of kind "map" of every item.
Save variables processed: the visibility event flag bit of the requested region and, for a region with a Map Fragment, the one InventoryHeld record of that fragment; SaveEngine validates expectedRevision and finishes with full success or rollback.
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

// SetMapRegionRevealedEndpointID is the stable backend identifier of SetMapRegionRevealed.
const SetMapRegionRevealedEndpointID = "set_map_region_revealed"

// SetMapRegionRevealedDefinition describes the public mutation contract.
var SetMapRegionRevealedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetMapRegionRevealed",
	ID:                         SetMapRegionRevealedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "MapRegionDocument plus goods ItemDocument declaring the matching map unlock",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "mapRegionKind", "mapRegionKey", "revealed", "expectedRevision"},
	Description:                "Sets the visibility of the specified map region without providing general access to raw flags.",
})

// mapFragmentUnlockKind is the item.unlocks kind that pairs a goods item with
// the visibility flag of a map region. It is the only declared relation between
// the two vocabularies and therefore the only selector: no name, no area label,
// no ordering and no arithmetic on the flag identifier decides which item
// belongs to which region.
const mapFragmentUnlockKind = "map"

// SetMapRegionRevealedResult reports the committed state in public catalog
// terms. SaveEngine supplies the session state; this endpoint adds the catalog
// identity it resolved without exposing any event flag or item game ID.
type SetMapRegionRevealedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	MapRegionKind schema.ResourceKind `json:"mapRegionKind"`
	MapRegionKey  string              `json:"mapRegionKey"`
	Revealed      bool                `json:"revealed"`
}

// SetMapRegionRevealed reveals or hides one catalog map region in a character
// slot of an existing save session, reproducing the semantics of SaveForge 1.5.8
// and 1.6.8: the visibility flag of the region is written, and a region that has
// a Map Fragment gains or loses that one Inventory item with it.
//
// The transient acquired flag of block 63, the system map display flags, the
// unsafe sub-region flags and the Fog of War bitfield are all outside this
// contract and are never written. Storage is never read or written either: the
// fragment lives in InventoryHeld, exactly where the legacy path put it.
func SetMapRegionRevealed(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	mapRegionKind string,
	mapRegionKey string,
	revealed bool,
	expectedRevision string,
) (SetMapRegionRevealedResult, error) {
	if engine == nil {
		return SetMapRegionRevealedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetMapRegionRevealedResult{}, errors.New("game catalog is not available")
	}
	if mapRegionKind != string(schema.ResourceKindMapRegion) {
		return SetMapRegionRevealedResult{}, fmt.Errorf(
			"resource kind %q is not %q", mapRegionKind, schema.ResourceKindMapRegion)
	}

	// The shared resolver validates the whole curated table — a missing document
	// or two regions claiming one flag — before a save byte is touched. It covers
	// every map region resource, so a key it does not carry is unknown.
	declared, err := catalogMapRegions(gameCatalog)
	if err != nil {
		return SetMapRegionRevealedResult{}, err
	}
	var matched declaredMapRegion
	found := false
	for _, region := range declared {
		if region.entry.Key == mapRegionKey {
			matched = region
			found = true
			break
		}
	}
	if !found {
		return SetMapRegionRevealedResult{}, fmt.Errorf(
			"unknown resource key %q in kind %q", mapRegionKey, mapRegionKind)
	}

	fragments, err := catalogMapFragments(gameCatalog, declared)
	if err != nil {
		return SetMapRegionRevealedResult{}, err
	}

	mutation, err := engine.SetMapRegionRevealed(
		saveSessionID,
		characterID,
		matched.eventFlagID,
		fragments[matched.eventFlagID],
		revealed,
		expectedRevision,
	)
	if err != nil {
		return SetMapRegionRevealedResult{}, err
	}
	return SetMapRegionRevealedResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		MapRegionKind: matched.entry.Kind,
		MapRegionKey:  matched.entry.Key,
		Revealed:      mutation.Revealed,
	}, nil
}

// catalogMapFragments indexes the Map Fragment items by the map region
// visibility flag their unlock declares. A region without an entry has no
// fragment, which is the normal case: the curated table covers the dungeon maps
// too, and only the overworld regions were ever paired with an item.
//
// The whole index is built and validated even when the requested region has no
// fragment, because a catalog in which two items claim one region, or in which
// an item points at a flag no region declares, cannot be answered with a state.
func catalogMapFragments(
	gameCatalog *gamecatalog.Catalog,
	declared []declaredMapRegion,
) (map[uint32]uint32, error) {
	regionOwners := make(map[uint32]string, len(declared))
	for _, region := range declared {
		regionOwners[region.eventFlagID] = region.entry.Key
	}

	fragments := make(map[uint32]uint32)
	owners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindItem {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("item %q: %w", summary.Key, err)
		}
		if resource.Item == nil {
			continue
		}
		flag, declares, err := mapFragmentFlagOf(resource)
		if err != nil {
			return nil, err
		}
		if !declares {
			continue
		}
		if _, exists := regionOwners[flag]; !exists {
			return nil, fmt.Errorf(
				"map fragment %q declares event flag %d, which no map region declares",
				summary.Key, flag)
		}
		if owner, taken := owners[flag]; taken {
			return nil, fmt.Errorf(
				"map fragments %q and %q both declare event flag %d", owner, summary.Key, flag)
		}
		owners[flag] = summary.Key
		fragments[flag] = resource.Item.GameID.Value
	}
	return fragments, nil
}

// mapFragmentFlagOf applies the one Map Fragment definition rule. A resource
// without a map unlock is not a fragment; once it declares one, every required
// field is fail-closed and a second unlock of that kind is a conflict rather
// than a choice.
func mapFragmentFlagOf(resource schema.Resource) (uint32, bool, error) {
	flag := uint32(0)
	found := 0
	for _, unlock := range resource.Item.Unlocks {
		if !unlock.Kind.Known || unlock.Kind.Value != mapFragmentUnlockKind {
			continue
		}
		if found == 0 {
			if !unlock.EventFlagID.Known {
				return 0, false, fmt.Errorf(
					"map fragment %q declares a map unlock with an unknown event flag", resource.Key)
			}
			flag = unlock.EventFlagID.Value
		}
		found++
	}
	if found == 0 {
		return 0, false, nil
	}
	if found > 1 {
		return 0, false, fmt.Errorf(
			"map fragment %q declares %d map unlocks, want exactly 1", resource.Key, found)
	}
	if resource.Item.Family.Value != schema.ItemFamilyGoods || !resource.Item.Family.Known {
		return 0, false, fmt.Errorf(
			"map fragment %q has item family %q, want %q",
			resource.Key, resource.Item.Family.Value, schema.ItemFamilyGoods)
	}
	if !resource.Item.GameID.Known || resource.Item.GameID.Value == 0 {
		return 0, false, fmt.Errorf("map fragment %q has no known game ID", resource.Key)
	}
	return flag, true, nil
}
