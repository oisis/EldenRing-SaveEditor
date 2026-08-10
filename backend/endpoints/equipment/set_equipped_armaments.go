/*
Endpoint: SetEquippedArmaments
EndpointID: set_equipped_armaments
Purpose: Atomically sets armaments in every hand slot and validates slot types and the existence of owned instances.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Weapon z capability equipment.
Input variables: characterID, slotAssignments, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetEquippedArmamentsEndpointID is the stable backend identifier of SetEquippedArmaments.
const SetEquippedArmamentsEndpointID = "set_equipped_armaments"

// SetEquippedArmamentsDefinition describes the public mutation contract.
var SetEquippedArmamentsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetEquippedArmaments",
	ID:                         SetEquippedArmamentsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Weapon z capability equipment",
	SupportedResourceVariables: []string{"characterID", "slotAssignments", "expectedRevision"},
	Description:                "Atomically sets armaments in every hand slot and validates slot types and the existence of owned instances.",
})
