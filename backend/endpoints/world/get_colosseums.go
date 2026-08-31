/*
Endpoint: GetColosseums
EndpointID: get_colosseums
Purpose: Returns colosseums and their unlock state.
How it works: The handler reads every colosseum resource GameCatalog declares, requires one unlock event flag per resource and a unique flag across them, and resolves all of those flags in one bulk SaveEngine read. It decodes no flag itself.
Supported resource types: ColosseumDocument.
Input variables: saveSessionID, characterID.
GameCatalog variables read: resource kind and key plus the colosseum name and unlock event flag ID.
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

// GetColosseumsEndpointID is the stable backend identifier of GetColosseums.
const GetColosseumsEndpointID = "get_colosseums"

// GetColosseumsDefinition describes the public getter contract.
var GetColosseumsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetColosseums",
	ID:                         GetColosseumsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ColosseumDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns colosseums and their unlock state.",
})

// ColosseumEntry is one catalog-declared colosseum and its current state. The
// unlock event flag stays an internal save-format detail.
type ColosseumEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	Name     string              `json:"name"`
	Unlocked bool                `json:"unlocked"`
}

// GetColosseumsResult is the deterministic result for one character slot.
type GetColosseumsResult struct {
	SaveSessionID string           `json:"saveSessionID"`
	SaveRevision  string           `json:"saveRevision"`
	CharacterID   int              `json:"characterID"`
	Active        bool             `json:"active"`
	Colosseums    []ColosseumEntry `json:"colosseums"`
}

// GetColosseums joins the catalog declarations with the save-side unlock flags.
// An inactive or residual slot reports active false and every entry locked
// without its slot data being read.
func GetColosseums(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetColosseumsResult, error) {
	if engine == nil {
		return GetColosseumsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetColosseumsResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogColosseums(gameCatalog)
	if err != nil {
		return GetColosseumsResult{}, err
	}
	eventFlagIDs := make([]uint32, len(declared))
	for index, colosseum := range declared {
		eventFlagIDs[index] = colosseum.eventFlagID
	}
	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetColosseumsResult{}, err
	}

	result := GetColosseumsResult{
		SaveSessionID: flags.SaveSessionID,
		SaveRevision:  flags.SaveRevision,
		CharacterID:   flags.CharacterID,
		Active:        flags.Active,
		Colosseums:    make([]ColosseumEntry, 0, len(declared)),
	}
	for _, colosseum := range declared {
		entry := colosseum.entry
		entry.Unlocked = flags.Flags[colosseum.eventFlagID]
		result.Colosseums = append(result.Colosseums, entry)
	}
	return result, nil
}

type declaredColosseum struct {
	entry       ColosseumEntry
	eventFlagID uint32
}

// catalogColosseums returns the declared colosseums ordered by name and then
// key. It fails closed on a resource whose colosseum document is missing and on
// two colosseums that claim the same event flag, which no single document can
// rule out.
func catalogColosseums(gameCatalog *gamecatalog.Catalog) ([]declaredColosseum, error) {
	declared := make([]declaredColosseum, 0)
	flagOwners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindColosseum {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("colosseum %q: %w", summary.Key, err)
		}
		if resource.Colosseum == nil {
			return nil, fmt.Errorf("colosseum %q carries no colosseum document", summary.Key)
		}
		flag := resource.Colosseum.UnlockEventFlagID.Value
		if owner, duplicate := flagOwners[flag]; duplicate {
			return nil, fmt.Errorf(
				"colosseums %q and %q both declare event flag %d", owner, summary.Key, flag)
		}
		flagOwners[flag] = summary.Key
		declared = append(declared, declaredColosseum{
			entry: ColosseumEntry{
				Kind: resource.Kind,
				Key:  resource.Key,
				Name: resource.Colosseum.Name.Value,
			},
			eventFlagID: flag,
		})
	}

	sort.SliceStable(declared, func(i, j int) bool {
		if declared[i].entry.Name != declared[j].entry.Name {
			return declared[i].entry.Name < declared[j].entry.Name
		}
		return declared[i].entry.Key < declared[j].entry.Key
	})
	return declared, nil
}
