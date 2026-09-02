/*
Endpoint: RedoLastOperation
EndpointID: redo_last_operation
Purpose: Atomically reapplies the newest operation on the Redo stack.
How it works: Delegates revision validation, replay, validation and the mutation receipt to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the replay delta of the newest Redo operation in the private session snapshot.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const RedoLastOperationEndpointID = "redo_last_operation"

var RedoLastOperationDefinition = contract.MustDefine(contract.Definition{
	Name: "RedoLastOperation", ID: RedoLastOperationEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"saveSessionID", "expectedRevision"},
	Description: "Atomically reapplies the newest operation on the Redo stack.",
})

type RedoLastOperationResult = saveengine.HistoryMutationResult

func RedoLastOperation(engine *saveengine.Engine, saveSessionID, expectedRevision string) (RedoLastOperationResult, error) {
	if engine == nil {
		return RedoLastOperationResult{}, errors.New("save engine is not available")
	}
	return engine.RedoLastOperation(saveSessionID, expectedRevision)
}
