/*
Endpoint: SetCharacterActive
EndpointID: set_character_active
Purpose: Zmienia stan aktywności slotu postaci.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: characterID, active, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetCharacterActiveEndpointID is the stable backend identifier of SetCharacterActive.
const SetCharacterActiveEndpointID = "set_character_active"

// SetCharacterActiveDefinition describes the public mutation contract.
var SetCharacterActiveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterActive",
	ID:                         SetCharacterActiveEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"characterID", "active", "expectedRevision"},
	Description:                "Zmienia stan aktywności slotu postaci.",
})
