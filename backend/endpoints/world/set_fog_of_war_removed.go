/*
Endpoint: SetFogOfWarRemoved
EndpointID: set_fog_of_war_removed
Purpose: Removes the global cosmetic Fog of War overlay of one character without exposing the raw map layout.
How it works: The runtime handler validates the request and delegates one atomic, revision-controlled bitfield fill to SaveEngine. No catalog resource is resolved, because the field carries no per-region identity.
Supported resource types: —.
Input variables: saveSessionID, characterID, removed, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the confirmed 2099-byte global Fog of War bitfield behind the UnlockedRegions list; the mutation validates every offset and bound before the first write and finishes with full success or rollback.
Implementation status: implemented
*/
package world

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetFogOfWarRemovedEndpointID is the stable backend identifier of SetFogOfWarRemoved.
const SetFogOfWarRemovedEndpointID = "set_fog_of_war_removed"

// SetFogOfWarRemovedDefinition describes the public mutation contract.
var SetFogOfWarRemovedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetFogOfWarRemoved",
	ID:                         SetFogOfWarRemovedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "removed", "expectedRevision"},
	Description:                "Removes the global cosmetic Fog of War overlay of one character without exposing the raw map layout.",
})

// SetFogOfWarRemovedResult reports the committed state and session revision.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetFogOfWarRemovedResult struct {
	saveengine.MutationReceipt
	CharacterID int  `json:"characterID"`
	Removed     bool `json:"removed"`
}

// SetFogOfWarRemoved removes the global Fog of War overlay of one character slot
// in an existing save session, reproducing the semantics SaveForge 1.5.8 and
// 1.6.8 shared: the confirmed bitfield is filled with 0xFF, in place.
//
// Only removed=true is accepted. The inverse has no confirmed contract — the
// bit-to-tile mapping of the field is unknown, so zeroing it would destroy the
// exploration state the save carries instead of restoring an earlier one — and
// SaveEngine rejects it before the session is opened or read and before any
// mutation.
//
// The operation is global and cosmetic: it names no map region, reads no
// GameCatalog, and never touches UnlockedRegions, event flags, Map Fragments,
// the DLC cover layer, Inventory or Storage.
func SetFogOfWarRemoved(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	removed bool,
	expectedRevision string,
) (SetFogOfWarRemovedResult, error) {
	if engine == nil {
		return SetFogOfWarRemovedResult{}, errors.New("save engine is not available")
	}

	mutation, err := engine.SetFogOfWarRemoved(saveSessionID, characterID, removed, expectedRevision)
	if err != nil {
		return SetFogOfWarRemovedResult{}, err
	}
	return SetFogOfWarRemovedResult{
		MutationReceipt: mutation.MutationReceipt,
		CharacterID:     mutation.CharacterID,
		Removed:         mutation.Removed,
	}, nil
}
