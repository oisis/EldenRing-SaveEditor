/*
Endpoint: ApplyAppearancePreset
EndpointID: apply_appearance_preset
Purpose: Stosuje zweryfikowany preset wyglądu przez tę samą operację domenową co SetCharacterAppearance.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: characterID, presetID, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package appearance

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// ApplyAppearancePresetEndpointID is the stable backend identifier of ApplyAppearancePreset.
const ApplyAppearancePresetEndpointID = "apply_appearance_preset"

// ApplyAppearancePresetDefinition describes the public mutation contract.
var ApplyAppearancePresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyAppearancePreset",
	ID:                         ApplyAppearancePresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"characterID", "presetID", "expectedRevision"},
	Description:                "Stosuje zweryfikowany preset wyglądu przez tę samą operację domenową co SetCharacterAppearance.",
})
