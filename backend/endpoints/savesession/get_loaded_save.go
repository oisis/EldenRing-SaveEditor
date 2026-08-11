/*
Endpoint: GetLoadedSave
EndpointID: get_loaded_save
Purpose: Returns the identity, platform, format version, change state, and safe session metadata of a loaded save.
How it works: The runtime handler passes saveSessionID to SaveEngine and returns the safe metadata of the session SaveEngine already holds. It opens no file, reads no snapshot and parses no save data. The session must have been created earlier by LoadSave; validating and resolving the identifier belongs to SaveEngine alone.
Supported resource types: —.
Input variables: saveSessionID.
GameCatalog variables read: none required by the current contract.
Save variables read: none; only the session metadata SaveEngine exposes, and the call is non-mutating.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetLoadedSaveEndpointID is the stable backend identifier of GetLoadedSave.
const GetLoadedSaveEndpointID = "get_loaded_save"

// GetLoadedSaveDefinition describes the public getter contract.
var GetLoadedSaveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetLoadedSave",
	ID:                         GetLoadedSaveEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID"},
	Description:                "Returns the identity, platform, format version, change state, and safe session metadata of a loaded save.",
})

// GetLoadedSaveResult is the typed result of GetLoadedSave: the identifier of
// the session, its platform, its container format and its unsaved-changes
// state. The shape is owned by SaveEngine, so the endpoint neither reshapes nor
// duplicates it.
type GetLoadedSaveResult = saveengine.SessionInfo

// GetLoadedSave returns the safe metadata of an existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. It holds no session-identifier rule of its own, so validating and
// resolving saveSessionID happens in SaveEngine, which matches the identifier
// exactly. The session must already exist; this endpoint never creates one, so
// it neither calls LoadSave nor opens a file, and it reads no snapshot, no save
// byte and no character data.
func GetLoadedSave(engine *saveengine.Engine, saveSessionID string) (GetLoadedSaveResult, error) {
	if engine == nil {
		return GetLoadedSaveResult{}, errors.New("save engine is not available")
	}
	return engine.GetSessionInfo(saveSessionID)
}
