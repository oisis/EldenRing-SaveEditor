/*
Endpoint: LoadSave
EndpointID: load_save
Purpose: Loads a save from the specified source, identifies its platform and format, validates its structure, and creates a new session without modifying the input file.
How it works: The runtime handler passes the source path and the expected platform to SaveEngine, which opens the local file read-only, recognises the container, validates it and creates the session. LoadSave takes no expected revision and performs no atomic file mutation: it creates a new read-only session and leaves the input file untouched.
Supported resource types: —.
Input variables: source, expectedPlatform.
GameCatalog variables read: none required by the current contract.
Save variables read: the container recognition and structure validation performed by SaveEngine; no character, inventory or slot data is read at this stage.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// LoadSaveEndpointID is the stable backend identifier of LoadSave.
const LoadSaveEndpointID = "load_save"

// LoadSaveDefinition describes the public mutation contract.
var LoadSaveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "LoadSave",
	ID:                         LoadSaveEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"source", "expectedPlatform"},
	Description:                "Loads a save from the specified source, identifies its platform and format, validates its structure, and creates a new session without modifying the input file.",
})

// LoadSaveResult is the typed result of LoadSave: the identifier of the created
// session, its platform, its container format and its unsaved-changes state,
// which is always false for a freshly created read-only session. The shape is
// owned by SaveEngine, so the endpoint neither reshapes nor duplicates it.
type LoadSaveResult = saveengine.SessionInfo

// LoadSave creates a new read-only save session for the local file at source.
//
// The endpoint is thin: it holds no format rule, no magic, no offset and no
// platform validation of its own. Recognition, the expectedPlatform check and
// the structural validation all happen in SaveEngine, which never modifies the
// input file. When SaveEngine rejects the input, no session is created and the
// error is returned unchanged.
func LoadSave(
	engine *saveengine.Engine,
	source string,
	expectedPlatform string,
) (LoadSaveResult, error) {
	if engine == nil {
		return LoadSaveResult{}, errors.New("save engine is not available")
	}
	return engine.LoadSave(source, expectedPlatform)
}
