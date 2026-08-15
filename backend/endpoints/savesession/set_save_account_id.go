/*
Endpoint: SetSaveAccountID
EndpointID: set_save_account_id
Purpose: Sets the save owner identifier according to platform-specific rules.
How it works: The runtime handler delegates accountID and expectedRevision unchanged to SaveEngine, which owns the complete atomic PC-only operation.
Supported resource types: —.
Input variables: saveSessionID, accountID, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the global UserData10 account identifier and the own copy of every active character slot; SaveEngine resolves and range-checks every target before the first write, so the operation finishes with full success or rollback. PS4 is rejected before any field is read.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetSaveAccountIDEndpointID is the stable backend identifier of SetSaveAccountID.
const SetSaveAccountIDEndpointID = "set_save_account_id"

// SetSaveAccountIDDefinition describes the public mutation contract.
var SetSaveAccountIDDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetSaveAccountID",
	ID:                         SetSaveAccountIDEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "accountID", "expectedRevision"},
	Description:                "Sets the save owner identifier according to platform-specific rules.",
})

// SetSaveAccountIDResult is owned by SaveEngine; the endpoint does not reshape
// it. It reports the session and its new revision only, never the identifier.
type SetSaveAccountIDResult = saveengine.SetSaveAccountIDResult

// SetSaveAccountID sets the owner identifier of one save session. SaveEngine
// owns every platform, layout, revision and atomicity rule; the endpoint adds no
// parsing, normalisation or default of its own.
//
// accountID is a string carrying the canonical decimal representation of a
// uint64, because JavaScript and JSON lose the precision of large identifiers.
// It is passed through unchanged and is never echoed back in a result or error.
func SetSaveAccountID(
	engine *saveengine.Engine,
	saveSessionID string,
	accountID string,
	expectedRevision string,
) (SetSaveAccountIDResult, error) {
	if engine == nil {
		return SetSaveAccountIDResult{}, errors.New("save engine is not available")
	}
	return engine.SetSaveAccountID(saveSessionID, accountID, expectedRevision)
}
