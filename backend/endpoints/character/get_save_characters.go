/*
Endpoint: GetSaveCharacters
EndpointID: get_save_characters
Purpose: Zwraca podsumowanie wszystkich slotów postaci wraz ze stabilnymi CharacterID, aktywnością i podstawowymi danymi prezentacyjnymi.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: saveSessionID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetSaveCharactersEndpointID is the stable backend identifier of GetSaveCharacters.
const GetSaveCharactersEndpointID = "get_save_characters"

// GetSaveCharactersDefinition describes the public getter contract.
var GetSaveCharactersDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetSaveCharacters",
	ID:                         GetSaveCharactersEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID"},
	Description:                "Zwraca podsumowanie wszystkich slotów postaci wraz ze stabilnymi CharacterID, aktywnością i podstawowymi danymi prezentacyjnymi.",
})
