/*
Endpoint: GetCookbooks
EndpointID: get_cookbooks
Purpose: Returns cookbooks and their unlock state.
How it works: The runtime handler asks GameCatalog for every ItemDocument of family "goods" that declares exactly one item.unlocks entry of kind "cookbook", collects the event flag identifiers those entries carry and asks SaveEngine once for the state of all of them in one character slot of an already loaded session. A cookbook is unlocked when its own event flag is set. The endpoint opens no file, reads no snapshot, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument: Cookbook.
Input variables: saveSessionID, characterID, availabilityFilter.
GameCatalog variables read: for every item resource the resource kind, key and family and, per item.unlocks entry of kind "cookbook", the eventFlagID, the name and the category.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the event flag bits of the collected identifiers; the getter is non-mutating and writes nothing.
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

// GetCookbooksEndpointID is the stable backend identifier of GetCookbooks.
const GetCookbooksEndpointID = "get_cookbooks"

// GetCookbooksDefinition describes the public getter contract.
var GetCookbooksDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCookbooks",
	ID:                         GetCookbooksEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Cookbook",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "availabilityFilter"},
	Description:                "Returns cookbooks and their unlock state.",
})

// The two non-empty availabilityFilter values. The empty string is the third
// accepted value and means "every cookbook"; it needs no constant of its own.
const (
	CookbookAvailabilityUnlocked = "unlocked"
	CookbookAvailabilityLocked   = "locked"
)

// cookbookUnlockKind is the item.unlocks kind that selects a cookbook. It is the
// only selector: no name pattern and no acquisition field decides whether an
// ItemDocument declares one. The family of the declaring resource is not a
// second selector either — it is a required property of a resource already
// selected by this kind.
const cookbookUnlockKind = "cookbook"

// CookbookEntry is one cookbook the catalog declares, together with whether the
// requested character has unlocked it. One declaring resource produces exactly
// one entry, so the kind and key pair is unique across a result. It carries the
// public GameCatalog identity of that resource and the two presentation values the
// unlock entry stores — nothing else. The event flag identifier behind the state
// stays internal, and no provenance, source record, offset or raw save byte is
// exposed.
type CookbookEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	Name     string              `json:"name"`
	Category string              `json:"category"`
	Unlocked bool                `json:"unlocked"`
}

// GetCookbooksResult is the typed result of GetCookbooks: the cookbooks that
// passed availabilityFilter, in a deterministic order.
//
// An inactive slot — including a residual one — reports Active false and every
// catalog cookbook as locked, because its event flags are never read.
type GetCookbooksResult struct {
	SaveSessionID string          `json:"saveSessionID"`
	CharacterID   int             `json:"characterID"`
	Active        bool            `json:"active"`
	Cookbooks     []CookbookEntry `json:"cookbooks"`
}

// GetCookbooks returns every cookbook GameCatalog declares, each one marked with
// whether the requested character slot of an existing save session has unlocked
// it.
//
// GameCatalog is the only source of a cookbook's identity, name, category and
// event flag: a cookbook is an ItemDocument of family "goods" that carries
// exactly one item.unlocks entry of kind "cookbook", and the eventFlagID of that
// entry is the only state the unlock is read from. acquisition.worldPickupFlagID
// is deliberately not consulted, and neither is the inventory: an owned but
// unregistered cookbook item is not an unlock and is never reported as one.
//
// One declaring resource yields exactly one entry, so the public kind and key
// identity of a cookbook is unique across a result.
//
// The identifiers of every declared cookbook are collected first and handed to
// SaveEngine in one call, so the slot is located and walked once. A flag
// SaveEngine cannot place is its error and is never answered as a false.
//
// The session must already exist; this endpoint never creates one, so it calls
// neither LoadSave nor any other endpoint, opens no file and returns no raw save
// byte.
//
// availabilityFilter accepts the empty string for every cookbook, "unlocked" for
// the unlocked ones and "locked" for the rest. It is matched exactly and
// case-sensitively, is never trimmed and has no alias, so anything else is
// rejected. It can only remove entries; it never changes their order.
//
// A cookbook without the catalog data this result states — an empty resource
// key, an unknown or non-goods item family, a second cookbook unlock in one
// resource, an unknown eventFlagID, an unknown or empty name, an unknown or
// empty category, or a second cookbook on an eventFlagID another one already
// owns — is a fail-closed error: no value is invented, no entry is silently
// dropped and no conflict is resolved by preferring one of the two.
func GetCookbooks(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	availabilityFilter string,
) (GetCookbooksResult, error) {
	if engine == nil {
		return GetCookbooksResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetCookbooksResult{}, errors.New("game catalog is not available")
	}
	switch availabilityFilter {
	case "", CookbookAvailabilityUnlocked, CookbookAvailabilityLocked:
	default:
		return GetCookbooksResult{}, fmt.Errorf(
			"availabilityFilter must be %q, %q or empty; got %q",
			CookbookAvailabilityUnlocked, CookbookAvailabilityLocked, availabilityFilter)
	}

	declared, err := catalogCookbooks(gameCatalog)
	if err != nil {
		return GetCookbooksResult{}, err
	}

	eventFlagIDs := make([]uint32, 0, len(declared))
	for _, cookbook := range declared {
		eventFlagIDs = append(eventFlagIDs, cookbook.eventFlagID)
	}

	// One call for the whole list: the slot is located once and an inactive slot
	// leaves the returned map empty, so every cookbook stays locked without its
	// bitfield ever being located or read.
	state, err := engine.GetEventFlags(saveSessionID, characterID, eventFlagIDs)
	if err != nil {
		return GetCookbooksResult{}, err
	}

	result := GetCookbooksResult{
		SaveSessionID: state.SaveSessionID,
		CharacterID:   state.CharacterID,
		Active:        state.Active,
		Cookbooks:     make([]CookbookEntry, 0, len(declared)),
	}
	for _, cookbook := range declared {
		entry := cookbook.entry
		entry.Unlocked = state.Flags[cookbook.eventFlagID]
		switch availabilityFilter {
		case CookbookAvailabilityUnlocked:
			if !entry.Unlocked {
				continue
			}
		case CookbookAvailabilityLocked:
			if entry.Unlocked {
				continue
			}
		}
		result.Cookbooks = append(result.Cookbooks, entry)
	}
	return result, nil
}

// declaredCookbook is one catalog cookbook and the event flag its state is read
// from. The identifier is kept next to the entry instead of inside it, because
// it is internal and never reaches the public result.
type declaredCookbook struct {
	entry       CookbookEntry
	eventFlagID uint32
}

// catalogCookbooks returns every cookbook GameCatalog declares in the
// deterministic order of the result.
//
// The order is category, then name, then the key of the declaring resource. One
// resource contributes exactly one entry, so the three keys of two entries can
// only all agree when two resources share a category and a name and repeat a
// key, which the catalog rejects; the sort stays stable so catalog order remains
// the last deterministic tie-breaker instead of the internal identifier, which
// would leak into the contract.
func catalogCookbooks(gameCatalog *gamecatalog.Catalog) ([]declaredCookbook, error) {
	declared := make([]declaredCookbook, 0)
	owners := make(map[uint32]string)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindItem {
			continue
		}
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return nil, fmt.Errorf("item %q: %w", summary.Key, err)
		}
		cookbook, found, err := declaredCookbookFromResource(resource)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		if owner, taken := owners[cookbook.eventFlagID]; taken {
			return nil, fmt.Errorf(
				"cookbooks %q and %q both declare event flag %d",
				owner, resource.Key, cookbook.eventFlagID)
		}
		owners[cookbook.eventFlagID] = resource.Key
		declared = append(declared, cookbook)
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

// declaredCookbookFromResource applies the one cookbook-definition rule shared
// by the getter and setter. A resource without a cookbook unlock is not a
// cookbook; once it declares one, every required field is fail-closed.
func declaredCookbookFromResource(resource schema.Resource) (declaredCookbook, bool, error) {
	if resource.Item == nil {
		return declaredCookbook{}, false, nil
	}

	index, found := 0, 0
	for position, unlock := range resource.Item.Unlocks {
		if unlock.Kind.Known && unlock.Kind.Value == cookbookUnlockKind {
			if found == 0 {
				index = position
			}
			found++
		}
	}
	if found == 0 {
		return declaredCookbook{}, false, nil
	}
	if resource.Key == "" {
		return declaredCookbook{}, false, fmt.Errorf("a cookbook resource carries no key")
	}
	if !resource.Item.Family.Known {
		return declaredCookbook{}, false, fmt.Errorf(
			"cookbook %q has no known item family, want %q",
			resource.Key, schema.ItemFamilyGoods)
	}
	if resource.Item.Family.Value != schema.ItemFamilyGoods {
		return declaredCookbook{}, false, fmt.Errorf(
			"cookbook %q has item family %q, want %q",
			resource.Key, resource.Item.Family.Value, schema.ItemFamilyGoods)
	}
	if found > 1 {
		return declaredCookbook{}, false, fmt.Errorf(
			"cookbook %q declares %d cookbook unlocks, want exactly one",
			resource.Key, found)
	}

	unlock := resource.Item.Unlocks[index]
	if !unlock.EventFlagID.Known {
		return declaredCookbook{}, false, fmt.Errorf(
			"cookbook %q unlock %d has no known event flag ID", resource.Key, index)
	}
	if !unlock.Name.Known || unlock.Name.Value == "" {
		return declaredCookbook{}, false, fmt.Errorf(
			"cookbook %q unlock %d has no known name", resource.Key, index)
	}
	if !unlock.Category.Known || unlock.Category.Value == "" {
		return declaredCookbook{}, false, fmt.Errorf(
			"cookbook %q unlock %d has no known category", resource.Key, index)
	}
	block := unlock.EventFlagID.Value / 1000
	if block != 67 && block != 68 {
		return declaredCookbook{}, false, fmt.Errorf(
			"event flag %d lies in block %d, which this reader does not support",
			unlock.EventFlagID.Value, block)
	}

	return declaredCookbook{
		entry: CookbookEntry{
			Kind:     resource.Kind,
			Key:      resource.Key,
			Name:     unlock.Name.Value,
			Category: unlock.Category.Value,
		},
		eventFlagID: unlock.EventFlagID.Value,
	}, true, nil
}
