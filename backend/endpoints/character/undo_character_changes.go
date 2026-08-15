/*
Endpoint: UndoCharacterChanges
EndpointID: undo_character_changes
Purpose: Reverts the last committed mutation of the specified character when the undo point still matches the current session and save revision.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates the expected revision, the undo token and the ownership of its single undo point, then atomically restores the captured ranges or leaves the session unchanged.
Supported resource types: —.
Input variables: saveSessionID, characterID, undoToken, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the character's slot data, its ProfileSummary and its UserData10 activity flag are restored from the undo point; every other byte and every other slot remain untouched.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// UndoCharacterChangesEndpointID is the stable backend identifier of UndoCharacterChanges.
const UndoCharacterChangesEndpointID = "undo_character_changes"

// UndoCharacterChangesDefinition describes the public mutation contract.
//
// undoToken is an opaque SaveEngine identifier, not a GameResource reference,
// so this endpoint supports no catalog resource type and reads no catalog data.
var UndoCharacterChangesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "UndoCharacterChanges",
	ID:                         UndoCharacterChangesEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "undoToken", "expectedRevision"},
	Description:                "Reverts the last committed mutation of the specified character when the undo point still matches the current session and save revision.",
})

// UndoCharacterChangesResult is the SaveEngine mutation receipt.
type UndoCharacterChangesResult = saveengine.UndoCharacterChangesResult

// UndoCharacterChanges reverts the last committed mutation of one character of
// an existing save session. SaveEngine owns the token, revision, atomicity and
// binary-format rules.
func UndoCharacterChanges(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	undoToken string,
	expectedRevision string,
) (UndoCharacterChangesResult, error) {
	if engine == nil {
		return UndoCharacterChangesResult{}, errors.New("save engine is not available")
	}
	return engine.UndoCharacterChanges(saveSessionID, characterID, undoToken, expectedRevision)
}
