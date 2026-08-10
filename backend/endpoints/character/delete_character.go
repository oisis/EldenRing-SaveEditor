/*
Endpoint: DeleteCharacter
EndpointID: delete_character
Purpose: Atomically deletes the character from the specified slot and clears only data owned by that slot.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// DeleteCharacterEndpointID is the stable backend identifier of DeleteCharacter.
const DeleteCharacterEndpointID = "delete_character"

// DeleteCharacterDefinition describes the public mutation contract.
var DeleteCharacterDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeleteCharacter",
	ID:                         DeleteCharacterEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "expectedRevision"},
	Description:                "Atomically deletes the character from the specified slot and clears only data owned by that slot.",
})
