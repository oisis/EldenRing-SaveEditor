/*
Endpoint: CloseSave
EndpointID: close_save
Purpose: Closes an existing save session and releases its private snapshot.
How it works: The runtime handler passes saveSessionID to SaveEngine, which removes the session entry under its own lock. The endpoint opens no file, reads no snapshot and writes nothing; the source save is untouched. Validating and resolving the identifier belongs to SaveEngine alone.
Supported resource types: —.
Input variables: saveSessionID.
GameCatalog variables read: none required by the current contract.
Save variables processed: none; only the in-memory session entry is removed, and no save file is read or written.
Implementation status: implemented.
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// CloseSaveEndpointID is the stable backend identifier of CloseSave.
const CloseSaveEndpointID = "close_save"

// CloseSaveDefinition describes the public mutation contract.
var CloseSaveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CloseSave",
	ID:                         CloseSaveEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID"},
	Description:                "Closes an existing save session and releases its private snapshot.",
})

// CloseSave removes an existing save session from SaveEngine.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. It holds no session-identifier rule of its own, so validating and
// resolving saveSessionID happens in SaveEngine, which matches the identifier
// exactly. The session must already exist; this endpoint never creates one, so
// it neither calls LoadSave nor GetLoadedSave, and it opens, reads and writes
// no save file.
//
// Closing releases SaveEngine's reference to the session and its private
// snapshot. The memory is reclaimed by the ordinary garbage collector, not by
// an immediate manual operation.
func CloseSave(engine *saveengine.Engine, saveSessionID string) error {
	if engine == nil {
		return errors.New("save engine is not available")
	}
	return engine.CloseSession(saveSessionID)
}
