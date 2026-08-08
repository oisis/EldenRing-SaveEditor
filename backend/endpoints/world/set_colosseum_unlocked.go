/*
Endpoint: SetColosseumUnlocked
EndpointID: set_colosseum_unlocked
Purpose: Ustawia stan odblokowania koloseum.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: Colosseum z grant.endpoint=set_colosseum_unlocked.
Input variables: characterID, colosseumKind, colosseumKey, unlocked, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetColosseumUnlockedEndpointID is the stable backend identifier of SetColosseumUnlocked.
const SetColosseumUnlockedEndpointID = "set_colosseum_unlocked"

// SetColosseumUnlockedDefinition describes the public mutation contract.
var SetColosseumUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetColosseumUnlocked",
	ID:                         SetColosseumUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "Colosseum z grant.endpoint=set_colosseum_unlocked",
	SupportedResourceVariables: []string{"characterID", "colosseumKind", "colosseumKey", "unlocked", "expectedRevision"},
	Description:                "Ustawia stan odblokowania koloseum.",
})
