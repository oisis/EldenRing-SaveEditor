/*
Endpoint: SetGraceVisited
EndpointID: set_grace_visited
Purpose: Sets the visited state of a Site of Grace together with its required confirmed dependencies.
How it works: The handler validates the complete curated grace catalog, resolves the requested resource by its exact kind and key, hands SaveEngine the visit flag and the private door flag of that grace, and delegates one atomic mutation under expectedRevision control; the confirmed Gatefront companion set is applied by SaveEngine itself and can never be named by a caller.
Supported resource types: GraceDocument.
Input variables: saveSessionID, characterID, graceKind, graceKey, visited, expectedRevision.
GameCatalog variables read: resource kind and key plus the grace.visitEventFlagID and the private grace.doorEventFlagID of the requested grace.
Save variables processed: the visit event flag bit, the door event flag bit when the grace declares one, and the four confirmed Gatefront companion bits on activation; SaveEngine validates expectedRevision and finishes with full success or rollback.
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

// SetGraceVisitedEndpointID is the stable backend identifier of SetGraceVisited.
const SetGraceVisitedEndpointID = "set_grace_visited"

// SetGraceVisitedDefinition describes the public mutation contract.
var SetGraceVisitedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetGraceVisited",
	ID:                         SetGraceVisitedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GraceDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "graceKind", "graceKey", "visited", "expectedRevision"},
	Description:                "Sets the visited state of a Site of Grace together with its required confirmed dependencies.",
})

// SetGraceVisitedResult reports the committed state in public catalog terms.
// SaveEngine supplies the session state; this endpoint adds the catalog identity
// it resolved without exposing the internal visit or door flags.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetGraceVisitedResult struct {
	saveengine.MutationReceipt
	CharacterID int                 `json:"characterID"`
	GraceKind   schema.ResourceKind `json:"graceKind"`
	GraceKey    string              `json:"graceKey"`
	Visited     bool                `json:"visited"`
}

// SetGraceVisited sets or clears the visit state of one catalog grace in a
// character slot of an existing save session, reproducing the semantics of
// SaveForge 1.5.8 and 1.6.8: the visit flag and the optional door flag follow
// visited symmetrically, and the Gatefront companion flags are set on activation
// only. LastRestedGrace, the map, the regions and the inventory stay untouched.
func SetGraceVisited(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	graceKind string,
	graceKey string,
	visited bool,
	expectedRevision string,
) (SetGraceVisitedResult, error) {
	if engine == nil {
		return SetGraceVisitedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetGraceVisitedResult{}, errors.New("game catalog is not available")
	}
	if graceKind != string(schema.ResourceKindGrace) {
		return SetGraceVisitedResult{}, fmt.Errorf(
			"resource kind %q is not %q", graceKind, schema.ResourceKindGrace)
	}

	// The shared resolver validates the whole curated list — a missing document
	// or two graces claiming one visit flag — before a save byte is touched. It
	// covers every grace resource, so a key it does not carry is unknown.
	declared, err := catalogGraces(gameCatalog)
	if err != nil {
		return SetGraceVisitedResult{}, err
	}
	var matched declaredGrace
	found := false
	for _, grace := range declared {
		if grace.entry.Key == graceKey {
			matched = grace
			found = true
			break
		}
	}
	if !found {
		return SetGraceVisitedResult{}, fmt.Errorf(
			"unknown resource key %q in kind %q", graceKey, graceKind)
	}

	mutation, err := engine.SetGraceVisited(
		saveSessionID,
		characterID,
		matched.eventFlagID,
		matched.doorEventFlagID,
		visited,
		expectedRevision,
	)
	if err != nil {
		return SetGraceVisitedResult{}, err
	}
	return SetGraceVisitedResult{
		MutationReceipt: mutation.MutationReceipt,
		CharacterID:     mutation.CharacterID,
		GraceKind:       matched.entry.Kind,
		GraceKey:        matched.entry.Key,
		Visited:         mutation.Visited,
	}, nil
}
