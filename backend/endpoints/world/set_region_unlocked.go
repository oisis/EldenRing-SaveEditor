/*
Endpoint: SetRegionUnlocked
EndpointID: set_region_unlocked
Purpose: Sets the unlock state of a region.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: Region z grant.endpoint=set_region_unlocked.
Input variables: characterID, regionKind, regionKey, unlocked, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetRegionUnlockedEndpointID is the stable backend identifier of SetRegionUnlocked.
const SetRegionUnlockedEndpointID = "set_region_unlocked"

// SetRegionUnlockedDefinition describes the public mutation contract.
var SetRegionUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetRegionUnlocked",
	ID:                         SetRegionUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "Region z grant.endpoint=set_region_unlocked",
	SupportedResourceVariables: []string{"characterID", "regionKind", "regionKey", "unlocked", "expectedRevision"},
	Description:                "Sets the unlock state of a region.",
})
