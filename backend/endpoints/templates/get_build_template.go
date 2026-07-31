/*
Endpoint: GetBuildTemplate
EndpointID: get_build_template
Purpose: Zwraca jeden kompletny Build Template.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: GameResource references.
Input variables: templateID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package templates

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetBuildTemplateEndpointID is the stable backend identifier of GetBuildTemplate.
const GetBuildTemplateEndpointID = "get_build_template"

// GetBuildTemplateDefinition describes the public getter contract.
var GetBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBuildTemplate",
	ID:                         GetBuildTemplateEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"templateID"},
	Description:                "Zwraca jeden kompletny Build Template.",
})
