/*
Endpoint: GetBuildTemplate
EndpointID: get_build_template
Purpose: Returns one complete Build Template from the local templates library.
How it works: The runtime handler resolves templateID through _index.json via the templates store, loads, decodes fail-closed and strictly validates the template payload, and returns the complete portable BuildTemplate document without modifying save or application state.
Supported resource types: GameResource references.
Input variables: templateID.
GameCatalog variables read: none; the getter is non-mutating and reads only the local templates library.
Save variables read: none; the getter is non-mutating and reads only the local templates library.
Implementation status: implemented; GetBuildTemplate is the runtime handler of this contract.
*/
package templates

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// GetBuildTemplateEndpointID is the stable backend identifier of GetBuildTemplate.
const GetBuildTemplateEndpointID = "get_build_template"

// GetBuildTemplateDefinition describes the public getter contract.
var GetBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBuildTemplate",
	ID:                         GetBuildTemplateEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"templateID"},
	Description:                "Returns one complete Build Template.",
})

// BuildTemplate is the portable on-wire and on-disk representation of a Build Template.
type BuildTemplate = buildtemplates.BuildTemplate

// GetBuildTemplate loads and returns one complete Build Template by its templateID.
func GetBuildTemplate(store *buildtemplates.Store, templateID string) (*BuildTemplate, error) {
	if store == nil {
		return nil, errors.New("templates store is not available")
	}
	return store.GetTemplate(templateID)
}
