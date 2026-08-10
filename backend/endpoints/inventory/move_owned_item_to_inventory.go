/*
Endpoint: MoveOwnedItemToInventory
EndpointID: move_owned_item_to_inventory
Purpose: Atomically moves a specific instance from Storage to Inventory.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument dozwolony w Inventory.
Input variables: characterID, ownedItemID, targetPosition, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// MoveOwnedItemToInventoryEndpointID is the stable backend identifier of MoveOwnedItemToInventory.
const MoveOwnedItemToInventoryEndpointID = "move_owned_item_to_inventory"

// MoveOwnedItemToInventoryDefinition describes the public mutation contract.
var MoveOwnedItemToInventoryDefinition = contract.MustDefine(contract.Definition{
	Name:                       "MoveOwnedItemToInventory",
	ID:                         MoveOwnedItemToInventoryEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument dozwolony w Inventory",
	SupportedResourceVariables: []string{"characterID", "ownedItemID", "targetPosition", "expectedRevision"},
	Description:                "Atomically moves a specific instance from Storage to Inventory.",
})
