/*
Endpoint: GetCharacterStats
EndpointID: get_character_stats
Purpose: Zwraca edytowalne statystyki postaci oraz wartości wyliczone przez backend.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: characterID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetCharacterStatsEndpointID is the stable backend identifier of GetCharacterStats.
const GetCharacterStatsEndpointID = "get_character_stats"

// GetCharacterStatsDefinition describes the public getter contract.
var GetCharacterStatsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCharacterStats",
	ID:                         GetCharacterStatsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"characterID"},
	Description:                "Zwraca edytowalne statystyki postaci oraz wartości wyliczone przez backend.",
})
