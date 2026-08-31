/*
Endpoint: LoadSave
EndpointID: load_save
Purpose: Loads a save from the specified source, identifies its platform and format, validates its structure, and creates a new session without modifying the input file.
How it works: The runtime handler passes the source path, the expected platform and the source kind to SaveEngine, which opens the local file read-only, recognises the container, validates it and creates the session. The source path is recorded on the session as metadata only; the session then reads exclusively from its private snapshot. LoadSave takes no expected revision and performs no atomic file mutation: it creates a new read-only session and leaves the input file untouched.
Supported resource types: —.
Input variables: source, expectedPlatform, sourceKind.
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
	SupportedResourceVariables: []string{"source", "expectedPlatform", "sourceKind"},
	Description:                "Loads a save from the specified source, identifies its platform and format, validates its structure, and creates a new session without modifying the input file.",
})

// LoadSaveResult is the typed result of LoadSave: the identifier of the created
// session, its platform, its container format, the exact source path and source
// kind it was created from, its save revision, which is always "0" for a freshly
// created session, and its unsaved-changes state, which is always false for one.
// The shape is owned by SaveEngine, so the endpoint neither reshapes nor
// duplicates it.
type LoadSaveResult = saveengine.SessionInfo

// LoadSave creates a new save session from the local file at source.
//
// The endpoint is thin: it holds no format rule, no magic, no offset, no
// platform validation and no source-kind rule of its own. Recognition, the
// expectedPlatform check, the sourceKind check and the structural validation all
// happen in SaveEngine, which never modifies the input file. When SaveEngine
// rejects the input, no session is created and the error is returned unchanged.
//
// sourceKind states what source is: "local" for a durable file the user owns and
// "temporary" for a working copy that is not one. It is required, and SaveEngine
// matches it exactly; the endpoint supplies no default for a caller that omits
// it, because a session must never claim an origin nobody stated.
func LoadSave(
	engine *saveengine.Engine,
	source string,
	expectedPlatform string,
	sourceKind string,
) (LoadSaveResult, error) {
	if engine == nil {
		return LoadSaveResult{}, errors.New("save engine is not available")
	}
	return engine.LoadSave(source, expectedPlatform, sourceKind)
}
