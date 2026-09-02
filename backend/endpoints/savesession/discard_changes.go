/*
Endpoint: DiscardChanges
EndpointID: discard_changes
Purpose: Restores the active session to its last loaded or successfully saved baseline.
How it works: SaveEngine validates and restores its baseline atomically, clears history and recovery, and emits one mutation receipt.
Supported resource types: —.
Input variables: saveSessionID, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the complete private session snapshot and operation history; no durable save is written.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const DiscardChangesEndpointID = "discard_changes"

var DiscardChangesDefinition = contract.MustDefine(contract.Definition{
	Name: "DiscardChanges", ID: DiscardChangesEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"saveSessionID", "expectedRevision"},
	Description: "Restores the active session to its last loaded or successfully saved baseline.",
})

type DiscardChangesResult = saveengine.DiscardChangesResult

func DiscardChanges(engine *saveengine.Engine, saveSessionID, expectedRevision string) (DiscardChangesResult, error) {
	if engine == nil {
		return DiscardChangesResult{}, errors.New("save engine is not available")
	}
	return engine.DiscardChanges(saveSessionID, expectedRevision)
}
