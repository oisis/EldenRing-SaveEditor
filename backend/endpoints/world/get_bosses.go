/*
Endpoint: GetBosses
EndpointID: get_bosses
Purpose: Returns bosses and whether they have been defeated.
How it works: The handler reads every boss resource GameCatalog declares, requires one defeat event flag per resource and a unique flag across them, and resolves all of those flags in one bulk SaveEngine read. It decodes no flag itself.
Supported resource types: BossDocument.
Input variables: saveSessionID, characterID.
GameCatalog variables read: resource kind and key plus the boss name, region label, encounter type, remembrance fact and defeat event flag ID.
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

// GetBossesEndpointID is the stable backend identifier of GetBosses.
const GetBossesEndpointID = "get_bosses"

// GetBossesDefinition describes the public getter contract. The old contract
// declared regionKind and regionKey, which the curated Bosses table does not
// support: it declares a plain text region and no region resource identity at
// all. Publishing that pair would promise a relation no source declares, so the
// input is exactly the session and the character slot.
var GetBossesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBosses",
	ID:                         GetBossesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "BossDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns bosses and whether they have been defeated.",
})

// BossEntry is one catalog-declared boss encounter and its current state.
// RegionLabel is the curated presentation and grouping label of the encounter,
// not a reference to a region resource. The defeat event flag stays an internal
// save-format detail.
type BossEntry struct {
	Kind          schema.ResourceKind `json:"kind"`
	Key           string              `json:"key"`
	Name          string              `json:"name"`
	RegionLabel   string              `json:"regionLabel"`
	EncounterType string              `json:"encounterType"`
	Remembrance   bool                `json:"remembrance"`
	Defeated      bool                `json:"defeated"`
}

// GetBossesResult is the deterministic result for one character slot.
type GetBossesResult struct {
	SaveSessionID string      `json:"saveSessionID"`
	SaveRevision  string      `json:"saveRevision"`
	CharacterID   int         `json:"characterID"`
	Active        bool        `json:"active"`
	Bosses        []BossEntry `json:"bosses"`
}

// GetBosses joins the catalog declarations with the save-side defeat flags. An
// inactive or residual slot reports active false and every entry undefeated
// without its slot data being read. The whole catalog is validated before one
// save byte is touched, so incomplete or conflicting catalog data can never be
// answered with a state.
func GetBosses(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetBossesResult, error) {
	if engine == nil {
		return GetBossesResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetBossesResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogBosses(gameCatalog)
	if err != nil {
		return GetBossesResult{}, err
	}
	eventFlagIDs := make([]uint32, len(declared))
	for index, boss := range declared {
		eventFlagIDs[index] = boss.eventFlagID
	}
	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetBossesResult{}, err
	}

	result := GetBossesResult{
		SaveSessionID: flags.SaveSessionID,
		SaveRevision:  flags.SaveRevision,
		CharacterID:   flags.CharacterID,
		Active:        flags.Active,
		Bosses:        make([]BossEntry, 0, len(declared)),
	}
	for _, boss := range declared {
		entry := boss.entry
		entry.Defeated = flags.Flags[boss.eventFlagID]
		result.Bosses = append(result.Bosses, entry)
	}
	return result, nil
}

type declaredBoss struct {
	entry       BossEntry
	eventFlagID uint32
}

// catalogBosses returns the declared bosses ordered by region label, then name
// and then key. It fails closed on a resource whose boss document is missing and
// on two bosses that claim the same defeat flag, which no single document can
// rule out.
func catalogBosses(gameCatalog *gamecatalog.Catalog) ([]declaredBoss, error) {
	declared := make([]declaredBoss, 0)
	flagOwners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindBoss {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("boss %q: %w", summary.Key, err)
		}
		if resource.Boss == nil {
			return nil, fmt.Errorf("boss %q carries no boss document", summary.Key)
		}
		flag := resource.Boss.DefeatEventFlagID.Value
		if owner, duplicate := flagOwners[flag]; duplicate {
			return nil, fmt.Errorf(
				"bosses %q and %q both declare event flag %d", owner, summary.Key, flag)
		}
		flagOwners[flag] = summary.Key
		declared = append(declared, declaredBoss{
			entry: BossEntry{
				Kind:          resource.Kind,
				Key:           resource.Key,
				Name:          resource.Boss.Name.Value,
				RegionLabel:   resource.Boss.RegionLabel.Value,
				EncounterType: resource.Boss.EncounterType.Value,
				Remembrance:   resource.Boss.Remembrance.Value,
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
