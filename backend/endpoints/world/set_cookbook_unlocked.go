/*
Endpoint: SetCookbookUnlocked
EndpointID: set_cookbook_unlocked
Purpose: Sets the unlock state of a cookbook by mutating its confirmed event flag.
How it works: The runtime handler resolves the requested cookbook through GameCatalog, validates that it is a goods item declaring exactly one cookbook unlock, and delegates one atomic event flag mutation to SaveEngine under expectedRevision control.
Supported resource types: ItemDocument: Cookbook.
Input variables: saveSessionID, characterID, cookbookKind, cookbookKey, unlocked, expectedRevision.
GameCatalog variables read: item.family, item.unlocks (kind, eventFlagID, name, category).
Save variables processed: the event flag bit associated with the cookbook in the requested slot's bitfield; SaveEngine validates expectedRevision and finishes with full success or rollback.
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

// SetCookbookUnlockedEndpointID is the stable backend identifier of SetCookbookUnlocked.
const SetCookbookUnlockedEndpointID = "set_cookbook_unlocked"

// SetCookbookUnlockedDefinition describes the public mutation contract.
var SetCookbookUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCookbookUnlocked",
	ID:                         SetCookbookUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Cookbook",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "cookbookKind", "cookbookKey", "unlocked", "expectedRevision"},
	Description:                "Sets the unlock state of a cookbook through its confirmed event flag.",
})

// SetCookbookUnlockedResult reports the committed mutation in public catalog
// terms. SaveEngine supplies the session state; this endpoint adds the catalog
// identity it resolved without exposing the internal event flag identifier.
type SetCookbookUnlockedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	CookbookKind  schema.ResourceKind `json:"cookbookKind"`
	CookbookKey   string              `json:"cookbookKey"`
	Unlocked      bool                `json:"unlocked"`
}

// SetCookbookUnlocked sets or clears the unlock state of a catalog cookbook
// in one character slot of an existing save session.
func SetCookbookUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	cookbookKind string,
	cookbookKey string,
	unlocked bool,
	expectedRevision string,
) (SetCookbookUnlockedResult, error) {
	if engine == nil {
		return SetCookbookUnlockedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetCookbookUnlockedResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogCookbooks(gameCatalog)
	if err != nil {
		return SetCookbookUnlockedResult{}, err
	}

	var matched declaredCookbook
	found := false
	for _, cookbook := range declared {
		if cookbook.entry.Kind == schema.ResourceKind(cookbookKind) && cookbook.entry.Key == cookbookKey {
			matched = cookbook
			found = true
			break
		}
	}
	if !found {
		resource, err := gameCatalog.ResourceByKindAndKey(schema.ResourceKind(cookbookKind), cookbookKey)
		if err != nil {
			return SetCookbookUnlockedResult{}, err
		}
		if resource.Item == nil {
			return SetCookbookUnlockedResult{}, fmt.Errorf(
				"resource kind %q key %q has no item document", cookbookKind, cookbookKey)
		}
		matched, found, err = declaredCookbookFromResource(resource)
		if err != nil {
			return SetCookbookUnlockedResult{}, err
		}
		if !found {
			return SetCookbookUnlockedResult{}, fmt.Errorf(
				"resource kind %q key %q declares no cookbook unlock", cookbookKind, cookbookKey)
		}
	}

	mutation, err := engine.SetCookbookUnlocked(
		saveSessionID,
		characterID,
		matched.eventFlagID,
		unlocked,
		expectedRevision,
	)
	if err != nil {
		return SetCookbookUnlockedResult{}, err
	}
	return SetCookbookUnlockedResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		CookbookKind:  matched.entry.Kind,
		CookbookKey:   matched.entry.Key,
		Unlocked:      mutation.Unlocked,
	}, nil
}
