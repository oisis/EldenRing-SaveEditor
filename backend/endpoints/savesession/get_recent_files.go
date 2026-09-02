/*
Endpoint: GetRecentFiles
EndpointID: get_recent_files
Purpose: Returns up to ten durable local save files accepted by the desktop flow.
How it works: Delegates to SaveEngine's protected host-local Recent Files store.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none; this reads application metadata only.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const GetRecentFilesEndpointID = "get_recent_files"

var GetRecentFilesDefinition = contract.MustDefine(contract.Definition{
	Name: "GetRecentFiles", ID: GetRecentFilesEndpointID, Kind: contract.Getter,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{},
	Description: "Returns up to ten durable local save files accepted by the desktop flow.",
})

type GetRecentFilesResult = []saveengine.RecentFile

func GetRecentFiles(engine *saveengine.Engine) (GetRecentFilesResult, error) {
	if engine == nil {
		return nil, errors.New("save engine is not available")
	}
	return engine.GetRecentFiles()
}
