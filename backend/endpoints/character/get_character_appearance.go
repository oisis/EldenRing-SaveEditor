/*
Endpoint: GetCharacterAppearance
EndpointID: get_character_appearance
Purpose: Zwraca kompletny, typowany model wyglądu postaci.
How it works: The runtime handler passes saveSessionID and characterID to SaveEngine, which reads one slot of the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none required by the current contract.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the confirmed raw gender, voice type and appearance block of its slot data; the getter is non-mutating and computes no value.
Implementation status: implemented.
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetCharacterAppearanceEndpointID is the stable backend identifier of GetCharacterAppearance.
const GetCharacterAppearanceEndpointID = "get_character_appearance"

// GetCharacterAppearanceDefinition describes the public getter contract.
var GetCharacterAppearanceDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCharacterAppearance",
	ID:                         GetCharacterAppearanceEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Zwraca kompletny, typowany model wyglądu postaci.",
})

// GetCharacterAppearanceResult is the typed result of GetCharacterAppearance.
// The shape is owned by SaveEngine, so the endpoint neither reshapes nor
// duplicates it. Every value stays exactly as stored in the save.
type GetCharacterAppearanceResult = saveengine.CharacterAppearance

// GetCharacterAppearance returns the raw appearance stored in one character slot
// of an existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID and characterID, reading the snapshot and
// deciding what an active, inactive or residual slot exposes belong to
// SaveEngine. The session must already exist; this endpoint never creates one,
// so it calls neither LoadSave nor any other endpoint, opens no file and returns
// no raw save byte.
func GetCharacterAppearance(engine *saveengine.Engine, saveSessionID string, characterID int) (GetCharacterAppearanceResult, error) {
	if engine == nil {
		return GetCharacterAppearanceResult{}, errors.New("save engine is not available")
	}
	return engine.GetCharacterAppearance(saveSessionID, characterID)
}
