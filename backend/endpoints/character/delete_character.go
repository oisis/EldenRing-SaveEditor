/*
Endpoint: DeleteCharacter
EndpointID: delete_character
Purpose: Atomically deletes the character from the specified slot and clears only data owned by that slot.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates the slot and expected revision and atomically clears only the target slot's owned ranges.
Supported resource types: —.
Input variables: saveSessionID, characterID, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the target slot's 0x280000-byte data block, one UserData10 activity flag and the complete 0x24C-byte ProfileSummary; adjacent slots and PC MD5 prefixes remain untouched.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// DeleteCharacterEndpointID is the stable backend identifier of DeleteCharacter.
const DeleteCharacterEndpointID = "delete_character"

// DeleteCharacterDefinition describes the public mutation contract.
var DeleteCharacterDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeleteCharacter",
	ID:                         DeleteCharacterEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "expectedRevision"},
	Description:                "Atomically deletes the character from the specified slot and clears only data owned by that slot.",
})

// DeleteCharacterResult is the SaveEngine deletion receipt.
type DeleteCharacterResult = saveengine.DeleteCharacterResult

// DeleteCharacter permanently clears one active or residual physical slot.
// SaveEngine owns every occupancy, revision and binary-format rule.
func DeleteCharacter(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	expectedRevision string,
) (DeleteCharacterResult, error) {
	if engine == nil {
		return DeleteCharacterResult{}, errors.New("save engine is not available")
	}
	return engine.DeleteCharacter(saveSessionID, characterID, expectedRevision)
}
