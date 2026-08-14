/*
Endpoint: ApplyAppearancePreset
EndpointID: apply_appearance_preset
Purpose: Applies a verified appearance preset through the same domain operation as SetCharacterAppearance.
How it works: The runtime handler resolves presetID exactly in the loaded GameCatalog, converts its verified UI model selections to raw PartsIds, and delegates the complete model to SaveEngine.SetCharacterAppearance.
Supported resource types: —.
Input variables: saveSessionID, characterID, presetID, expectedRevision.
GameCatalog variables read: the ID, bodyType, voiceType, eight model selections, faceShape, body and skin of one appearance preset.
Save variables processed: the confirmed appearance fields through SaveEngine.SetCharacterAppearance; no separate writer exists here.
Implementation status: implemented
*/
package appearance

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// ApplyAppearancePresetEndpointID is the stable backend identifier of ApplyAppearancePreset.
const ApplyAppearancePresetEndpointID = "apply_appearance_preset"

// ApplyAppearancePresetDefinition describes the public mutation contract.
var ApplyAppearancePresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyAppearancePreset",
	ID:                         ApplyAppearancePresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "presetID", "expectedRevision"},
	Description:                "Applies a verified appearance preset through the same domain operation as SetCharacterAppearance.",
})

// ApplyAppearancePresetResult reports the selected preset and the committed
// appearance assignment.
type ApplyAppearancePresetResult struct {
	SaveSessionID string                               `json:"saveSessionID"`
	SaveRevision  string                               `json:"saveRevision"`
	CharacterID   int                                  `json:"characterID"`
	PresetID      string                               `json:"presetID"`
	Appearance    saveengine.CharacterAppearanceValues `json:"appearance"`
}

// ApplyAppearancePreset resolves one catalog preset and applies it through the
// existing complete appearance mutation. It owns no binary-layout rule.
func ApplyAppearancePreset(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	presetID string,
	expectedRevision string,
) (ApplyAppearancePresetResult, error) {
	if engine == nil {
		return ApplyAppearancePresetResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return ApplyAppearancePresetResult{}, errors.New("game catalog is not loaded")
	}
	if presetID == "" {
		return ApplyAppearancePresetResult{}, errors.New("presetID is required")
	}

	presets, err := gameCatalog.AppearancePresets()
	if err != nil {
		return ApplyAppearancePresetResult{}, err
	}
	var selected *gamecatalog.AppearancePreset
	for index := range presets {
		if presets[index].ID == presetID {
			selected = &presets[index]
			break
		}
	}
	if selected == nil {
		return ApplyAppearancePresetResult{}, fmt.Errorf(
			"unknown appearance preset %q", presetID)
	}

	modelIDs, err := gamecatalog.AppearanceModelIDs(*selected)
	if err != nil {
		return ApplyAppearancePresetResult{}, err
	}
	appearance := saveengine.CharacterAppearanceValues{
		Gender:    selected.BodyType,
		VoiceType: selected.VoiceType,
		ModelIDs:  modelIDs,
		FaceShape: selected.FaceShape,
		Body:      selected.Body,
		Skin:      selected.Skin,
	}
	committed, err := engine.SetCharacterAppearance(
		saveSessionID, characterID, appearance, expectedRevision)
	if err != nil {
		return ApplyAppearancePresetResult{}, err
	}
	return ApplyAppearancePresetResult{
		SaveSessionID: committed.SaveSessionID,
		SaveRevision:  committed.SaveRevision,
		CharacterID:   committed.CharacterID,
		PresetID:      selected.ID,
		Appearance:    committed.Appearance,
	}, nil
}
