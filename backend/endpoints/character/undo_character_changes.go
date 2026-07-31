/*
Endpoint: UndoCharacterChanges
EndpointID: undo_character_changes
Purpose: Cofa ostatnią zatwierdzoną mutację wskazanej postaci, jeżeli punkt cofnięcia nadal odpowiada aktualnej sesji i rewizji.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: characterID, undoToken, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// UndoCharacterChangesEndpointID is the stable backend identifier of UndoCharacterChanges.
const UndoCharacterChangesEndpointID = "undo_character_changes"

// UndoCharacterChangesDefinition describes the public mutation contract.
var UndoCharacterChangesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "UndoCharacterChanges",
	ID:                         UndoCharacterChangesEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"characterID", "undoToken", "expectedRevision"},
	Description:                "Cofa ostatnią zatwierdzoną mutację wskazanej postaci, jeżeli punkt cofnięcia nadal odpowiada aktualnej sesji i rewizji.",
})
