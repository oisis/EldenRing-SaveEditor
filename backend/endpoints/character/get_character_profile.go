/*
Endpoint: GetCharacterProfile
EndpointID: get_character_profile
Purpose: Zwraca profil jednej postaci: nazwę, klasę początkową, poziom, czas gry i inne dane identyfikacyjne.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetCharacterProfileEndpointID is the stable backend identifier of GetCharacterProfile.
const GetCharacterProfileEndpointID = "get_character_profile"

// GetCharacterProfileDefinition describes the public getter contract.
var GetCharacterProfileDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCharacterProfile",
	ID:                         GetCharacterProfileEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Zwraca profil jednej postaci: nazwę, klasę początkową, poziom, czas gry i inne dane identyfikacyjne.",
})
