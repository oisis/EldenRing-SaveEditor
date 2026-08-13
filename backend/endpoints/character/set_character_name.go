/*
Endpoint: SetCharacterName
EndpointID: set_character_name
Purpose: Validates and sets the character name.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates the name and expected revision and atomically synchronizes PlayerGameData with UserData10 ProfileSummary.
Supported resource types: —.
Input variables: saveSessionID, characterID, name, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the active-slot flag and the two confirmed 16-unit UTF-16LE name fields in PlayerGameData and UserData10 ProfileSummary; both copies succeed together or the prior bytes are restored.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetCharacterNameEndpointID is the stable backend identifier of SetCharacterName.
const SetCharacterNameEndpointID = "set_character_name"

// SetCharacterNameDefinition describes the public mutation contract.
var SetCharacterNameDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterName",
	ID:                         SetCharacterNameEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "name", "expectedRevision"},
	Description:                "Validates and sets the character name.",
})

// SetCharacterNameResult is the SaveEngine mutation receipt. Character identity
// needs no catalog shape, so the endpoint does not duplicate or reshape it.
type SetCharacterNameResult = saveengine.SetCharacterNameResult

// SetCharacterName assigns one active character's name in an existing save
// session. SaveEngine owns every name, slot, revision and binary-format rule;
// this endpoint only rejects missing wiring and delegates one atomic mutation.
func SetCharacterName(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	name string,
	expectedRevision string,
) (SetCharacterNameResult, error) {
	if engine == nil {
		return SetCharacterNameResult{}, errors.New("save engine is not available")
	}
	return engine.SetCharacterName(
		saveSessionID, characterID, name, expectedRevision)
}
