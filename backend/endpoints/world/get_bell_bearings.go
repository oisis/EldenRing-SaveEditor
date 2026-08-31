/*
Endpoint: GetBellBearings
EndpointID: get_bell_bearings
Purpose: Returns Bell Bearings and their unlock state.
How it works: The handler selects goods resources that declare exactly one bell_bearing unlock and reads all of their acquisition flags in one read-only SaveEngine operation.
Supported resource types: ItemDocument: BellBearing.
Input variables: saveSessionID, characterID, availabilityFilter.
GameCatalog variables read: resource kind and key plus item family and the name, category and eventFlagID of the bell_bearing unlock.
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

// GetBellBearingsEndpointID is the stable backend identifier of GetBellBearings.
const GetBellBearingsEndpointID = "get_bell_bearings"

// GetBellBearingsDefinition describes the public getter contract.
var GetBellBearingsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBellBearings",
	ID:                         GetBellBearingsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: BellBearing",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "availabilityFilter"},
	Description:                "Returns Bell Bearings and their unlock state.",
})

const (
	BellBearingAvailabilityUnlocked = "unlocked"
	BellBearingAvailabilityLocked   = "locked"
	bellBearingUnlockKind           = "bell_bearing"
	bellBearingEventFlagBlock       = uint32(11109)
)

// BellBearingEntry is one catalog-declared Bell Bearing and its current state.
// The event flag remains an internal save-format detail.
type BellBearingEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	Name     string              `json:"name"`
	Category string              `json:"category"`
	Unlocked bool                `json:"unlocked"`
}

// GetBellBearingsResult is the deterministic filtered result for one character.
type GetBellBearingsResult struct {
	SaveSessionID string             `json:"saveSessionID"`
	SaveRevision  string             `json:"saveRevision"`
	CharacterID   int                `json:"characterID"`
	Active        bool               `json:"active"`
	BellBearings  []BellBearingEntry `json:"bellBearings"`
}

// GetBellBearings joins catalog declarations with their acquisition flags.
// availabilityFilter accepts only empty, "unlocked" or "locked".
func GetBellBearings(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	availabilityFilter string,
) (GetBellBearingsResult, error) {
	if engine == nil {
		return GetBellBearingsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetBellBearingsResult{}, errors.New("game catalog is not available")
	}
	switch availabilityFilter {
	case "", BellBearingAvailabilityUnlocked, BellBearingAvailabilityLocked:
	default:
		return GetBellBearingsResult{}, fmt.Errorf(
			"availabilityFilter must be %q, %q or empty; got %q",
			BellBearingAvailabilityUnlocked, BellBearingAvailabilityLocked, availabilityFilter)
	}

	declared, err := catalogBellBearings(gameCatalog)
	if err != nil {
		return GetBellBearingsResult{}, err
	}
	eventFlagIDs := make([]uint32, len(declared))
	for index, bellBearing := range declared {
		eventFlagIDs[index] = bellBearing.eventFlagID
	}
	flags, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetBellBearingsResult{}, err
	}

	result := GetBellBearingsResult{
		SaveSessionID: flags.SaveSessionID,
		SaveRevision:  flags.SaveRevision,
		CharacterID:   flags.CharacterID,
		Active:        flags.Active,
		BellBearings:  make([]BellBearingEntry, 0, len(declared)),
	}
	for _, bellBearing := range declared {
		entry := bellBearing.entry
		entry.Unlocked = flags.Flags[bellBearing.eventFlagID]
		if availabilityFilter == BellBearingAvailabilityUnlocked && !entry.Unlocked {
			continue
		}
		if availabilityFilter == BellBearingAvailabilityLocked && entry.Unlocked {
			continue
		}
		result.BellBearings = append(result.BellBearings, entry)
	}
	return result, nil
}

type declaredBellBearing struct {
	entry       BellBearingEntry
	eventFlagID uint32
}

func catalogBellBearings(gameCatalog *gamecatalog.Catalog) ([]declaredBellBearing, error) {
	declared := make([]declaredBellBearing, 0)
	flagOwners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindItem {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("item %q: %w", summary.Key, err)
		}
		bellBearing, found, err := declaredBellBearingFromResource(resource)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		if owner, duplicate := flagOwners[bellBearing.eventFlagID]; duplicate {
			return nil, fmt.Errorf("bell bearings %q and %q both declare event flag %d",
				owner, resource.Key, bellBearing.eventFlagID)
		}
		flagOwners[bellBearing.eventFlagID] = resource.Key
		declared = append(declared, bellBearing)
	}

	sort.SliceStable(declared, func(i, j int) bool {
		left, right := declared[i].entry, declared[j].entry
		if left.Category != right.Category {
			return left.Category < right.Category
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.Key < right.Key
	})
	return declared, nil
}

func declaredBellBearingFromResource(
	resource schema.Resource,
) (declaredBellBearing, bool, error) {
	if resource.Item == nil {
		return declaredBellBearing{}, false, nil
	}
	index, count := 0, 0
	for position, unlock := range resource.Item.Unlocks {
		if unlock.Kind.Known && unlock.Kind.Value == bellBearingUnlockKind {
			index = position
			count++
		}
	}
	if count == 0 {
		return declaredBellBearing{}, false, nil
	}
	if resource.Key == "" {
		return declaredBellBearing{}, false, errors.New("a bell bearing resource carries no key")
	}
	if !resource.Item.Family.Known || resource.Item.Family.Value != schema.ItemFamilyGoods {
		return declaredBellBearing{}, false, fmt.Errorf(
			"bell bearing %q must have known item family %q", resource.Key, schema.ItemFamilyGoods)
	}
	if count != 1 {
		return declaredBellBearing{}, false, fmt.Errorf(
			"bell bearing %q declares %d bell_bearing unlocks, want exactly one",
			resource.Key, count)
	}

	unlock := resource.Item.Unlocks[index]
	if !unlock.EventFlagID.Known {
		return declaredBellBearing{}, false, fmt.Errorf(
			"bell bearing %q unlock %d has no known event flag ID", resource.Key, index)
	}
	block := unlock.EventFlagID.Value / 1000
	if block != bellBearingEventFlagBlock {
		return declaredBellBearing{}, false, fmt.Errorf(
			"event flag %d lies in block %d, which this reader does not support",
			unlock.EventFlagID.Value, block)
	}
	if !unlock.Name.Known || unlock.Name.Value == "" {
		return declaredBellBearing{}, false, fmt.Errorf(
			"bell bearing %q unlock %d has no known name", resource.Key, index)
	}
	if !unlock.Category.Known || unlock.Category.Value == "" {
		return declaredBellBearing{}, false, fmt.Errorf(
			"bell bearing %q unlock %d has no known category", resource.Key, index)
	}
	return declaredBellBearing{
		entry: BellBearingEntry{
			Kind: resource.Kind, Key: resource.Key,
			Name: unlock.Name.Value, Category: unlock.Category.Value,
		},
		eventFlagID: unlock.EventFlagID.Value,
	}, true, nil
}
