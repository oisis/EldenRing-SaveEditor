/*
Endpoint: CloneCharacter
EndpointID: clone_character
Purpose: Atomically clones a character into the specified empty slot after fully validating its dependencies.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: saveSessionID, sourceCharacterID, targetSlotID, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// CloneCharacterEndpointID is the stable backend identifier of CloneCharacter.
const CloneCharacterEndpointID = "clone_character"

// CloneCharacterDefinition describes the public mutation contract.
var CloneCharacterDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CloneCharacter",
	ID:                         CloneCharacterEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "sourceCharacterID", "targetSlotID", "expectedRevision"},
	Description:                "Atomically clones a character into the specified empty slot after fully validating its dependencies.",
})
