/*
Endpoint: DeleteBuildTemplate
EndpointID: delete_build_template
Purpose: Usuwa wskazany Build Template z biblioteki.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: templateID, templateRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package templates

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// DeleteBuildTemplateEndpointID is the stable backend identifier of DeleteBuildTemplate.
const DeleteBuildTemplateEndpointID = "delete_build_template"

// DeleteBuildTemplateDefinition describes the public mutation contract.
var DeleteBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeleteBuildTemplate",
	ID:                         DeleteBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"templateID", "templateRevision"},
	Description:                "Usuwa wskazany Build Template z biblioteki.",
})
