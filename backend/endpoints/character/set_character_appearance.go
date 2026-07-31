/*
Endpoint: SetCharacterAppearance
EndpointID: set_character_appearance
Purpose: Waliduje i atomowo zapisuje kompletny model wyglądu postaci.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: characterID, appearance, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetCharacterAppearanceEndpointID is the stable backend identifier of SetCharacterAppearance.
const SetCharacterAppearanceEndpointID = "set_character_appearance"

// SetCharacterAppearanceDefinition describes the public mutation contract.
var SetCharacterAppearanceDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterAppearance",
	ID:                         SetCharacterAppearanceEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"characterID", "appearance", "expectedRevision"},
	Description:                "Waliduje i atomowo zapisuje kompletny model wyglądu postaci.",
})
