/*
Endpoint: CreateBuildTemplate
EndpointID: create_build_template
Purpose: Tworzy nowy Build Template ze zwalidowanego, jawnego wyboru danych.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: sourceCharacterID, selection, name, description, tags.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package templates

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// CreateBuildTemplateEndpointID is the stable backend identifier of CreateBuildTemplate.
const CreateBuildTemplateEndpointID = "create_build_template"

// CreateBuildTemplateDefinition describes the public mutation contract.
var CreateBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CreateBuildTemplate",
	ID:                         CreateBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"sourceCharacterID", "selection", "name", "description", "tags"},
	Description:                "Tworzy nowy Build Template ze zwalidowanego, jawnego wyboru danych.",
})
