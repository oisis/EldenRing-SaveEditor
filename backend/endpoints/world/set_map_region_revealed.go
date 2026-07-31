/*
Endpoint: SetMapRegionRevealed
EndpointID: set_map_region_revealed
Purpose: Ustawia widoczność wskazanego regionu mapy bez ogólnego dostępu do surowych flag.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: MapRegion z grant.endpoint=set_map_region_revealed.
Input variables: characterID, mapRegionResourceID, revealed, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetMapRegionRevealedEndpointID is the stable backend identifier of SetMapRegionRevealed.
const SetMapRegionRevealedEndpointID = "set_map_region_revealed"

// SetMapRegionRevealedDefinition describes the public mutation contract.
var SetMapRegionRevealedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetMapRegionRevealed",
	ID:                         SetMapRegionRevealedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "MapRegion z grant.endpoint=set_map_region_revealed",
	SupportedResourceVariables: []string{"characterID", "mapRegionResourceID", "revealed", "expectedRevision"},
	Description:                "Ustawia widoczność wskazanego regionu mapy bez ogólnego dostępu do surowych flag.",
})
