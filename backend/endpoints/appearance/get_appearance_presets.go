/*
Endpoint: GetAppearancePresets
EndpointID: get_appearance_presets
Purpose: Zwraca dostępne presety wyglądu wraz z ich metadanymi.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: search, tags.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package appearance

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetAppearancePresetsEndpointID is the stable backend identifier of GetAppearancePresets.
const GetAppearancePresetsEndpointID = "get_appearance_presets"

// GetAppearancePresetsDefinition describes the public getter contract.
var GetAppearancePresetsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetAppearancePresets",
	ID:                         GetAppearancePresetsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"search", "tags"},
	Description:                "Zwraca dostępne presety wyglądu wraz z ich metadanymi.",
})
