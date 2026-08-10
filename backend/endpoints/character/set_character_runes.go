/*
Endpoint: SetCharacterRunes
EndpointID: set_character_runes
Purpose: Sets the number of owned runes after validating its range.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, characterID, runes, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetCharacterRunesEndpointID is the stable backend identifier of SetCharacterRunes.
const SetCharacterRunesEndpointID = "set_character_runes"

// SetCharacterRunesDefinition describes the public mutation contract.
var SetCharacterRunesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterRunes",
	ID:                         SetCharacterRunesEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "runes", "expectedRevision"},
	Description:                "Sets the number of owned runes after validating its range.",
})
