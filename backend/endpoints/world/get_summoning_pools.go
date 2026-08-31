/*
Endpoint: GetSummoningPools
EndpointID: get_summoning_pools
Purpose: Returns Summoning Pools and their activation state.
How it works: The handler reads every summoning pool resource GameCatalog declares, requires one activation event flag per resource and a unique flag across them, and resolves all of those flags in one bulk SaveEngine read. It decodes no flag itself.
Supported resource types: SummoningPoolDocument.
Input variables: saveSessionID, characterID.
GameCatalog variables read: resource kind and key plus the pool name, region label and activation event flag ID.
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

// GetSummoningPoolsEndpointID is the stable backend identifier of GetSummoningPools.
const GetSummoningPoolsEndpointID = "get_summoning_pools"

// GetSummoningPoolsDefinition describes the public getter contract.
var GetSummoningPoolsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetSummoningPools",
	ID:                         GetSummoningPoolsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "SummoningPoolDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns Summoning Pools and their activation state.",
})

// SummoningPoolEntry is one catalog-declared summoning pool and its current
// state. RegionLabel is the curated presentation and grouping label of the pool,
// not a reference to a region resource. The activation event flag stays an
// internal save-format detail.
type SummoningPoolEntry struct {
	Kind        schema.ResourceKind `json:"kind"`
	Key         string              `json:"key"`
	Name        string              `json:"name"`
	RegionLabel string              `json:"regionLabel"`
	Activated   bool                `json:"activated"`
}

// GetSummoningPoolsResult is the deterministic result for one character slot.
type GetSummoningPoolsResult struct {
	SaveSessionID  string               `json:"saveSessionID"`
	SaveRevision   string               `json:"saveRevision"`
	CharacterID    int                  `json:"characterID"`
	Active         bool                 `json:"active"`
	SummoningPools []SummoningPoolEntry `json:"summoningPools"`
}

// GetSummoningPools joins the catalog declarations with the save-side activation
// flags. An inactive or residual slot reports active false and every entry
// deactivated without its slot data being read. The whole catalog is validated
// before one save byte is touched, so incomplete or conflicting catalog data can
// never be answered with a state.
func GetSummoningPools(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetSummoningPoolsResult, error) {
	if engine == nil {
		return GetSummoningPoolsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetSummoningPoolsResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogSummoningPools(gameCatalog)
	if err != nil {
		return GetSummoningPoolsResult{}, err
	}
	eventFlagIDs := make([]uint32, len(declared))
	for index, pool := range declared {
		eventFlagIDs[index] = pool.eventFlagID
	}
	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetSummoningPoolsResult{}, err
	}

	result := GetSummoningPoolsResult{
		SaveSessionID:  flags.SaveSessionID,
		SaveRevision:   flags.SaveRevision,
		CharacterID:    flags.CharacterID,
		Active:         flags.Active,
		SummoningPools: make([]SummoningPoolEntry, 0, len(declared)),
	}
	for _, pool := range declared {
		entry := pool.entry
		entry.Activated = flags.Flags[pool.eventFlagID]
		result.SummoningPools = append(result.SummoningPools, entry)
	}
	return result, nil
}

type declaredSummoningPool struct {
	entry       SummoningPoolEntry
	eventFlagID uint32
}

// catalogSummoningPools returns the declared pools ordered by region label, then
// name and then key. It fails closed on a resource whose summoning pool document
// is missing and on two pools that claim the same activation flag, which no
// single document can rule out.
func catalogSummoningPools(gameCatalog *gamecatalog.Catalog) ([]declaredSummoningPool, error) {
	declared := make([]declaredSummoningPool, 0)
	flagOwners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindSummoningPool {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("summoning pool %q: %w", summary.Key, err)
		}
		if resource.SummoningPool == nil {
			return nil, fmt.Errorf(
				"summoning pool %q carries no summoning pool document", summary.Key)
		}
		flag := resource.SummoningPool.ActivationEventFlagID.Value
		if owner, duplicate := flagOwners[flag]; duplicate {
			return nil, fmt.Errorf(
				"summoning pools %q and %q both declare event flag %d", owner, summary.Key, flag)
		}
		flagOwners[flag] = summary.Key
		declared = append(declared, declaredSummoningPool{
			entry: SummoningPoolEntry{
				Kind:        resource.Kind,
				Key:         resource.Key,
				Name:        resource.SummoningPool.Name.Value,
				RegionLabel: resource.SummoningPool.RegionLabel.Value,
			},
			eventFlagID: flag,
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
