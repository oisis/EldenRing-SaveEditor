/*
Endpoint: SetCharacterAppearance
EndpointID: set_character_appearance
Purpose: Validates and atomically writes the complete character appearance model.
How it works: The runtime handler delegates the complete raw appearance model and expected revision to one atomic SaveEngine mutation.
Supported resource types: —.
Input variables: saveSessionID, characterID, appearance, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the active-slot flag, gender, voice type, the first confirmed FACE block, and its two dependent sex-flag bytes; all writes succeed together or the prior bytes are restored.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetCharacterAppearanceEndpointID is the stable backend identifier of SetCharacterAppearance.
const SetCharacterAppearanceEndpointID = "set_character_appearance"

// SetCharacterAppearanceDefinition describes the public mutation contract.
var SetCharacterAppearanceDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterAppearance",
	ID:                         SetCharacterAppearanceEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "appearance", "expectedRevision"},
	Description:                "Validates and atomically writes the complete character appearance model.",
})

// CharacterAppearanceValues is the complete raw appearance model accepted by
// SaveEngine. The endpoint neither duplicates nor reshapes it.
type CharacterAppearanceValues = saveengine.CharacterAppearanceValues

// SetCharacterAppearanceResult is the SaveEngine mutation receipt.
type SetCharacterAppearanceResult = saveengine.SetCharacterAppearanceResult

// SetCharacterAppearance replaces one active character's complete appearance.
// SaveEngine owns every value, revision and binary-format rule.
func SetCharacterAppearance(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	appearance CharacterAppearanceValues,
	expectedRevision string,
) (SetCharacterAppearanceResult, error) {
	if engine == nil {
		return SetCharacterAppearanceResult{}, errors.New("save engine is not available")
	}
	return engine.SetCharacterAppearance(
		saveSessionID, characterID, appearance, expectedRevision)
}
