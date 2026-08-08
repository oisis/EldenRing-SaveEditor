/*
Endpoint: GetMapRegions
EndpointID: get_map_regions
Purpose: Zwraca regiony mapy, ich widoczność oraz stan eksploracji.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: MapRegion.
Input variables: characterID, parentRegionKind, parentRegionKey.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetMapRegionsEndpointID is the stable backend identifier of GetMapRegions.
const GetMapRegionsEndpointID = "get_map_regions"

// GetMapRegionsDefinition describes the public getter contract.
var GetMapRegionsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetMapRegions",
	ID:                         GetMapRegionsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "MapRegion",
	SupportedResourceVariables: []string{"characterID", "parentRegionKind", "parentRegionKey"},
	Description:                "Zwraca regiony mapy, ich widoczność oraz stan eksploracji.",
})
