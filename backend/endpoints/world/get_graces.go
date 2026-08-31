/*
Endpoint: GetGraces
EndpointID: get_graces
Purpose: Returns Sites of Grace and whether they have been visited.
How it works: The handler reads every grace resource GameCatalog declares, requires one visit event flag per resource and a unique flag across them, and resolves all of those flags in one bulk SaveEngine read. It decodes no flag itself.
Supported resource types: GraceDocument.
Input variables: saveSessionID, characterID.
GameCatalog variables read: resource kind and key plus the grace name, region label, boss-arena fact, dungeon type and visit event flag ID. The door event flag stays private catalog data.
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

// GetGracesEndpointID is the stable backend identifier of GetGraces.
const GetGracesEndpointID = "get_graces"

// GetGracesDefinition describes the public getter contract.
var GetGracesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetGraces",
	ID:                         GetGracesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GraceDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns Sites of Grace and whether they have been visited.",
})

// GraceEntry is one catalog-declared Site of Grace and its current state.
// RegionLabel is the curated presentation and grouping label of the grace, not a
// reference to a region resource. The visit event flag and the door event flag
// stay internal save-format details.
type GraceEntry struct {
	Kind        schema.ResourceKind `json:"kind"`
	Key         string              `json:"key"`
	Name        string              `json:"name"`
	RegionLabel string              `json:"regionLabel"`
	BossArena   bool                `json:"bossArena"`
	DungeonType string              `json:"dungeonType"`
	Visited     bool                `json:"visited"`
}

// GetGracesResult is the deterministic result for one character slot.
type GetGracesResult struct {
	SaveSessionID string       `json:"saveSessionID"`
	SaveRevision  string       `json:"saveRevision"`
	CharacterID   int          `json:"characterID"`
	Active        bool         `json:"active"`
	Graces        []GraceEntry `json:"graces"`
}

// GetGraces joins the catalog declarations with the save-side visit flags. An
// inactive or residual slot reports active false and every entry unvisited
// without its slot data being read. The whole catalog is validated before one
// save byte is touched, so incomplete or conflicting catalog data can never be
// answered with a state.
func GetGraces(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetGracesResult, error) {
	if engine == nil {
		return GetGracesResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetGracesResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogGraces(gameCatalog)
	if err != nil {
		return GetGracesResult{}, err
	}
	eventFlagIDs := make([]uint32, len(declared))
	for index, grace := range declared {
		eventFlagIDs[index] = grace.eventFlagID
	}
	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetGracesResult{}, err
	}

	result := GetGracesResult{
		SaveSessionID: flags.SaveSessionID,
		SaveRevision:  flags.SaveRevision,
		CharacterID:   flags.CharacterID,
		Active:        flags.Active,
		Graces:        make([]GraceEntry, 0, len(declared)),
	}
	for _, grace := range declared {
		entry := grace.entry
		entry.Visited = flags.Flags[grace.eventFlagID]
		result.Graces = append(result.Graces, entry)
	}
	return result, nil
}

// declaredGrace is the private projection both grace endpoints share.
// doorEventFlagID is the private catalog fact SetGraceVisited needs and no
// getter exposes; it is zero for a grace without a sealed dungeon entrance.
type declaredGrace struct {
	entry           GraceEntry
	eventFlagID     uint32
	doorEventFlagID uint32
}

// catalogGraces returns the declared graces ordered by region label, then name
// and then key. It fails closed on a resource whose grace document is missing
// and on two graces that claim the same visit flag, which no single document can
// rule out.
func catalogGraces(gameCatalog *gamecatalog.Catalog) ([]declaredGrace, error) {
	declared := make([]declaredGrace, 0)
	flagOwners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindGrace {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("grace %q: %w", summary.Key, err)
		}
		if resource.Grace == nil {
			return nil, fmt.Errorf("grace %q carries no grace document", summary.Key)
		}
		flag := resource.Grace.VisitEventFlagID.Value
		if owner, duplicate := flagOwners[flag]; duplicate {
			return nil, fmt.Errorf(
				"graces %q and %q both declare event flag %d", owner, summary.Key, flag)
		}
		flagOwners[flag] = summary.Key
		declared = append(declared, declaredGrace{
			entry: GraceEntry{
				Kind:        resource.Kind,
				Key:         resource.Key,
				Name:        resource.Grace.Name.Value,
				RegionLabel: resource.Grace.RegionLabel.Value,
				BossArena:   resource.Grace.BossArena.Value,
				DungeonType: resource.Grace.DungeonType.Value,
			},
			eventFlagID:     flag,
			doorEventFlagID: resource.Grace.DoorEventFlagID.Value,
		})
	}

	sort.SliceStable(declared, func(i, j int) bool {
		if declared[i].entry.RegionLabel != declared[j].entry.RegionLabel {
			return declared[i].entry.RegionLabel < declared[j].entry.RegionLabel
		}
		if declared[i].entry.Name != declared[j].entry.Name {
			return declared[i].entry.Name < declared[j].entry.Name
		}
		return declared[i].entry.Key < declared[j].entry.Key
	})
	return declared, nil
}
