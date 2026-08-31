/*
Endpoint: GetWhetblades
EndpointID: get_whetblades
Purpose: Returns Whetblades and their unlock state.
How it works: The handler selects goods resources that declare exactly one whetblade unlock, reads all of their event flags and checks positive-quantity common and key Inventory records. A resource is unlocked when either confirmed representation is present.
Supported resource types: ItemDocument: Whetblade.
Input variables: saveSessionID, characterID, availabilityFilter.
GameCatalog variables read: resource kind and key plus item family, gameID and the name and eventFlagID of the whetblade unlock.
Save variables read: the character activity flag, the requested event flag bits and positive-quantity goods records in common and key InventoryHeld; the getter writes nothing.
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

// GetWhetbladesEndpointID is the stable backend identifier of GetWhetblades.
const GetWhetbladesEndpointID = "get_whetblades"

// GetWhetbladesDefinition describes the public getter contract.
var GetWhetbladesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetWhetblades",
	ID:                         GetWhetbladesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Whetblade",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "availabilityFilter"},
	Description:                "Returns Whetblades and their unlock state.",
})

const (
	WhetbladeAvailabilityUnlocked = "unlocked"
	WhetbladeAvailabilityLocked   = "locked"
	whetbladeUnlockKind           = "whetblade"
)

// WhetbladeEntry is one catalog-declared Whetblade and its current state. The
// event flag and goods game ID remain internal save-format details.
type WhetbladeEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	Name     string              `json:"name"`
	Unlocked bool                `json:"unlocked"`
}

// GetWhetbladesResult is the deterministic filtered result for one character.
type GetWhetbladesResult struct {
	SaveSessionID string           `json:"saveSessionID"`
	SaveRevision  string           `json:"saveRevision"`
	CharacterID   int              `json:"characterID"`
	Active        bool             `json:"active"`
	Whetblades    []WhetbladeEntry `json:"whetblades"`
}

// GetWhetblades joins catalog declarations with both native unlock signals.
// availabilityFilter accepts only empty, "unlocked" or "locked".
func GetWhetblades(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	availabilityFilter string,
) (GetWhetbladesResult, error) {
	if engine == nil {
		return GetWhetbladesResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetWhetbladesResult{}, errors.New("game catalog is not available")
	}
	switch availabilityFilter {
	case "", WhetbladeAvailabilityUnlocked, WhetbladeAvailabilityLocked:
	default:
		return GetWhetbladesResult{}, fmt.Errorf(
			"availabilityFilter must be %q, %q or empty; got %q",
			WhetbladeAvailabilityUnlocked, WhetbladeAvailabilityLocked, availabilityFilter)
	}

	declared, err := catalogWhetblades(gameCatalog)
	if err != nil {
		return GetWhetbladesResult{}, err
	}
	eventFlagIDs := make([]uint32, len(declared))
	gameIDs := make([]uint32, len(declared))
	for index, whetblade := range declared {
		eventFlagIDs[index] = whetblade.eventFlagID
		gameIDs[index] = whetblade.gameID
	}

	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetWhetbladesResult{}, err
	}
	owned := map[uint32]bool{}
	if flags.Active {
		presence, err := engine.GetInventoryGoodsPresence(saveSessionID, characterID, gameIDs)
		if err != nil {
			return GetWhetbladesResult{}, err
		}
		if err := requireSameSaveRevision(flags.SaveRevision, presence.SaveRevision); err != nil {
			return GetWhetbladesResult{}, err
		}
		owned = presence.Presence
	}

	result := GetWhetbladesResult{
		SaveSessionID: flags.SaveSessionID,
		SaveRevision:  flags.SaveRevision,
		CharacterID:   flags.CharacterID,
		Active:        flags.Active,
		Whetblades:    make([]WhetbladeEntry, 0, len(declared)),
	}
	for _, whetblade := range declared {
		entry := whetblade.entry
		entry.Unlocked = flags.Flags[whetblade.eventFlagID] || owned[whetblade.gameID]
		if availabilityFilter == WhetbladeAvailabilityUnlocked && !entry.Unlocked {
			continue
		}
		if availabilityFilter == WhetbladeAvailabilityLocked && entry.Unlocked {
			continue
		}
		result.Whetblades = append(result.Whetblades, entry)
	}
	return result, nil
}

type declaredWhetblade struct {
	entry       WhetbladeEntry
	eventFlagID uint32
	gameID      uint32
}

func catalogWhetblades(gameCatalog *gamecatalog.Catalog) ([]declaredWhetblade, error) {
	declared := make([]declaredWhetblade, 0)
	flagOwners := make(map[uint32]string)
	gameIDOwners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindItem {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("item %q: %w", summary.Key, err)
		}
		whetblade, found, err := declaredWhetbladeFromResource(resource)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		if owner, duplicate := flagOwners[whetblade.eventFlagID]; duplicate {
			return nil, fmt.Errorf("whetblades %q and %q both declare event flag %d",
				owner, resource.Key, whetblade.eventFlagID)
		}
		if owner, duplicate := gameIDOwners[whetblade.gameID]; duplicate {
			return nil, fmt.Errorf("whetblades %q and %q both declare game ID 0x%08X",
				owner, resource.Key, whetblade.gameID)
		}
		flagOwners[whetblade.eventFlagID] = resource.Key
		gameIDOwners[whetblade.gameID] = resource.Key
		declared = append(declared, whetblade)
	}

	sort.SliceStable(declared, func(i, j int) bool {
		if declared[i].entry.Name != declared[j].entry.Name {
			return declared[i].entry.Name < declared[j].entry.Name
		}
		return declared[i].entry.Key < declared[j].entry.Key
	})
	return declared, nil
}

func declaredWhetbladeFromResource(
	resource schema.Resource,
) (declaredWhetblade, bool, error) {
	if resource.Item == nil {
		return declaredWhetblade{}, false, nil
	}
	index, count := 0, 0
	for position, unlock := range resource.Item.Unlocks {
		if unlock.Kind.Known && unlock.Kind.Value == whetbladeUnlockKind {
			index = position
			count++
		}
	}
	if count == 0 {
		return declaredWhetblade{}, false, nil
	}
	if resource.Key == "" {
		return declaredWhetblade{}, false, errors.New("a whetblade resource carries no key")
	}
	if !resource.Item.Family.Known || resource.Item.Family.Value != schema.ItemFamilyGoods {
		return declaredWhetblade{}, false, fmt.Errorf(
			"whetblade %q must have known item family %q", resource.Key, schema.ItemFamilyGoods)
	}
	if count != 1 {
		return declaredWhetblade{}, false, fmt.Errorf(
			"whetblade %q declares %d whetblade unlocks, want exactly one", resource.Key, count)
	}
	if !resource.Item.GameID.Known || resource.Item.GameID.Value&0xF0000000 != 0x40000000 {
		return declaredWhetblade{}, false, fmt.Errorf(
			"whetblade %q has no valid goods game ID", resource.Key)
	}
	unlock := resource.Item.Unlocks[index]
	if !unlock.EventFlagID.Known {
		return declaredWhetblade{}, false, fmt.Errorf(
			"whetblade %q unlock %d has no known event flag ID", resource.Key, index)
	}
	block := unlock.EventFlagID.Value / 1000
	if block != 60 && block != 65 {
		return declaredWhetblade{}, false, fmt.Errorf(
			"event flag %d lies in block %d, which this reader does not support",
			unlock.EventFlagID.Value, block)
	}
	if !unlock.Name.Known || unlock.Name.Value == "" {
		return declaredWhetblade{}, false, fmt.Errorf(
			"whetblade %q unlock %d has no known name", resource.Key, index)
	}
	return declaredWhetblade{
		entry:       WhetbladeEntry{Kind: resource.Kind, Key: resource.Key, Name: unlock.Name.Value},
		eventFlagID: unlock.EventFlagID.Value,
		gameID:      resource.Item.GameID.Value,
	}, true, nil
}
