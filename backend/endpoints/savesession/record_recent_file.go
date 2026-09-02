/*
Endpoint: RecordRecentFile
EndpointID: record_recent_file
Purpose: Records one accepted durable local session in Recent Files.
How it works: SaveEngine reads the session's exact source metadata, moves that path to the front and persists at most ten entries.
Supported resource types: —.
Input variables: saveSessionID.
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

const RecordRecentFileEndpointID = "record_recent_file"

var RecordRecentFileDefinition = contract.MustDefine(contract.Definition{
	Name: "RecordRecentFile", ID: RecordRecentFileEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"saveSessionID"},
	Description: "Records one accepted durable local session in Recent Files.",
})

type RecordRecentFileResult = []saveengine.RecentFile

func RecordRecentFile(engine *saveengine.Engine, saveSessionID string) (RecordRecentFileResult, error) {
	if engine == nil {
		return nil, errors.New("save engine is not available")
	}
	return engine.RecordRecentFile(saveSessionID)
}
