/*
Endpoint: SetBellBearingUnlocked
EndpointID: set_bell_bearing_unlocked
Purpose: Ustawia stan odblokowania Bell Bearing.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: BellBearing z grant.endpoint=set_bell_bearing_unlocked.
Input variables: characterID, bellBearingKind, bellBearingKey, unlocked, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetBellBearingUnlockedEndpointID is the stable backend identifier of SetBellBearingUnlocked.
const SetBellBearingUnlockedEndpointID = "set_bell_bearing_unlocked"

// SetBellBearingUnlockedDefinition describes the public mutation contract.
var SetBellBearingUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetBellBearingUnlocked",
	ID:                         SetBellBearingUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: BellBearing z grant.endpoint=set_bell_bearing_unlocked",
	SupportedResourceVariables: []string{"characterID", "bellBearingKind", "bellBearingKey", "unlocked", "expectedRevision"},
	Description:                "Ustawia stan odblokowania Bell Bearing.",
})
