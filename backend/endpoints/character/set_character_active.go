/*
Endpoint: SetCharacterActive
EndpointID: set_character_active
Purpose: Changes the activity state of a character slot.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates the expected revision and changes only the confirmed UserData10 activity flag.
Supported resource types: —.
Input variables: saveSessionID, characterID, active, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the UserData10 activity flag; reactivation also reads the confirmed statistics anchor and both character-name fields to reject a truly empty slot.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetCharacterActiveEndpointID is the stable backend identifier of SetCharacterActive.
const SetCharacterActiveEndpointID = "set_character_active"

// SetCharacterActiveDefinition describes the public mutation contract.
var SetCharacterActiveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterActive",
	ID:                         SetCharacterActiveEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "active", "expectedRevision"},
	Description:                "Changes the activity state of a character slot.",
})

// SetCharacterActiveResult is the SaveEngine mutation receipt.
type SetCharacterActiveResult = saveengine.SetCharacterActiveResult

// SetCharacterActive changes one physical slot's activity state. SaveEngine
// owns the slot, residual-data, revision and binary-format rules.
func SetCharacterActive(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	active bool,
	expectedRevision string,
) (SetCharacterActiveResult, error) {
	if engine == nil {
		return SetCharacterActiveResult{}, errors.New("save engine is not available")
	}
	return engine.SetCharacterActive(
		saveSessionID, characterID, active, expectedRevision)
}
