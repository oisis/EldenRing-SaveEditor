/*
Endpoint: GetRegions
EndpointID: get_regions
Purpose: Returns regions and their unlock state.
How it works: The handler reads every region resource GameCatalog declares, resolves the raw UnlockedRegions list once through SaveEngine and joins it by exact region ID membership. Unknown raw IDs remain private.
Supported resource types: RegionDocument.
Input variables: saveSessionID, characterID.
GameCatalog variables read: resource kind and key plus region ID, name and area.
Save variables read: the character activity flag and the raw UnlockedRegions list; the getter writes nothing.
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

// GetRegionsEndpointID is the stable backend identifier of GetRegions.
const GetRegionsEndpointID = "get_regions"

// GetRegionsDefinition describes the public getter contract.
var GetRegionsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetRegions",
	ID:                         GetRegionsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "RegionDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns regions and their unlock state.",
})

// RegionEntry is one curated catalog region and its state in the character's
// UnlockedRegions list. The raw region ID stays an internal matching detail.
type RegionEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	Name     string              `json:"name"`
	Area     string              `json:"area"`
	Unlocked bool                `json:"unlocked"`
}

// GetRegionsResult is the deterministic region view of one character slot.
type GetRegionsResult struct {
	SaveSessionID string        `json:"saveSessionID"`
	CharacterID   int           `json:"characterID"`
	Active        bool          `json:"active"`
	Regions       []RegionEntry `json:"regions"`
}

// GetRegions joins the curated region catalog with the raw save-side list.
// Raw IDs absent from GameCatalog are deliberately ignored, not rejected.
func GetRegions(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetRegionsResult, error) {
	if engine == nil {
		return GetRegionsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetRegionsResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogRegions(gameCatalog)
	if err != nil {
		return GetRegionsResult{}, err
	}
	stored, err := engine.GetRegions(saveSessionID, characterID)
	if err != nil {
		return GetRegionsResult{}, err
	}
	unlocked := make(map[uint32]struct{}, len(stored.RegionIDs))
	for _, regionID := range stored.RegionIDs {
		unlocked[regionID] = struct{}{}
	}

	result := GetRegionsResult{
		SaveSessionID: stored.SaveSessionID,
		CharacterID:   stored.CharacterID,
		Active:        stored.Active,
		Regions:       make([]RegionEntry, 0, len(declared)),
	}
	for _, region := range declared {
		entry := region.entry
		_, entry.Unlocked = unlocked[region.regionID]
		result.Regions = append(result.Regions, entry)
	}
	return result, nil
}

type declaredRegion struct {
	entry    RegionEntry
	regionID uint32
}

func catalogRegions(gameCatalog *gamecatalog.Catalog) ([]declaredRegion, error) {
	declared := make([]declaredRegion, 0)
	idOwners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindRegion {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("region %q: %w", summary.Key, err)
		}
		if resource.Region == nil {
			return nil, fmt.Errorf("region %q carries no region document", summary.Key)
		}
		regionID := resource.Region.RegionID.Value
		if owner, duplicate := idOwners[regionID]; duplicate {
			return nil, fmt.Errorf(
				"regions %q and %q both declare region ID %d", owner, summary.Key, regionID)
		}
		idOwners[regionID] = summary.Key
		declared = append(declared, declaredRegion{
			entry: RegionEntry{
				Kind: resource.Kind,
				Key:  resource.Key,
				Name: resource.Region.Name.Value,
				Area: resource.Region.Area.Value,
			},
			regionID: regionID,
		})
	}

	sort.SliceStable(declared, func(i, j int) bool {
		if declared[i].entry.Area != declared[j].entry.Area {
			return declared[i].entry.Area < declared[j].entry.Area
		}
		if declared[i].entry.Name != declared[j].entry.Name {
			return declared[i].entry.Name < declared[j].entry.Name
		}
		return declared[i].entry.Key < declared[j].entry.Key
	})
	return declared, nil
}
