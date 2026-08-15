/*
Endpoint: GetUndoState
EndpointID: get_undo_state
Purpose: Returns whether a safe undo point exists for the character and which operation it belongs to.
How it works: The runtime handler delegates to SaveEngine, which reports its single private undo point without reading or changing the save snapshot.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none.
Save variables read: none; the getter reads only the session's private undo point, revision and character index and mutates nothing.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetUndoStateEndpointID is the stable backend identifier of GetUndoState.
const GetUndoStateEndpointID = "get_undo_state"

// GetUndoStateDefinition describes the public getter contract.
var GetUndoStateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetUndoState",
	ID:                         GetUndoStateEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns whether a safe undo point exists for the character and which operation it belongs to.",
})

// CharacterUndoState is the SaveEngine undo-state report.
type CharacterUndoState = saveengine.CharacterUndoState

// GetUndoState reports whether the session holds a usable undo point for one
// character. SaveEngine owns the session, slot-index and revision rules; the
// endpoint adds none of its own and changes nothing.
func GetUndoState(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
) (CharacterUndoState, error) {
	if engine == nil {
		return CharacterUndoState{}, errors.New("save engine is not available")
	}
	return engine.GetUndoState(saveSessionID, characterID)
}
