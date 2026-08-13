/*
Endpoint: SetGestureUnlocked
EndpointID: set_gesture_unlocked
Purpose: Sets the unlock state of one logical gesture resource.
How it works: The runtime handler resolves the requested gesture through GameCatalog, validates its canonical save slot or the confirmed Ring of Miquella alias pair, and delegates one atomic GestureGameData mutation to SaveEngine under expectedRevision control.
Supported resource types: ItemDocument: Gesture.
Input variables: saveSessionID, characterID, gestureKind, gestureKey, unlocked, expectedRevision.
GameCatalog variables read: item.family and item.gesture.slots.slotID.
Save variables processed: the exact GestureGameData records associated with the requested gesture; unrelated, unknown and empty records are preserved.
Implementation status: implemented
*/
package world

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetGestureUnlockedEndpointID is the stable backend identifier of SetGestureUnlocked.
const SetGestureUnlockedEndpointID = "set_gesture_unlocked"

// SetGestureUnlockedDefinition describes the public mutation contract.
var SetGestureUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetGestureUnlocked",
	ID:                         SetGestureUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Gesture",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "gestureKind", "gestureKey", "unlocked", "expectedRevision"},
	Description:                "Sets the unlock state of one logical gesture resource.",
})

// SetGestureUnlockedResult reports the committed mutation in public catalog
// terms. The physical save slot ID remains internal.
type SetGestureUnlockedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	GestureKind   schema.ResourceKind `json:"gestureKind"`
	GestureKey    string              `json:"gestureKey"`
	Unlocked      bool                `json:"unlocked"`
}

// SetGestureUnlocked assigns the unlock state of one catalog gesture in an
// active character slot of an existing save session.
func SetGestureUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	gestureKind string,
	gestureKey string,
	unlocked bool,
	expectedRevision string,
) (SetGestureUnlockedResult, error) {
	if engine == nil {
		return SetGestureUnlockedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetGestureUnlockedResult{}, errors.New("game catalog is not available")
	}

	resource, err := gameCatalog.ResourceByKindAndKey(schema.ResourceKind(gestureKind), gestureKey)
	if err != nil {
		return SetGestureUnlockedResult{}, err
	}
	slotID, err := gestureMutationSlotID(resource)
	if err != nil {
		return SetGestureUnlockedResult{}, err
	}

	mutation, err := engine.SetGestureUnlocked(
		saveSessionID,
		characterID,
		slotID,
		unlocked,
		expectedRevision,
	)
	if err != nil {
		return SetGestureUnlockedResult{}, err
	}
	return SetGestureUnlockedResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		GestureKind:   resource.Kind,
		GestureKey:    resource.Key,
		Unlocked:      mutation.Unlocked,
	}, nil
}

func gestureMutationSlotID(resource schema.Resource) (uint32, error) {
	if resource.Item == nil {
		return 0, fmt.Errorf(
			"resource kind %q key %q has no item document", resource.Kind, resource.Key)
	}
	if !resource.Item.Family.Known {
		return 0, fmt.Errorf(
			"resource kind %q key %q has no known item family", resource.Kind, resource.Key)
	}
	if resource.Item.Family.Value != schema.ItemFamilyGesture {
		return 0, fmt.Errorf(
			"resource kind %q key %q has item family %q, want %q",
			resource.Kind, resource.Key, resource.Item.Family.Value, schema.ItemFamilyGesture)
	}
	if resource.Item.Gesture == nil {
		return 0, fmt.Errorf("gesture %q has no gesture document", resource.Key)
	}

	slots := resource.Item.Gesture.Slots
	if len(slots) == 0 {
		return 0, fmt.Errorf("gesture %q declares no save slot", resource.Key)
	}
	for index, slot := range slots {
		if !slot.SlotID.Known {
			return 0, fmt.Errorf("gesture %q slot %d has no known slot ID", resource.Key, index)
		}
		if slot.SlotID.Value == 0 || slot.SlotID.Value&1 == 0 || slot.SlotID.Value >= 0xFFFFFFFE {
			return 0, fmt.Errorf(
				"gesture %q slot %d has unsupported canonical slot ID %d",
				resource.Key, index, slot.SlotID.Value)
		}
	}

	if len(slots) == 1 {
		return slots[0].SlotID.Value, nil
	}
	if len(slots) == 2 {
		first, second := slots[0].SlotID.Value, slots[1].SlotID.Value
		if first == saveengine.GestureRingPreorderSlotID && second == saveengine.GestureRingEarnedSlotID ||
			first == saveengine.GestureRingEarnedSlotID && second == saveengine.GestureRingPreorderSlotID {
			return saveengine.GestureRingEarnedSlotID, nil
		}
	}
	return 0, fmt.Errorf(
		"gesture %q declares %d save slots; only one canonical slot or the alias pair 227/233 is supported",
		resource.Key, len(slots))
}
