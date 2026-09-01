/*
Endpoint: SetCharacterGender
EndpointID: set_character_gender
Purpose: Sets the body type or gender and every required confirmed dependency of that change.
How it works: The runtime handler resolves the confirmed default appearance preset for gender 0 or 1 from GameCatalog and delegates its complete appearance model to SaveEngine.SetCharacterGenderAppearance, the gender entry point of the single private appearance writer in SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, characterID, gender, expectedRevision.
GameCatalog variables read: the confirmed default Type A or Type B appearance preset and its complete appearance model.
Save variables processed: the active-slot flag, gender, voice type, the first confirmed FACE block and its two dependent sex-flag bytes through SaveEngine.SetCharacterGenderAppearance.
Implementation status: implemented
*/
package character

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetCharacterGenderEndpointID is the stable backend identifier of SetCharacterGender.
const SetCharacterGenderEndpointID = "set_character_gender"

// SetCharacterGenderDefinition describes the public mutation contract.
var SetCharacterGenderDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCharacterGender",
	ID:                         SetCharacterGenderEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "gender", "expectedRevision"},
	Description:                "Sets the body type or gender and every required confirmed dependency of that change.",
})

// SetCharacterGenderResult embeds the receipt of the committed mutation and
// adds the default preset selected for the body type and the complete
// appearance committed by SaveEngine. The receipt is the one the shared
// SaveEngine appearance writer produced, so operationKind is
// set_character_gender and never set_character_appearance.
type SetCharacterGenderResult struct {
	saveengine.MutationReceipt
	CharacterID int                                  `json:"characterID"`
	PresetID    string                               `json:"presetID"`
	Appearance  saveengine.CharacterAppearanceValues `json:"appearance"`
}

// SetCharacterGender applies the complete confirmed default appearance for one
// body type through the existing appearance mutation. It owns no binary rule.
func SetCharacterGender(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	gender uint8,
	expectedRevision string,
) (SetCharacterGenderResult, error) {
	if engine == nil {
		return SetCharacterGenderResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetCharacterGenderResult{}, errors.New("game catalog is not loaded")
	}

	preset, err := gameCatalog.DefaultAppearancePreset(gender)
	if err != nil {
		return SetCharacterGenderResult{}, err
	}
	modelIDs, err := gamecatalog.AppearanceModelIDs(preset)
	if err != nil {
		return SetCharacterGenderResult{}, err
	}
	appearance := saveengine.CharacterAppearanceValues{
		Gender:    preset.BodyType,
		VoiceType: preset.VoiceType,
		ModelIDs:  modelIDs,
		FaceShape: preset.FaceShape,
		Body:      preset.Body,
		Skin:      preset.Skin,
	}
	committed, err := engine.SetCharacterGenderAppearance(
		saveSessionID, characterID, appearance, expectedRevision)
	if err != nil {
		return SetCharacterGenderResult{}, err
	}
	return SetCharacterGenderResult{
		MutationReceipt: committed.MutationReceipt,
		CharacterID:     committed.CharacterID,
		PresetID:        preset.ID,
		Appearance:      committed.Appearance,
	}, nil
}
