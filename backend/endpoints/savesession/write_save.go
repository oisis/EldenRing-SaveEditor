/*
Endpoint: WriteSave
EndpointID: write_save
Purpose: Serializes, reloads, and validates the result, then atomically writes it to the explicitly specified destination.
How it works: The runtime handler delegates saveSessionID, expectedRevision, and target unchanged to SaveEngine, which owns the complete atomic operation.
Supported resource types: —.
Input variables: saveSessionID, expectedRevision, target.
GameCatalog variables read: none.
Save variables processed: the complete private session snapshot; SaveEngine serializes, reload-validates, and atomically persists it before committing the new revision.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// WriteSaveEndpointID is the stable backend identifier of WriteSave.
const WriteSaveEndpointID = "write_save"

// WriteSaveDefinition describes the public mutation contract.
var WriteSaveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "WriteSave",
	ID:                         WriteSaveEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "expectedRevision", "target"},
	Description:                "Serializes, reloads, and validates the result, then atomically writes it to the explicitly specified destination.",
})

// WriteSaveResult is owned by SaveEngine; the endpoint does not reshape it.
type WriteSaveResult = saveengine.WriteSaveResult

// WriteSave persists the current snapshot of one save session to target.
// SaveEngine owns every session, revision, validation and file-write rule.
func WriteSave(
	engine *saveengine.Engine,
	saveSessionID string,
	expectedRevision string,
	target string,
) (WriteSaveResult, error) {
	if engine == nil {
		return WriteSaveResult{}, errors.New("save engine is not available")
	}
	return engine.WriteSave(saveSessionID, expectedRevision, target)
}
