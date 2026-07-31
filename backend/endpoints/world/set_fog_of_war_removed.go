/*
Endpoint: SetFogOfWarRemoved
EndpointID: set_fog_of_war_removed
Purpose: Ustawia stan odsłonięcia fog of war przez potwierdzoną operację domenową, bez ujawniania surowego układu mapy.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: MapRegion.
Input variables: characterID, removed, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetFogOfWarRemovedEndpointID is the stable backend identifier of SetFogOfWarRemoved.
const SetFogOfWarRemovedEndpointID = "set_fog_of_war_removed"

// SetFogOfWarRemovedDefinition describes the public mutation contract.
var SetFogOfWarRemovedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetFogOfWarRemoved",
	ID:                         SetFogOfWarRemovedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "MapRegion",
	SupportedResourceVariables: []string{"characterID", "removed", "expectedRevision"},
	Description:                "Ustawia stan odsłonięcia fog of war przez potwierdzoną operację domenową, bez ujawniania surowego układu mapy.",
})
