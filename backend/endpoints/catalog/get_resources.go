/*
Endpoint: GetResources
EndpointID: get_resources
Purpose: Zwraca stronicowaną listę zasobów filtrowaną między innymi po typie, rodzinie, capability i endpoint; służy do budowania pickerów bez osobnych getterów dla każdej kategorii.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: GameResource.
Input variables: resourceType, family, capability, endpointId, search, page, pageSize.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package catalog

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetResourcesEndpointID is the stable backend identifier of GetResources.
const GetResourcesEndpointID = "get_resources"

// GetResourcesDefinition describes the public getter contract.
var GetResourcesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetResources",
	ID:                         GetResourcesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource",
	SupportedResourceVariables: []string{"resourceType", "family", "capability", "endpointId", "search", "page", "pageSize"},
	Description:                "Zwraca stronicowaną listę zasobów filtrowaną między innymi po typie, rodzinie, capability i endpoint; służy do budowania pickerów bez osobnych getterów dla każdej kategorii.",
})
