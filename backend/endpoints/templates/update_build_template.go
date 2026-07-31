/*
Endpoint: UpdateBuildTemplate
EndpointID: update_build_template
Purpose: Zmienia metadane albo zawartość istniejącego Build Template z kontrolą rewizji.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: templateID, templateRevision, metadata, content.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package templates

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// UpdateBuildTemplateEndpointID is the stable backend identifier of UpdateBuildTemplate.
const UpdateBuildTemplateEndpointID = "update_build_template"

// UpdateBuildTemplateDefinition describes the public mutation contract.
var UpdateBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "UpdateBuildTemplate",
	ID:                         UpdateBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"templateID", "templateRevision", "metadata", "content"},
	Description:                "Zmienia metadane albo zawartość istniejącego Build Template z kontrolą rewizji.",
})
