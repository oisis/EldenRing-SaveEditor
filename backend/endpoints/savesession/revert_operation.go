/*
Endpoint: RevertOperation
EndpointID: revert_operation
Purpose: Selectively removes one logical operation by rebuilding from the session baseline.
How it works: SaveEngine replays every retained operation in order, validates the candidate and commits it atomically or rejects the whole request.
Supported resource types: —.
Input variables: saveSessionID, operationID, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the complete private snapshot rebuilt from baseline and retained operation deltas.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const RevertOperationEndpointID = "revert_operation"

var RevertOperationDefinition = contract.MustDefine(contract.Definition{
	Name: "RevertOperation", ID: RevertOperationEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"saveSessionID", "operationID", "expectedRevision"},
	Description: "Selectively removes one logical operation by rebuilding from the session baseline.",
})

type RevertOperationResult = saveengine.HistoryMutationResult

func RevertOperation(engine *saveengine.Engine, saveSessionID, operationID, expectedRevision string) (RevertOperationResult, error) {
	if engine == nil {
		return RevertOperationResult{}, errors.New("save engine is not available")
	}
	return engine.RevertOperation(saveSessionID, operationID, expectedRevision)
}
