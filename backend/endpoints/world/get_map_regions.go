/*
Endpoint: GetMapRegions
EndpointID: get_map_regions
Purpose: Returns map regions and whether their map is visible.
How it works: The handler reads every map region resource GameCatalog declares, requires one visibility event flag per resource and a unique flag across them, and resolves all of those flags in one bulk SaveEngine read. It decodes no flag itself.
Supported resource types: MapRegionDocument.
Input variables: saveSessionID, characterID.
GameCatalog variables read: resource kind and key plus the map region name, area label and visibility event flag ID.
Save variables read: the character activity flag and the requested event flag bits; the getter writes nothing.
Implementation status: implemented
*/
package world

import (
	"errors"
	"fmt"
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetMapRegionsEndpointID is the stable backend identifier of GetMapRegions.
const GetMapRegionsEndpointID = "get_map_regions"

// GetMapRegionsDefinition describes the public getter contract. The old contract
// declared parentRegionKind and parentRegionKey, which the curated map
// visibility table does not support: it declares a plain text area and no region
// resource identity at all. Publishing that pair would promise a relation no
// source declares, so the input is exactly the session and the character slot.
var GetMapRegionsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetMapRegions",
	ID:                         GetMapRegionsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "MapRegionDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns map regions and whether their map is visible.",
})

// MapRegionEntry is one catalog-declared map region and its current visibility.
// AreaLabel is the curated presentation and grouping label of the region, not a
// reference to a region resource. The visibility event flag stays an internal
// save-format detail.
type MapRegionEntry struct {
	Kind      schema.ResourceKind `json:"kind"`
	Key       string              `json:"key"`
	Name      string              `json:"name"`
	AreaLabel string              `json:"areaLabel"`
	Visible   bool                `json:"visible"`
}

// GetMapRegionsResult is the deterministic result for one character slot.
type GetMapRegionsResult struct {
	SaveSessionID string           `json:"saveSessionID"`
	CharacterID   int              `json:"characterID"`
	Active        bool             `json:"active"`
	MapRegions    []MapRegionEntry `json:"mapRegions"`
}

// GetMapRegions joins the catalog declarations with the save-side visibility
// flags. An inactive or residual slot reports active false and every entry
// invisible without its slot data being read. The whole catalog is validated
// before one save byte is touched, so incomplete or conflicting catalog data can
// never be answered with a state.
func GetMapRegions(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetMapRegionsResult, error) {
	if engine == nil {
		return GetMapRegionsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetMapRegionsResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogMapRegions(gameCatalog)
	if err != nil {
		return GetMapRegionsResult{}, err
	}
	eventFlagIDs := make([]uint32, len(declared))
	for index, region := range declared {
		eventFlagIDs[index] = region.eventFlagID
	}
	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetMapRegionsResult{}, err
	}

	result := GetMapRegionsResult{
		SaveSessionID: flags.SaveSessionID,
		CharacterID:   flags.CharacterID,
		Active:        flags.Active,
		MapRegions:    make([]MapRegionEntry, 0, len(declared)),
	}
	for _, region := range declared {
		entry := region.entry
		entry.Visible = flags.Flags[region.eventFlagID]
		result.MapRegions = append(result.MapRegions, entry)
	}
	return result, nil
}

type declaredMapRegion struct {
	entry       MapRegionEntry
	eventFlagID uint32
}

// catalogMapRegions returns the declared map regions ordered by area label, then
// name and then key. It fails closed on a resource whose map region document is
// missing and on two regions that claim the same visibility flag, which no single
// document can rule out.
func catalogMapRegions(gameCatalog *gamecatalog.Catalog) ([]declaredMapRegion, error) {
	declared := make([]declaredMapRegion, 0)
	flagOwners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindMapRegion {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("map region %q: %w", summary.Key, err)
		}
		if resource.MapRegion == nil {
			return nil, fmt.Errorf("map region %q carries no map region document", summary.Key)
		}
		flag := resource.MapRegion.VisibleEventFlagID.Value
		if owner, duplicate := flagOwners[flag]; duplicate {
			return nil, fmt.Errorf(
				"map regions %q and %q both declare event flag %d", owner, summary.Key, flag)
		}
		flagOwners[flag] = summary.Key
		declared = append(declared, declaredMapRegion{
			entry: MapRegionEntry{
				Kind:      resource.Kind,
				Key:       resource.Key,
				Name:      resource.MapRegion.Name.Value,
				AreaLabel: resource.MapRegion.AreaLabel.Value,
			},
			eventFlagID: flag,
		})
	}

	sort.SliceStable(declared, func(i, j int) bool {
		if declared[i].entry.AreaLabel != declared[j].entry.AreaLabel {
			return declared[i].entry.AreaLabel < declared[j].entry.AreaLabel
		}
		if declared[i].entry.Name != declared[j].entry.Name {
			return declared[i].entry.Name < declared[j].entry.Name
		}
		return declared[i].entry.Key < declared[j].entry.Key
	})
	return declared, nil
}
