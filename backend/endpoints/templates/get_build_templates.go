/*
Endpoint: GetBuildTemplates
EndpointID: get_build_templates
Purpose: Zwraca bibliotekę Build Templates bez wczytywania pełnej zawartości każdego szablonu.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: GameResource references.
Input variables: search, tags, page, pageSize.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package templates

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetBuildTemplatesEndpointID is the stable backend identifier of GetBuildTemplates.
const GetBuildTemplatesEndpointID = "get_build_templates"

// GetBuildTemplatesDefinition describes the public getter contract.
var GetBuildTemplatesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBuildTemplates",
	ID:                         GetBuildTemplatesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"search", "tags", "page", "pageSize"},
	Description:                "Zwraca bibliotekę Build Templates bez wczytywania pełnej zawartości każdego szablonu.",
})
