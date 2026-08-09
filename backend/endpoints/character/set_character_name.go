/*
Endpoint: SetCharacterName
EndpointID: set_character_name
Purpose: Waliduje i ustawia nazwę postaci.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, characterID, name, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetCharacterNameEndpointID is the stable backend identifier of SetCharacterName.
const SetCharacterNameEndpointID = "set_character_name"

// SetCharacterNameDefinition describes the public mutation contract.
var SetCharacterNameDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterName",
	ID:                         SetCharacterNameEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "name", "expectedRevision"},
	Description:                "Waliduje i ustawia nazwę postaci.",
})
