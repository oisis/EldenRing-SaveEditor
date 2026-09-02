/*
Endpoint: RemoveRecentFile
EndpointID: remove_recent_file
Purpose: Removes one exact path from Recent Files.
How it works: SaveEngine matches the stored path byte-for-byte and persists the remaining entries.
Supported resource types: —.
Input variables: path.
GameCatalog variables read: none.
Save variables processed: none; only host-local application metadata changes.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const RemoveRecentFileEndpointID = "remove_recent_file"

var RemoveRecentFileDefinition = contract.MustDefine(contract.Definition{
	Name: "RemoveRecentFile", ID: RemoveRecentFileEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"path"},
	Description: "Removes one exact path from Recent Files.",
})

type RemoveRecentFileResult = []saveengine.RecentFile

func RemoveRecentFile(engine *saveengine.Engine, path string) (RemoveRecentFileResult, error) {
	if engine == nil {
		return nil, errors.New("save engine is not available")
	}
	return engine.RemoveRecentFile(path)
}
