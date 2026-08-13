/*
Endpoint: SetCharacterRunes
EndpointID: set_character_runes
Purpose: Sets the number of owned runes after validating its range.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates the legal range and expected revision and atomically assigns the confirmed PlayerGameData field.
Supported resource types: —.
Input variables: saveSessionID, characterID, runes, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the active-slot flag and the four-byte held-runes field in PlayerGameData; the adjacent TotalGetSoul field and every unrelated field remain untouched.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

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

// SetCharacterRunesResult is the SaveEngine mutation receipt.
type SetCharacterRunesResult = saveengine.SetCharacterRunesResult

// SetCharacterRunes assigns held runes in one active character of an existing
// save session. SaveEngine owns the range, revision and binary-format rules.
func SetCharacterRunes(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	runes uint32,
	expectedRevision string,
) (SetCharacterRunesResult, error) {
	if engine == nil {
		return SetCharacterRunesResult{}, errors.New("save engine is not available")
	}
	return engine.SetCharacterRunes(
		saveSessionID, characterID, runes, expectedRevision)
}
