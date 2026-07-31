/*
Endpoint: ApplyBuildTemplate
EndpointID: apply_build_template
Purpose: Buduje pełny plan i atomowo stosuje template do postaci albo nie wykonuje żadnej zmiany.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: characterID, templateID, selection, options, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package templates

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// ApplyBuildTemplateEndpointID is the stable backend identifier of ApplyBuildTemplate.
const ApplyBuildTemplateEndpointID = "apply_build_template"

// ApplyBuildTemplateDefinition describes the public mutation contract.
var ApplyBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyBuildTemplate",
	ID:                         ApplyBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"characterID", "templateID", "selection", "options", "expectedRevision"},
	Description:                "Buduje pełny plan i atomowo stosuje template do postaci albo nie wykonuje żadnej zmiany.",
})
