/*
Endpoint: ClearRecentFiles
EndpointID: clear_recent_files
Purpose: Clears the host-local Recent Files history.
How it works: SaveEngine atomically persists an empty list and touches no save session or save file.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const ClearRecentFilesEndpointID = "clear_recent_files"

var ClearRecentFilesDefinition = contract.MustDefine(contract.Definition{
	Name: "ClearRecentFiles", ID: ClearRecentFilesEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{},
	Description: "Clears the host-local Recent Files history.",
})

func ClearRecentFiles(engine *saveengine.Engine) error {
	if engine == nil {
		return errors.New("save engine is not available")
	}
	return engine.ClearRecentFiles()
}
