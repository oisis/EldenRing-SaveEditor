/*
Endpoint: UndoLastOperation
EndpointID: undo_last_operation
Purpose: Atomically reverts the newest logical operation and moves it to Redo.
How it works: Delegates revision validation, replay, validation and the mutation receipt to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the replay delta of the newest operation in the private session snapshot.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const UndoLastOperationEndpointID = "undo_last_operation"

var UndoLastOperationDefinition = contract.MustDefine(contract.Definition{
	Name: "UndoLastOperation", ID: UndoLastOperationEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"saveSessionID", "expectedRevision"},
	Description: "Atomically reverts the newest logical operation and moves it to Redo.",
})

type UndoLastOperationResult = saveengine.HistoryMutationResult

func UndoLastOperation(engine *saveengine.Engine, saveSessionID, expectedRevision string) (UndoLastOperationResult, error) {
	if engine == nil {
		return UndoLastOperationResult{}, errors.New("save engine is not available")
	}
	return engine.UndoLastOperation(saveSessionID, expectedRevision)
}
