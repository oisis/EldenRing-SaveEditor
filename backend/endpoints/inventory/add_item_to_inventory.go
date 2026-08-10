/*
Endpoint: AddItemToInventory
EndpointID: add_item_to_inventory
Purpose: Adds the specified resource or variant to Inventory after validating addToInventory, capacity, relations, and the complete mutation plan.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument z addToInventory.enabled=true.
Input variables: characterID, kind, key, variantID, quantity, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// AddItemToInventoryEndpointID is the stable backend identifier of AddItemToInventory.
const AddItemToInventoryEndpointID = "add_item_to_inventory"

// AddItemToInventoryDefinition describes the public mutation contract.
var AddItemToInventoryDefinition = contract.MustDefine(contract.Definition{
	Name:                       "AddItemToInventory",
	ID:                         AddItemToInventoryEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument z addToInventory.enabled=true",
	SupportedResourceVariables: []string{"characterID", "kind", "key", "variantID", "quantity", "expectedRevision"},
	Description:                "Adds the specified resource or variant to Inventory after validating addToInventory, capacity, relations, and the complete mutation plan.",
})
