/*
Endpoint: UpdateBuildTemplate
EndpointID: update_build_template
Purpose: Changes the metadata or contents of an existing Build Template with revision control.
How it works: The runtime handler delegates the mutation to the local templates store, which validates the canonical templateRevision, index, and existing payload, replaces the payload file atomically, and then replaces the _index.json file atomically with in-process rollback on failure.
Supported resource types: GameResource references.
Input variables: templateID, templateRevision, metadata, content.
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

// UpdateBuildTemplateEndpointID is the stable backend identifier of UpdateBuildTemplate.
const UpdateBuildTemplateEndpointID = "update_build_template"

// UpdateBuildTemplateDefinition describes the public mutation contract.
var UpdateBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "UpdateBuildTemplate",
	ID:                         UpdateBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"templateID", "templateRevision", "metadata", "content"},
	Description:                "Changes the metadata or contents of an existing Build Template with revision control.",
})

// UpdateBuildTemplateMetadata contains the editable metadata fields for UpdateBuildTemplate.
type UpdateBuildTemplateMetadata = buildtemplates.TemplateMetadataUpdate

// UpdateBuildTemplateRequest is the typed request body for UpdateBuildTemplate.
type UpdateBuildTemplateRequest struct {
	TemplateRevision string                       `json:"templateRevision"`
	Metadata         *UpdateBuildTemplateMetadata `json:"metadata,omitempty"`
	Content          *BuildTemplate               `json:"content,omitempty"`
}

// UpdateBuildTemplateResult is the typed receipt of UpdateBuildTemplate.
type UpdateBuildTemplateResult struct {
	TemplateID       string `json:"templateID"`
	TemplateRevision string `json:"templateRevision"`
}

// UpdateBuildTemplate updates an existing Build Template in the local library.
func UpdateBuildTemplate(
	store *buildtemplates.Store,
	templateID string,
	req UpdateBuildTemplateRequest,
) (UpdateBuildTemplateResult, error) {
	if store == nil {
		return UpdateBuildTemplateResult{}, errors.New("templates store is not available")
	}
	newRevision, err := store.UpdateTemplate(templateID, req.TemplateRevision, req.Metadata, req.Content)
	if err != nil {
		return UpdateBuildTemplateResult{}, err
	}
	return UpdateBuildTemplateResult{
		TemplateID:       templateID,
		TemplateRevision: newRevision,
	}, nil
}
