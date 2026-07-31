/*
Endpoint: GetResource
EndpointID: get_resource
Purpose: Zwraca pełny widok jednego zasobu wraz z capabilities, wariantami, relacjami, prezentacją i provenance.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: GameResource.
Input variables: resourceID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package catalog

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetResourceEndpointID is the stable backend identifier of GetResource.
const GetResourceEndpointID = "get_resource"

// GetResourceDefinition describes the public getter contract.
var GetResourceDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetResource",
	ID:                         GetResourceEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource",
	SupportedResourceVariables: []string{"resourceID"},
	Description:                "Zwraca pełny widok jednego zasobu wraz z capabilities, wariantami, relacjami, prezentacją i provenance.",
})
