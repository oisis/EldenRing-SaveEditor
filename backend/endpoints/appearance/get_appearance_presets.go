/*
Endpoint: GetAppearancePresets
EndpointID: get_appearance_presets
Purpose: Returns available appearance presets with their metadata.
How it works: The runtime handler reads the appearance presets of the already loaded GameCatalog, which owns and validates backend/gamecatalog/data/presets/appearance.json together with the assets under backend/gamecatalog/data/assets/appearance, filters them by search and tags, and returns list metadata only. It reads no save and no file of its own, and it modifies nothing.
Supported resource types: —.
Input variables: search, tags.
GameCatalog variables read: appearance presets of presets/appearance.json — id, name, image, bodyType and tags of each preset.
Save variables read: none.
Implementation status: implemented
*/
package appearance

import (
	"errors"
	"fmt"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
)

// GetAppearancePresetsEndpointID is the stable backend identifier of GetAppearancePresets.
const GetAppearancePresetsEndpointID = "get_appearance_presets"

// GetAppearancePresetsDefinition describes the public getter contract.
var GetAppearancePresetsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetAppearancePresets",
	ID:                         GetAppearancePresetsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"search", "tags"},
	Description:                "Returns available appearance presets with their metadata.",
})

// AppearancePresetSummary is the list metadata of one preset. It deliberately
// carries no voice type, no model identifier and none of the faceShape, body or
// skin blobs: the full appearance configuration stays inside GameCatalog and is
// applied by ApplyAppearancePreset, not handed to a list consumer.
type AppearancePresetSummary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
	// BodyType is the public label of the stored numeric value: "Type A" for 1
	// and "Type B" for 0.
	BodyType string   `json:"bodyType"`
	Tags     []string `json:"tags"`
}

// GetAppearancePresetsResult is the typed result of GetAppearancePresets.
type GetAppearancePresetsResult struct {
	Presets []AppearancePresetSummary `json:"presets"`
}

// GetAppearancePresets reports the appearance presets the backend offers today,
// read from the loaded GameCatalog and reduced to the metadata a list needs.
//
// An empty search does not filter. A non-empty search is a case-insensitive
// substring match against the identifier and the name; it is never trimmed and
// never normalised, so a value with surrounding spaces matches only a preset
// that really contains them.
//
// A nil or empty tags list does not filter. Every supplied tag must be present
// on a preset for it to be returned, so the filter has AND semantics. Tags are
// compared exactly and case-sensitively, and an empty tag is rejected instead of
// being ignored.
//
// The presets keep the order of presets/appearance.json, which the catalog loads
// and validates once; no value is copied here and no file is read per call. The
// result is built per call, so no caller can reach catalog state or another
// caller's result through it.
func GetAppearancePresets(
	gameCatalog *gamecatalog.Catalog,
	search string,
	tags []string,
) (GetAppearancePresetsResult, error) {
	if gameCatalog == nil {
		return GetAppearancePresetsResult{}, errors.New("game catalog is not loaded")
	}
	for index, tag := range tags {
		if tag == "" {
			return GetAppearancePresetsResult{}, fmt.Errorf("tag %d must not be empty", index)
		}
	}
	// The catalog returns an independent copy, so a caller mutating one result
	// cannot affect another.
	presets, err := gameCatalog.AppearancePresets()
	if err != nil {
		return GetAppearancePresetsResult{}, err
	}

	loweredSearch := strings.ToLower(search)
	summaries := make([]AppearancePresetSummary, 0, len(presets))
	for _, preset := range presets {
		if search != "" &&
			!strings.Contains(strings.ToLower(preset.ID), loweredSearch) &&
			!strings.Contains(strings.ToLower(preset.Name), loweredSearch) {
			continue
		}

		matchesEveryTag := true
		for _, wanted := range tags {
			found := false
			for _, tag := range preset.Tags {
				if tag == wanted {
					found = true
					break
				}
			}
			if !found {
				matchesEveryTag = false
				break
			}
		}
		if !matchesEveryTag {
			continue
		}

		bodyType := "Type B"
		if preset.BodyType == 1 {
			bodyType = "Type A"
		}
		// ponytail: the catalog copy is already per-call, so its tag slice is
		// handed over instead of copied again.
		summaries = append(summaries, AppearancePresetSummary{
			ID:       preset.ID,
			Name:     preset.Name,
			Image:    preset.Image,
			BodyType: bodyType,
			Tags:     preset.Tags,
		})
	}

	return GetAppearancePresetsResult{Presets: summaries}, nil
}
