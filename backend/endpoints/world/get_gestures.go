/*
Endpoint: GetGestures
EndpointID: get_gestures
Purpose: Returns gestures and whether each one is unlocked.
How it works: The runtime handler asks SaveEngine for the raw 64-record GestureGameData block of one character slot of an already loaded session, asks GameCatalog for every gesture slot it knows, and reports a gesture as unlocked when its canonical slotID is present in that block. The endpoint opens no file, reads no snapshot, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument: Gesture.
Input variables: saveSessionID, characterID, availabilityFilter.
GameCatalog variables read: for every item resource of the gesture family the resource kind and key and, per declared gesture slot, the canonical slotID, the name and the category.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the 64 raw GestureGameData records of its slot data; the getter is non-mutating and writes nothing.
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

// GetGesturesEndpointID is the stable backend identifier of GetGestures.
const GetGesturesEndpointID = "get_gestures"

// GetGesturesDefinition describes the public getter contract.
var GetGesturesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetGestures",
	ID:                         GetGesturesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Gesture",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "availabilityFilter"},
	Description:                "Returns gestures and whether each one is unlocked.",
})

// The two non-empty availabilityFilter values. The empty string is the third
// accepted value and means "every gesture"; it needs no constant of its own.
const (
	GestureAvailabilityUnlocked = "unlocked"
	GestureAvailabilityLocked   = "locked"
)

// The two stored values that unlock nothing. 0xFFFFFFFE is the confirmed native
// empty-slot sentinel of GestureGameData and 0 is the value an untouched record
// carries; neither is a canonical slotID, so neither can match one. They are
// named here because the result states the rule, not because the raw block is
// filtered: SaveEngine returns every record exactly as stored.
const (
	gestureEmptySentinel uint32 = 0xFFFFFFFE
	gestureZeroRecord    uint32 = 0
)

// GestureEntry is one gesture the catalog knows, together with whether the
// requested character has it. It carries the public GameCatalog identity of the
// resource, the canonical save slot ID of this one gesture and the two
// presentation values the catalog stores for it — nothing else. No provenance,
// no source records, no icon and no raw save byte is exposed.
//
// Kind and Key identify the catalog resource. SlotID identifies the gesture
// inside that resource, because one resource may declare more than one gesture
// slot; two entries then share Kind and Key and differ in SlotID.
type GestureEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	SlotID   uint32              `json:"slotID"`
	Name     string              `json:"name"`
	Category string              `json:"category"`
	Unlocked bool                `json:"unlocked"`
}

// GetGesturesResult is the typed result of GetGestures: the gestures that passed
// availabilityFilter, in a deterministic order.
//
// An inactive slot — including a residual one — reports Active false and every
// catalog gesture as locked, because its GestureGameData is never read.
type GetGesturesResult struct {
	SaveSessionID string         `json:"saveSessionID"`
	CharacterID   int            `json:"characterID"`
	Active        bool           `json:"active"`
	Gestures      []GestureEntry `json:"gestures"`
}

// GetGestures returns every gesture GameCatalog knows, each one marked with
// whether the requested character slot of an existing save session has unlocked
// it.
//
// The unlock state comes from the raw 64-record GestureGameData block of that
// slot and from nothing else. A gesture is unlocked when its canonical slotID is
// present in the block as an exact uint32 match. No event flag is read, no even
// value is converted into an odd one, no body-type pairing is assumed, and the
// save is never sanitised, repaired or normalised. A stored value that matches
// no canonical slotID — including the 0xFFFFFFFE empty sentinel, a plain 0 and an
// unknown value — unlocks nothing, and the same slotID stored twice still marks
// its one catalog entry unlocked once.
//
// The endpoint is thin, but it owns the join: it rejects a missing engine, a
// missing catalog and an unknown availabilityFilter, asks SaveEngine for the raw
// block and GameCatalog for the gesture definitions, and combines the two. The
// session must already exist; this endpoint never creates one, so it calls
// neither LoadSave nor any other endpoint, opens no file and returns no raw save
// byte.
//
// availabilityFilter accepts the empty string for every gesture, "unlocked" for
// the unlocked ones and "locked" for the rest. It is matched exactly and
// case-sensitively, is never trimmed and has no alias, so anything else is
// rejected. It can only remove entries; it never changes their order.
//
// A gesture resource without the catalog data this result states — a missing
// item or gesture document, an empty key, an unknown slotID, an unknown or empty
// name or an unknown or empty category — is a fail-closed error: no value is
// invented for it.
func GetGestures(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	availabilityFilter string,
) (GetGesturesResult, error) {
	if engine == nil {
		return GetGesturesResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetGesturesResult{}, errors.New("game catalog is not available")
	}
	switch availabilityFilter {
	case "", GestureAvailabilityUnlocked, GestureAvailabilityLocked:
	default:
		return GetGesturesResult{}, fmt.Errorf(
			"availabilityFilter must be %q, %q or empty; got %q",
			GestureAvailabilityUnlocked, GestureAvailabilityLocked, availabilityFilter)
	}

	stored, err := engine.GetGestures(saveSessionID, characterID)
	if err != nil {
		return GetGesturesResult{}, err
	}

	// An inactive slot leaves the set empty, so every catalog gesture stays
	// locked without its GestureGameData ever being located or read.
	unlocked := make(map[uint32]struct{}, len(stored.Slots))
	for _, raw := range stored.Slots {
		if raw == gestureEmptySentinel || raw == gestureZeroRecord {
			continue
		}
		unlocked[raw] = struct{}{}
	}

	// Every gesture slot GameCatalog declares, in the deterministic order the
	// result uses: category first, then name, then the canonical slotID as the
	// tie-breaker between two slots of one resource. The unlock state is not part
	// of the order, so a filtered result keeps the order of the full one.
	//
	// The summaries select the gesture resources without deep-copying every
	// document in the catalog, and only the selected ones are then read in full.
	entries := make([]GestureEntry, 0)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindItem {
			continue
		}
		if !summary.FamilyKnown || summary.Family != schema.ItemFamilyGesture {
			continue
		}
		if summary.Key == "" {
			return GetGesturesResult{}, fmt.Errorf("a gesture resource carries no key")
		}

		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			return GetGesturesResult{}, fmt.Errorf("gesture %q: %w", summary.Key, err)
		}
		if resource.Item == nil {
			return GetGesturesResult{}, fmt.Errorf("gesture %q has no item document", summary.Key)
		}
		if resource.Item.Gesture == nil {
			return GetGesturesResult{}, fmt.Errorf("gesture %q has no gesture document", summary.Key)
		}

		for index, slot := range resource.Item.Gesture.Slots {
			if !slot.SlotID.Known {
				return GetGesturesResult{}, fmt.Errorf("gesture %q slot %d has no known slot ID", summary.Key, index)
			}
			if !slot.Name.Known || slot.Name.Value == "" {
				return GetGesturesResult{}, fmt.Errorf("gesture %q slot %d has no known name", summary.Key, index)
			}
			if !slot.Category.Known || slot.Category.Value == "" {
				return GetGesturesResult{}, fmt.Errorf("gesture %q slot %d has no known category", summary.Key, index)
			}
			entries = append(entries, GestureEntry{
				Kind:     resource.Kind,
				Key:      resource.Key,
				SlotID:   slot.SlotID.Value,
				Name:     slot.Name.Value,
				Category: slot.Category.Value,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Category != entries[j].Category {
			return entries[i].Category < entries[j].Category
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].SlotID < entries[j].SlotID
	})

	result := GetGesturesResult{
		SaveSessionID: stored.SaveSessionID,
		CharacterID:   stored.CharacterID,
		Active:        stored.Active,
		Gestures:      make([]GestureEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		_, present := unlocked[entry.SlotID]
		entry.Unlocked = present
		switch availabilityFilter {
		case GestureAvailabilityUnlocked:
			if !entry.Unlocked {
				continue
			}
		case GestureAvailabilityLocked:
			if entry.Unlocked {
				continue
			}
		}
		result.Gestures = append(result.Gestures, entry)
	}
	return result, nil
}
