/*
Endpoint: SetCharacterStartingClass
EndpointID: set_character_starting_class
Purpose: Atomically changes a character's starting class as a confirmed destructive build reset, replacing the eight attributes and the level with the base values of the target class and synchronising both class bytes.
How it works: The runtime handler delegates the complete request to SaveEngine, which requires confirmReset to be true, validates the startingClassID, active slot and expected revision, copies the eight base attributes and the base level of the target class straight from its GameCatalog class document, and atomically writes the PlayerGameData block, the PlayerGameData class byte, the ProfileSummary level and the ProfileSummary class byte. SoulMemory and the held runes are preserved unchanged.
Supported resource types: —.
Input variables: saveSessionID, characterID, startingClassID, confirmReset, expectedRevision.
GameCatalog variables read: class resources (base attributes and base level).
Save variables processed: the active-slot flag, the PlayerGameData starting class, attributes and level, and the ProfileSummary level and starting class; HP/FP/SP, TotalGetSoul, held runes, name, appearance, inventory and every unrelated field remain untouched.
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
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "startingClassID", "confirmReset", "expectedRevision"},
	Description:                "Atomically changes a character's starting class as a confirmed destructive reset, replacing the eight attributes and the level with the base values of the target class, preserving SoulMemory and held runes, and synchronising both class copies.",
})

// SetCharacterStartingClassResult is the SaveEngine mutation receipt.
type SetCharacterStartingClassResult = saveengine.SetCharacterStartingClassResult

// SetCharacterStartingClass changes the starting class of one active character of
// an existing save session. SaveEngine owns the class definition, the reset
// rules, the confirmReset gate, the revision and the binary-format rules.
//
// confirmReset is passed through untouched. This layer never defaults it: a
// caller that omits it reaches SaveEngine as false and is rejected there without
// any mutation.
func SetCharacterStartingClass(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	startingClassID uint8,
	confirmReset bool,
	expectedRevision string,
) (SetCharacterStartingClassResult, error) {
	if engine == nil {
		return SetCharacterStartingClassResult{}, errors.New("save engine is not available")
	}
	return engine.SetCharacterStartingClass(
		saveSessionID, characterID, startingClassID, confirmReset, expectedRevision)
}
