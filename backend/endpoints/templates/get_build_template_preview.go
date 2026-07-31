/*
Endpoint: GetBuildTemplatePreview
EndpointID: get_build_template_preview
Purpose: Buduje niemutujący podgląd zastosowania template do wskazanej postaci.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: GameResource references.
Input variables: characterID, templateID, selection, options.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package templates

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetBuildTemplatePreviewEndpointID is the stable backend identifier of GetBuildTemplatePreview.
const GetBuildTemplatePreviewEndpointID = "get_build_template_preview"

// GetBuildTemplatePreviewDefinition describes the public getter contract.
var GetBuildTemplatePreviewDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBuildTemplatePreview",
	ID:                         GetBuildTemplatePreviewEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"characterID", "templateID", "selection", "options"},
	Description:                "Buduje niemutujący podgląd zastosowania template do wskazanej postaci.",
})
