/*
Endpoint: SetWhetbladeUnlocked
EndpointID: set_whetblade_unlocked
Purpose: Sets the unlock state of one Whetblade.
How it works: The handler validates all catalog Whetblade declarations, resolves the requested resource, and delegates one atomic Inventory-and-event-flags mutation to SaveEngine.
Supported resource types: goods ItemDocument declaring exactly one whetblade unlock and complete related event flags.
Input variables: saveSessionID, characterID, whetbladeKind, whetbladeKey, unlocked, expectedRevision.
GameCatalog variables read: item.family, item.gameID, the whetblade unlock and item.links.relatedEventFlags of kinds whetblade_related and aow_menu_unlock.
Save variables processed: the target goods record in Inventory common or key; the target main and related event flags; and the shared Ashes of War menu flag. Bundled acquisitions remain separate resources and are not mutated.
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

// SetWhetbladeUnlockedEndpointID is the stable backend identifier of SetWhetbladeUnlocked.
const SetWhetbladeUnlockedEndpointID = "set_whetblade_unlocked"

// SetWhetbladeUnlockedDefinition describes the public mutation contract.
var SetWhetbladeUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetWhetbladeUnlocked",
	ID:                         SetWhetbladeUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "goods ItemDocument declaring exactly one whetblade unlock and complete related event flags",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "whetbladeKind", "whetbladeKey", "unlocked", "expectedRevision"},
	Description:                "Sets the unlock state of one Whetblade.",
})

// SetWhetbladeUnlockedResult reports the committed state in public catalog
// terms without exposing game IDs or event flags.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetWhetbladeUnlockedResult struct {
	saveengine.MutationReceipt
	CharacterID   int                 `json:"characterID"`
	WhetbladeKind schema.ResourceKind `json:"whetbladeKind"`
	WhetbladeKey  string              `json:"whetbladeKey"`
	Unlocked      bool                `json:"unlocked"`
}

type whetbladeMutationDeclaration struct {
	declaredWhetblade
	relatedEventFlagIDs []uint32
	aowMenuEventFlagID  uint32
}

// SetWhetbladeUnlocked sets or clears one catalog Whetblade, its related
// affinity flags and its Inventory item. The shared menu flag follows the final
// state of the complete catalog-declared Whetblade set.
func SetWhetbladeUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	whetbladeKind string,
	whetbladeKey string,
	unlocked bool,
	expectedRevision string,
) (SetWhetbladeUnlockedResult, error) {
	if engine == nil {
		return SetWhetbladeUnlockedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetWhetbladeUnlockedResult{}, errors.New("game catalog is not available")
	}

	declarations, err := catalogWhetbladeMutations(gameCatalog)
	if err != nil {
		return SetWhetbladeUnlockedResult{}, err
	}
	targetIndex := -1
	for index, declaration := range declarations {
		if declaration.entry.Kind == schema.ResourceKind(whetbladeKind) &&
			declaration.entry.Key == whetbladeKey {
			targetIndex = index
			break
		}
	}
	if targetIndex < 0 {
		resource, err := gameCatalog.ResourceByKindAndKey(
			schema.ResourceKind(whetbladeKind), whetbladeKey)
		if err != nil {
			return SetWhetbladeUnlockedResult{}, err
		}
		if resource.Item == nil {
			return SetWhetbladeUnlockedResult{}, fmt.Errorf(
				"resource kind %q key %q has no item document", whetbladeKind, whetbladeKey)
		}
		if _, found, err := declaredWhetbladeFromResource(resource); err != nil {
			return SetWhetbladeUnlockedResult{}, err
		} else if !found {
			return SetWhetbladeUnlockedResult{}, fmt.Errorf(
				"resource kind %q key %q declares no whetblade unlock",
				whetbladeKind, whetbladeKey)
		}
		return SetWhetbladeUnlockedResult{}, fmt.Errorf(
			"resource kind %q key %q is not part of the validated Whetblade catalog",
			whetbladeKind, whetbladeKey)
	}

	target := declarations[targetIndex]
	others := make([]saveengine.WhetbladeState, 0, len(declarations)-1)
	for index, declaration := range declarations {
		if index == targetIndex {
			continue
		}
		others = append(others, saveengine.WhetbladeState{
			EventFlagID: declaration.eventFlagID,
			GameID:      declaration.gameID,
		})
	}
	mutation, err := engine.SetWhetbladeUnlocked(
		saveSessionID,
		characterID,
		saveengine.WhetbladeState{EventFlagID: target.eventFlagID, GameID: target.gameID},
		target.relatedEventFlagIDs,
		others,
		target.aowMenuEventFlagID,
		unlocked,
		expectedRevision,
	)
	if err != nil {
		return SetWhetbladeUnlockedResult{}, err
	}
	return SetWhetbladeUnlockedResult{
		MutationReceipt: mutation.MutationReceipt,
		CharacterID:     mutation.CharacterID,
		WhetbladeKind:   target.entry.Kind,
		WhetbladeKey:    target.entry.Key,
		Unlocked:        mutation.Unlocked,
	}, nil
}

func catalogWhetbladeMutations(
	gameCatalog *gamecatalog.Catalog,
) ([]whetbladeMutationDeclaration, error) {
	declared, err := catalogWhetblades(gameCatalog)
	if err != nil {
		return nil, err
	}
	if len(declared) == 0 {
		return nil, errors.New("game catalog declares no Whetblades")
	}
	if len(declared) != 6 {
		return nil, fmt.Errorf("game catalog declares %d Whetblades, want exactly 6", len(declared))
	}
	result := make([]whetbladeMutationDeclaration, 0, len(declared))
	flagOwners := make(map[uint32]string)
	var sharedMenuFlag uint32
	for _, whetblade := range declared {
		resource, err := gameCatalog.ResourceByKindAndKey(
			whetblade.entry.Kind, whetblade.entry.Key)
		if err != nil {
			return nil, err
		}
		mutation, err := whetbladeMutationFromResource(resource, whetblade)
		if err != nil {
			return nil, err
		}
		if sharedMenuFlag == 0 {
			sharedMenuFlag = mutation.aowMenuEventFlagID
		} else if mutation.aowMenuEventFlagID != sharedMenuFlag {
			return nil, fmt.Errorf(
				"whetblade %q declares AoW menu flag %d, want shared flag %d",
				whetblade.entry.Key, mutation.aowMenuEventFlagID, sharedMenuFlag)
		}
		for _, flagID := range append(
			[]uint32{mutation.eventFlagID}, mutation.relatedEventFlagIDs...) {
			if owner, duplicate := flagOwners[flagID]; duplicate {
				return nil, fmt.Errorf("whetblades %q and %q both own event flag %d",
					owner, mutation.entry.Key, flagID)
			}
			flagOwners[flagID] = mutation.entry.Key
		}
		result = append(result, mutation)
	}
	if owner, collision := flagOwners[sharedMenuFlag]; collision {
		return nil, fmt.Errorf("whetblade %q owns shared AoW menu flag %d", owner, sharedMenuFlag)
	}
	return result, nil
}

func whetbladeMutationFromResource(
	resource schema.Resource,
	declared declaredWhetblade,
) (whetbladeMutationDeclaration, error) {
	mutation := whetbladeMutationDeclaration{declaredWhetblade: declared}
	menuCount := 0
	for index, link := range resource.Item.Links.RelatedEventFlags {
		if !link.Kind.Known || !link.EventFlagID.Known {
			return whetbladeMutationDeclaration{}, fmt.Errorf(
				"whetblade %q related event flag %d has unknown data", resource.Key, index)
		}
		switch link.Kind.Value {
		case schema.RelatedEventFlagWhetblade:
			mutation.relatedEventFlagIDs = append(
				mutation.relatedEventFlagIDs, link.EventFlagID.Value)
		case schema.RelatedEventFlagAoWMenu:
			menuCount++
			mutation.aowMenuEventFlagID = link.EventFlagID.Value
		default:
			return whetbladeMutationDeclaration{}, fmt.Errorf(
				"whetblade %q related event flag %d has unsupported kind %q",
				resource.Key, index, link.Kind.Value)
		}
	}
	if len(mutation.relatedEventFlagIDs) == 0 {
		return whetbladeMutationDeclaration{}, fmt.Errorf(
			"whetblade %q declares no whetblade_related event flag", resource.Key)
	}
	if menuCount != 1 {
		return whetbladeMutationDeclaration{}, fmt.Errorf(
			"whetblade %q declares %d aow_menu_unlock event flags, want exactly one",
			resource.Key, menuCount)
	}
	return mutation, nil
}
