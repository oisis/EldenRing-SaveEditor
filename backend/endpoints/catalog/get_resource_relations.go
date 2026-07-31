/*
Endpoint: GetResourceRelations
EndpointID: get_resource_relations
Purpose: Zwraca relacje wychodzące i przychodzące wskazanego zasobu.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: GameResource.
Input variables: resourceID, relationType, direction.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package catalog

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetResourceRelationsEndpointID is the stable backend identifier of GetResourceRelations.
const GetResourceRelationsEndpointID = "get_resource_relations"

// GetResourceRelationsDefinition describes the public getter contract.
var GetResourceRelationsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetResourceRelations",
	ID:                         GetResourceRelationsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource",
	SupportedResourceVariables: []string{"resourceID", "relationType", "direction"},
	Description:                "Zwraca relacje wychodzące i przychodzące wskazanego zasobu.",
})
