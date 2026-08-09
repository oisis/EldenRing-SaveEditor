/*
Endpoint: GetCharacterAppearance
EndpointID: get_character_appearance
Purpose: Zwraca kompletny, typowany model wyglądu postaci.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetCharacterAppearanceEndpointID is the stable backend identifier of GetCharacterAppearance.
const GetCharacterAppearanceEndpointID = "get_character_appearance"

// GetCharacterAppearanceDefinition describes the public getter contract.
var GetCharacterAppearanceDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCharacterAppearance",
	ID:                         GetCharacterAppearanceEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Zwraca kompletny, typowany model wyglądu postaci.",
})
