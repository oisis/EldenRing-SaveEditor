/*
Endpoint: SetCharacterGender
EndpointID: set_character_gender
Purpose: Ustawia typ ciała/płeć oraz wszystkie wymagane, potwierdzone zależności tej zmiany.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, gender, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetCharacterGenderEndpointID is the stable backend identifier of SetCharacterGender.
const SetCharacterGenderEndpointID = "set_character_gender"

// SetCharacterGenderDefinition describes the public mutation contract.
var SetCharacterGenderDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterGender",
	ID:                         SetCharacterGenderEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "gender", "expectedRevision"},
	Description:                "Ustawia typ ciała/płeć oraz wszystkie wymagane, potwierdzone zależności tej zmiany.",
})
