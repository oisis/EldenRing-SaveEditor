/*
Endpoint: SetInventoryOrder
EndpointID: set_inventory_order
Purpose: Ustawia pełną kolejność obsługiwanych instancji Inventory bez zmiany ich semantycznej zawartości.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument.
Input variables: characterID, orderedOwnedItemIDs, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetInventoryOrderEndpointID is the stable backend identifier of SetInventoryOrder.
const SetInventoryOrderEndpointID = "set_inventory_order"

// SetInventoryOrderDefinition describes the public mutation contract.
var SetInventoryOrderDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetInventoryOrder",
	ID:                         SetInventoryOrderEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"characterID", "orderedOwnedItemIDs", "expectedRevision"},
	Description:                "Ustawia pełną kolejność obsługiwanych instancji Inventory bez zmiany ich semantycznej zawartości.",
})
