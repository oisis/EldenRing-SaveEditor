/*
Endpoint: SaveAs
EndpointID: save_as
Purpose: Persists a validated revision to an explicit target and makes that target the durable source of the session.
How it works: SaveEngine verifies the token, backs up an existing target, serializes, reload-validates, atomically replaces and verifies it, then updates the session baseline and source.
Supported resource types: —.
Input variables: saveSessionID, expectedRevision, validationToken, confirmWarnings, confirmBanRisk, target.
GameCatalog variables read: none.
Save variables processed: the complete private snapshot and explicit target.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const SaveAsEndpointID = "save_as"

var SaveAsDefinition = contract.MustDefine(contract.Definition{
	Name: "SaveAs", ID: SaveAsEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{
		"saveSessionID", "expectedRevision", "validationToken", "confirmWarnings", "confirmBanRisk", "target"},
	Description: "Persists a validated revision to an explicit target and makes that target the durable source of the session.",
})

type SaveAsResult = saveengine.SaveLifecycleResult

func SaveAs(
	engine *saveengine.Engine,
	saveSessionID, expectedRevision, validationToken string,
	confirmWarnings, confirmBanRisk bool,
	target string,
) (SaveAsResult, error) {
	if engine == nil {
		return SaveAsResult{}, errors.New("save engine is not available")
	}
	return engine.SaveAs(
		saveSessionID, expectedRevision, validationToken,
		confirmWarnings, confirmBanRisk, target)
}
