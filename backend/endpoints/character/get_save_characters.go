/*
Endpoint: GetSaveCharacters
EndpointID: get_save_characters
Purpose: Returns a summary of every character slot, including stable CharacterIDs, activity state, and basic presentation data.
How it works: The runtime handler passes saveSessionID to SaveEngine, which reads the ten slot summaries from the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: —.
Input variables: saveSessionID.
GameCatalog variables read: none required by the current contract.
Save variables read: the UserData10 slot activity flags and, for an active slot, the confirmed character name and level; the getter is non-mutating.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetSaveCharactersEndpointID is the stable backend identifier of GetSaveCharacters.
const GetSaveCharactersEndpointID = "get_save_characters"

// GetSaveCharactersDefinition describes the public getter contract.
var GetSaveCharactersDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetSaveCharacters",
	ID:                         GetSaveCharactersEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID"},
	Description:                "Returns a summary of every character slot, including stable CharacterIDs, activity state, and basic presentation data.",
})

// GetSaveCharactersResult is the typed result of GetSaveCharacters: the session
// that was read and one summary per physical slot. The shape is owned by
// SaveEngine, so the endpoint neither reshapes nor duplicates it.
type GetSaveCharactersResult = saveengine.SaveCharacters

// GetSaveCharacters returns the summaries of all ten character slots of an
// existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating and resolving saveSessionID, reading the snapshot and
// deciding what a slot exposes belong to SaveEngine. The session must already
// exist; this endpoint never creates one, so it calls neither LoadSave nor any
// other endpoint, opens no file and returns no raw save byte.
func GetSaveCharacters(engine *saveengine.Engine, saveSessionID string) (GetSaveCharactersResult, error) {
	if engine == nil {
		return GetSaveCharactersResult{}, errors.New("save engine is not available")
	}
	return engine.GetSaveCharacters(saveSessionID)
}
