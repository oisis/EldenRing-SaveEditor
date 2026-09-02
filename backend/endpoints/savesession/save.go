/*
Endpoint: Save
EndpointID: save
Purpose: Persists a validated revision to the durable local source of the active session.
How it works: SaveEngine verifies the validation token, creates a required backup, serializes, reload-validates, atomically replaces and verifies the target, then clears history.
Supported resource types: —.
Input variables: saveSessionID, expectedRevision, validationToken, confirmWarnings, confirmBanRisk.
GameCatalog variables read: none.
Save variables processed: the complete private snapshot and durable local source.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const SaveEndpointID = "save"

var SaveDefinition = contract.MustDefine(contract.Definition{
	Name: "Save", ID: SaveEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{
		"saveSessionID", "expectedRevision", "validationToken", "confirmWarnings", "confirmBanRisk"},
	Description: "Persists a validated revision to the durable local source of the active session.",
})

type SaveResult = saveengine.SaveLifecycleResult

func Save(
	engine *saveengine.Engine,
	saveSessionID, expectedRevision, validationToken string,
	confirmWarnings, confirmBanRisk bool,
) (SaveResult, error) {
	if engine == nil {
		return SaveResult{}, errors.New("save engine is not available")
	}
	return engine.Save(
		saveSessionID, expectedRevision, validationToken, confirmWarnings, confirmBanRisk)
}
