/*
Endpoint: DeleteBuildTemplate
EndpointID: delete_build_template
Purpose: Deletes the specified Build Template from the library.
How it works: The runtime handler delegates the whole mutation to the local templates store, which validates the canonical templateRevision and the complete index before writing, commits the new _index.json atomically, and only then unlinks the payload the removed entry pointed at.
Supported resource types: GameResource references.
Input variables: templateID, templateRevision.
GameCatalog variables read: none; the mutation touches only the local templates library.
Save variables processed: none; the endpoint accepts no saveSessionID and no characterID, and never reads or writes save state.
Implementation status: implemented; the runtime handler delegates to buildtemplates.Store.
*/
package templates

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// DeleteBuildTemplateEndpointID is the stable backend identifier of DeleteBuildTemplate.
const DeleteBuildTemplateEndpointID = "delete_build_template"

// DeleteBuildTemplateDefinition describes the public mutation contract.
var DeleteBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeleteBuildTemplate",
	ID:                         DeleteBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"templateID", "templateRevision"},
	Description:                "Deletes the specified Build Template from the library.",
})

// DeleteBuildTemplateResult is the typed receipt of DeleteBuildTemplate. It
// names the deleted template and deliberately exposes no filename and no
// internal revision counter.
type DeleteBuildTemplateResult struct {
	TemplateID string `json:"templateID"`
}

// DeleteBuildTemplate removes one Build Template from the local library.
// Every error path returns the zero DeleteBuildTemplateResult.
func DeleteBuildTemplate(
	store *buildtemplates.Store,
	templateID string,
	templateRevision string,
) (DeleteBuildTemplateResult, error) {
	if store == nil {
		return DeleteBuildTemplateResult{}, errors.New("templates store is not available")
	}
	if err := store.DeleteTemplate(templateID, templateRevision); err != nil {
		return DeleteBuildTemplateResult{}, err
	}
	return DeleteBuildTemplateResult{TemplateID: templateID}, nil
}
