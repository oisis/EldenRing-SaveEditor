/*
Endpoint: GetOperationHistory
EndpointID: get_operation_history
Purpose: Returns the authoritative ordered Review Changes history of one save session.
How it works: Delegates the exact saveSessionID to SaveEngine and returns its safe operation projections without replay bytes.
Supported resource types: —.
Input variables: saveSessionID.
GameCatalog variables read: none.
Save variables read: session operation history metadata only; no save bytes leave SaveEngine.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const GetOperationHistoryEndpointID = "get_operation_history"

var GetOperationHistoryDefinition = contract.MustDefine(contract.Definition{
	Name: "GetOperationHistory", ID: GetOperationHistoryEndpointID, Kind: contract.Getter,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"saveSessionID"},
	Description: "Returns the authoritative ordered Review Changes history of one save session.",
})

type GetOperationHistoryResult = saveengine.OperationHistory

func GetOperationHistory(engine *saveengine.Engine, saveSessionID string) (GetOperationHistoryResult, error) {
	if engine == nil {
		return GetOperationHistoryResult{}, errors.New("save engine is not available")
	}
	return engine.GetOperationHistory(saveSessionID)
}
