/*
Endpoint: SetColosseumUnlocked
EndpointID: set_colosseum_unlocked
Purpose: Sets the unlock state of a colosseum together with its confirmed matchmaking and map-marker flags.
How it works: The handler validates the complete colosseum catalog through the shared resolver, resolves the requested resource by its exact kind and key, hands SaveEngine the declared activation flag only, and delegates one atomic mutation under expectedRevision control; the confirmed four-flag set of that colosseum and the three global SET-only flags are SaveEngine's own closed rule and can never be named by a caller.
Supported resource types: ColosseumDocument.
Input variables: saveSessionID, characterID, colosseumKind, colosseumKey, unlocked, expectedRevision.
GameCatalog variables read: resource kind and key plus the colosseum.unlockEventFlagID of the requested colosseum.
Save variables processed: the four confirmed event flag bits of the requested colosseum and, on activation, the three global colosseum bits; SaveEngine validates expectedRevision and finishes with full success or rollback.
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

// SetColosseumUnlockedEndpointID is the stable backend identifier of SetColosseumUnlocked.
const SetColosseumUnlockedEndpointID = "set_colosseum_unlocked"

// SetColosseumUnlockedDefinition describes the public mutation contract.
var SetColosseumUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetColosseumUnlocked",
	ID:                         SetColosseumUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ColosseumDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "colosseumKind", "colosseumKey", "unlocked", "expectedRevision"},
	Description:                "Sets the unlock state of a colosseum together with its confirmed matchmaking and map-marker flags.",
})

// SetColosseumUnlockedResult reports the committed state in public catalog
// terms. SaveEngine supplies the session state; this endpoint adds the catalog
// identity it resolved without exposing any internal event flag.
type SetColosseumUnlockedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	ColosseumKind schema.ResourceKind `json:"colosseumKind"`
	ColosseumKey  string              `json:"colosseumKey"`
	Unlocked      bool                `json:"unlocked"`
}

// SetColosseumUnlocked sets or clears the unlock state of one catalog colosseum
// in a character slot of an existing save session, reproducing the semantics of
// SaveForge 1.5.8 and 1.6.8: activation writes the four confirmed flags of that
// arena plus the three global SET-only flags, deactivation clears those four
// flags only. The physical gate state in WorldGeom is not an event flag and is
// not touched, and no summoning pool, grace, item or region changes.
func SetColosseumUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	colosseumKind string,
	colosseumKey string,
	unlocked bool,
	expectedRevision string,
) (SetColosseumUnlockedResult, error) {
	if engine == nil {
		return SetColosseumUnlockedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetColosseumUnlockedResult{}, errors.New("game catalog is not available")
	}
	if colosseumKind != string(schema.ResourceKindColosseum) {
		return SetColosseumUnlockedResult{}, fmt.Errorf(
			"resource kind %q is not %q", colosseumKind, schema.ResourceKindColosseum)
	}

	// The shared resolver validates the whole colosseum list — a missing document
	// or two colosseums claiming one flag — before a save byte is touched. It
	// covers every colosseum resource, so a key it does not carry is unknown.
	declared, err := catalogColosseums(gameCatalog)
	if err != nil {
		return SetColosseumUnlockedResult{}, err
	}
	var matched declaredColosseum
	found := false
	for _, colosseum := range declared {
		if colosseum.entry.Key == colosseumKey {
			matched = colosseum
			found = true
			break
		}
	}
	if !found {
		return SetColosseumUnlockedResult{}, fmt.Errorf(
			"unknown resource key %q in kind %q", colosseumKey, colosseumKind)
	}

	mutation, err := engine.SetColosseumUnlocked(
		saveSessionID,
		characterID,
		matched.eventFlagID,
		unlocked,
		expectedRevision,
	)
	if err != nil {
		return SetColosseumUnlockedResult{}, err
	}
	return SetColosseumUnlockedResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		ColosseumKind: matched.entry.Kind,
		ColosseumKey:  matched.entry.Key,
		Unlocked:      mutation.Unlocked,
	}, nil
}
