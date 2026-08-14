/*
Endpoint: CloneCharacter
EndpointID: clone_character
Purpose: Atomically clones a character into the specified empty slot after fully validating its dependencies.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates both physical slots, chooses a unique name and atomically writes only the target slot.
Supported resource types: —.
Input variables: saveSessionID, sourceCharacterID, targetSlotID, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: all slot activity flags and confirmed character names are read for validation and unique naming; the target slot's 0x280000-byte data block, one activity flag and complete 0x24C-byte ProfileSummary are written atomically.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// CloneCharacterEndpointID is the stable backend identifier of CloneCharacter.
const CloneCharacterEndpointID = "clone_character"

// CloneCharacterDefinition describes the public mutation contract.
var CloneCharacterDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CloneCharacter",
	ID:                         CloneCharacterEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "sourceCharacterID", "targetSlotID", "expectedRevision"},
	Description:                "Atomically clones a character into the specified empty slot after fully validating its dependencies.",
})

// CloneCharacterResult is the SaveEngine clone receipt.
type CloneCharacterResult = saveengine.CloneCharacterResult

// CloneCharacter copies one active character into one completely empty target
// slot. SaveEngine owns all validation, naming and binary-format rules.
func CloneCharacter(
	engine *saveengine.Engine,
	saveSessionID string,
	sourceCharacterID int,
	targetSlotID int,
	expectedRevision string,
) (CloneCharacterResult, error) {
	if engine == nil {
		return CloneCharacterResult{}, errors.New("save engine is not available")
	}
	return engine.CloneCharacter(
		saveSessionID, sourceCharacterID, targetSlotID, expectedRevision)
}
