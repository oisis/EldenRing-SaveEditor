/*
Endpoint: SetCharacterStats
EndpointID: set_character_stats
Purpose: Atomowo ustawia powiązany zestaw statystyk i przelicza wartości pochodne zgodnie z jednym kontraktem domenowym.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, characterID, attributes, levelPolicy, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetCharacterStatsEndpointID is the stable backend identifier of SetCharacterStats.
const SetCharacterStatsEndpointID = "set_character_stats"

// SetCharacterStatsDefinition describes the public mutation contract.
var SetCharacterStatsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterStats",
	ID:                         SetCharacterStatsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "attributes", "levelPolicy", "expectedRevision"},
	Description:                "Atomowo ustawia powiązany zestaw statystyk i przelicza wartości pochodne zgodnie z jednym kontraktem domenowym.",
})
