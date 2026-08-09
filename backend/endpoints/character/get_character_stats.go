/*
Endpoint: GetCharacterStats
EndpointID: get_character_stats
Purpose: Zwraca surowe statystyki jednej postaci zapisane w save: osiem atrybutów, poziom oraz zapisane wartości HP, FP i SP wraz z ich wartościami maksymalnymi i bazowymi.
How it works: The runtime handler passes saveSessionID and characterID to SaveEngine, which reads one slot of the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none required by the current contract.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the confirmed raw attributes, level and HP/FP/SP values of its slot data; the getter is non-mutating and computes no value.
Implementation status: implemented.
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetCharacterStatsEndpointID is the stable backend identifier of GetCharacterStats.
const GetCharacterStatsEndpointID = "get_character_stats"

// GetCharacterStatsDefinition describes the public getter contract.
var GetCharacterStatsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCharacterStats",
	ID:                         GetCharacterStatsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Zwraca surowe statystyki jednej postaci zapisane w save: osiem atrybutów, poziom oraz zapisane wartości HP, FP i SP wraz z ich wartościami maksymalnymi i bazowymi.",
})

// GetCharacterStatsResult is the typed result of GetCharacterStats. The shape is
// owned by SaveEngine, so the endpoint neither reshapes nor duplicates it. Every
// numeric field stays the raw uint32 stored in the save.
type GetCharacterStatsResult = saveengine.CharacterStats

// GetCharacterStats returns the raw statistics stored in one character slot of
// an existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID and characterID, reading the snapshot and
// deciding what an active, inactive or residual slot exposes belong to
// SaveEngine. The session must already exist; this endpoint never creates one,
// so it calls neither LoadSave nor any other endpoint, opens no file and returns
// no raw save byte.
func GetCharacterStats(engine *saveengine.Engine, saveSessionID string, characterID int) (GetCharacterStatsResult, error) {
	if engine == nil {
		return GetCharacterStatsResult{}, errors.New("save engine is not available")
	}
	return engine.GetCharacterStats(saveSessionID, characterID)
}
