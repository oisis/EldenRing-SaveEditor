/*
Endpoint: GetCharacterProfile
EndpointID: get_character_profile
Purpose: Returns one character profile: name, starting class identifier, level, play time, and raw body type identifier.
How it works: The runtime handler passes saveSessionID and characterID to SaveEngine, which reads one slot of the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none required by the current contract.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the confirmed name, level, secondsPlayed, gender and startingClassID; the getter is non-mutating.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetCharacterProfileEndpointID is the stable backend identifier of GetCharacterProfile.
const GetCharacterProfileEndpointID = "get_character_profile"

// GetCharacterProfileDefinition describes the public getter contract.
var GetCharacterProfileDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCharacterProfile",
	ID:                         GetCharacterProfileEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns one character profile: name, starting class identifier, level, play time, and raw body type identifier.",
})

// GetCharacterProfileResult is the typed result of GetCharacterProfile. The
// shape is owned by SaveEngine, so the endpoint neither reshapes nor duplicates
// it. StartingClassID and Gender stay raw identifiers, and SecondsPlayed stays a
// raw number of seconds.
type GetCharacterProfileResult = saveengine.CharacterProfile

// GetCharacterProfile returns the confirmed profile of one character slot of an
// existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID and characterID, reading the snapshot and
// deciding what an active, inactive or residual slot exposes belong to
// SaveEngine. The session must already exist; this endpoint never creates one,
// so it calls neither LoadSave nor any other endpoint, opens no file and returns
// no raw save byte.
func GetCharacterProfile(engine *saveengine.Engine, saveSessionID string, characterID int) (GetCharacterProfileResult, error) {
	if engine == nil {
		return GetCharacterProfileResult{}, errors.New("save engine is not available")
	}
	return engine.GetCharacterProfile(saveSessionID, characterID)
}
