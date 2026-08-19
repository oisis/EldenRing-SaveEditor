/*
Endpoint: SetCharacterStartingClass
EndpointID: set_character_starting_class
Purpose: Atomically changes a character's starting class, raising deficient attributes to the new class minima, recalculating level and SoulMemory, and synchronising both class bytes.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates the startingClassID, active slot and expected revision, applies the max(current, minimum) rule for each attribute, recalculates level and SoulMemory floor, and atomically writes the PlayerGameData block, the PlayerGameData class byte, the ProfileSummary level and the ProfileSummary class byte.
Supported resource types: —.
Input variables: saveSessionID, characterID, startingClassID, expectedRevision.
GameCatalog variables read: class resources (for starting class minima).
Save variables processed: the active-slot flag, the PlayerGameData starting class, attributes, level and TotalGetSoul, and the ProfileSummary level and starting class; HP/FP/SP, held runes, name, appearance, inventory and every unrelated field remain untouched.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetCharacterStartingClassEndpointID is the stable backend identifier of SetCharacterStartingClass.
const SetCharacterStartingClassEndpointID = "set_character_starting_class"

// SetCharacterStartingClassDefinition describes the public mutation contract.
var SetCharacterStartingClassDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterStartingClass",
	ID:                         SetCharacterStartingClassEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "startingClassID", "expectedRevision"},
	Description:                "Atomically changes a character's starting class, raises deficient attributes to the new class minima, recalculates level and SoulMemory, and synchronises both class copies.",
})

// SetCharacterStartingClassResult is the SaveEngine mutation receipt.
type SetCharacterStartingClassResult = saveengine.SetCharacterStartingClassResult

// SetCharacterStartingClass changes the starting class of one active character of an
// existing save session. SaveEngine owns the class minima, attribute raise, level,
// SoulMemory, revision and binary-format rules.
func SetCharacterStartingClass(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	startingClassID uint8,
	expectedRevision string,
) (SetCharacterStartingClassResult, error) {
	if engine == nil {
		return SetCharacterStartingClassResult{}, errors.New("save engine is not available")
	}
	return engine.SetCharacterStartingClass(
		saveSessionID, characterID, startingClassID, expectedRevision)
}
