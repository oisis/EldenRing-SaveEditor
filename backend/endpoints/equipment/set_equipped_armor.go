/*
Endpoint: SetEquippedArmor
EndpointID: set_equipped_armor
Purpose: Atomically sets armor in every armor slot.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Armor z capability equipment.
Input variables: characterID, slotAssignments, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetEquippedArmorEndpointID is the stable backend identifier of SetEquippedArmor.
const SetEquippedArmorEndpointID = "set_equipped_armor"

// SetEquippedArmorDefinition describes the public mutation contract.
var SetEquippedArmorDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetEquippedArmor",
	ID:                         SetEquippedArmorEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Armor z capability equipment",
	SupportedResourceVariables: []string{"characterID", "slotAssignments", "expectedRevision"},
	Description:                "Atomically sets armor in every armor slot.",
})
