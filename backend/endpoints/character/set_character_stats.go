/*
Endpoint: SetCharacterStats
EndpointID: set_character_stats
Purpose: Atomically sets the eight character attributes and recalculates the values the save keeps consistent with them.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates the attribute ranges, the starting-class minima, the level policy and the expected revision, then atomically writes the attributes, the recalculated level and, when required, a raised TotalGetSoul.
Supported resource types: —.
Input variables: saveSessionID, characterID, attributes, levelPolicy, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the active-slot flag, the PlayerGameData starting class, attribute, level and TotalGetSoul fields, and the ProfileSummary level; HP/FP/SP, held runes and every unrelated field remain untouched.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetCharacterStatsEndpointID is the stable backend identifier of SetCharacterStats.
const SetCharacterStatsEndpointID = "set_character_stats"

// SetCharacterStatsDefinition describes the public mutation contract.
var SetCharacterStatsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterStats",
	ID:                         SetCharacterStatsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "attributes", "levelPolicy", "expectedRevision"},
	Description:                "Atomically sets the related group of statistics and recalculates derived values under one domain contract.",
})

// CharacterAttributes is the complete writable attribute set.
type CharacterAttributes = saveengine.CharacterAttributes

// SetCharacterStatsResult is the SaveEngine mutation receipt.
type SetCharacterStatsResult = saveengine.SetCharacterStatsResult

// SetCharacterStats assigns the eight attributes of one active character of an
// existing save session. SaveEngine owns the range, starting-class, level,
// SoulMemory, revision and binary-format rules.
func SetCharacterStats(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	attributes CharacterAttributes,
	levelPolicy string,
	expectedRevision string,
) (SetCharacterStatsResult, error) {
	if engine == nil {
		return SetCharacterStatsResult{}, errors.New("save engine is not available")
	}
	return engine.SetCharacterStats(
		saveSessionID, characterID, attributes, levelPolicy, expectedRevision)
}
