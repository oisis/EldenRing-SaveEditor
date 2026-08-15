/*
Endpoint: SetSummoningPoolActivated
EndpointID: set_summoning_pool_activated
Purpose: Sets the activation state of a Summoning Pool.
How it works: The handler validates the complete curated summoning pool catalog, resolves the requested resource by its exact kind and key, and delegates one atomic event flag mutation to SaveEngine under expectedRevision control.
Supported resource types: SummoningPoolDocument.
Input variables: saveSessionID, characterID, summoningPoolKind, summoningPoolKey, activated, expectedRevision.
GameCatalog variables read: resource kind and key plus the summoningPool.activationEventFlagID of the requested pool.
Save variables processed: the activation event flag bit of the requested slot's bitfield; SaveEngine validates expectedRevision and finishes with full success or rollback.
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

// SetSummoningPoolActivatedEndpointID is the stable backend identifier of SetSummoningPoolActivated.
const SetSummoningPoolActivatedEndpointID = "set_summoning_pool_activated"

// SetSummoningPoolActivatedDefinition describes the public mutation contract.
var SetSummoningPoolActivatedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetSummoningPoolActivated",
	ID:                         SetSummoningPoolActivatedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "SummoningPoolDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "summoningPoolKind", "summoningPoolKey", "activated", "expectedRevision"},
	Description:                "Sets the activation state of a Summoning Pool.",
})

// SetSummoningPoolActivatedResult reports the committed state in public catalog
// terms. SaveEngine supplies the session state; this endpoint adds the catalog
// identity it resolved without exposing the internal activation event flag.
type SetSummoningPoolActivatedResult struct {
	SaveSessionID     string              `json:"saveSessionID"`
	SaveRevision      string              `json:"saveRevision"`
	CharacterID       int                 `json:"characterID"`
	SummoningPoolKind schema.ResourceKind `json:"summoningPoolKind"`
	SummoningPoolKey  string              `json:"summoningPoolKey"`
	Activated         bool                `json:"activated"`
}

// SetSummoningPoolActivated sets or clears the activation state of one catalog
// summoning pool in a character slot of an existing save session. It changes
// that one event flag and nothing else, exactly like SaveForge 1.5.8 and 1.6.8.
func SetSummoningPoolActivated(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	summoningPoolKind string,
	summoningPoolKey string,
	activated bool,
	expectedRevision string,
) (SetSummoningPoolActivatedResult, error) {
	if engine == nil {
		return SetSummoningPoolActivatedResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetSummoningPoolActivatedResult{}, errors.New("game catalog is not available")
	}
	if summoningPoolKind != string(schema.ResourceKindSummoningPool) {
		return SetSummoningPoolActivatedResult{}, fmt.Errorf(
			"resource kind %q is not %q", summoningPoolKind, schema.ResourceKindSummoningPool)
	}

	// The shared resolver validates the whole curated list — a missing document
	// or two pools claiming one flag — before a save byte is touched. It covers
	// every summoning_pool resource, so a key it does not carry is unknown.
	declared, err := catalogSummoningPools(gameCatalog)
	if err != nil {
		return SetSummoningPoolActivatedResult{}, err
	}
	var matched declaredSummoningPool
	found := false
	for _, pool := range declared {
		if pool.entry.Key == summoningPoolKey {
			matched = pool
			found = true
			break
		}
	}
	if !found {
		return SetSummoningPoolActivatedResult{}, fmt.Errorf(
			"unknown resource key %q in kind %q", summoningPoolKey, summoningPoolKind)
	}

	mutation, err := engine.SetSummoningPoolActivated(
		saveSessionID,
		characterID,
		matched.eventFlagID,
		activated,
		expectedRevision,
	)
	if err != nil {
		return SetSummoningPoolActivatedResult{}, err
	}
	return SetSummoningPoolActivatedResult{
		SaveSessionID:     mutation.SaveSessionID,
		SaveRevision:      mutation.SaveRevision,
		CharacterID:       mutation.CharacterID,
		SummoningPoolKind: matched.entry.Kind,
		SummoningPoolKey:  matched.entry.Key,
		Activated:         mutation.Activated,
	}, nil
}
