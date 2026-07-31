/*
Endpoint: GetRegions
EndpointID: get_regions
Purpose: Zwraca regiony i stan ich odblokowania.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: Region.
Input variables: characterID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetRegionsEndpointID is the stable backend identifier of GetRegions.
const GetRegionsEndpointID = "get_regions"

// GetRegionsDefinition describes the public getter contract.
var GetRegionsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetRegions",
	ID:                         GetRegionsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "Region",
	SupportedResourceVariables: []string{"characterID"},
	Description:                "Zwraca regiony i stan ich odblokowania.",
})
