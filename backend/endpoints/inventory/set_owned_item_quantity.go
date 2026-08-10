/*
Endpoint: SetOwnedItemQuantity
EndpointID: set_owned_item_quantity
Purpose: Sets the quantity of an existing item instance while respecting stack and container limits.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument z capability stack.
Input variables: characterID, ownedItemID, quantity, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetOwnedItemQuantityEndpointID is the stable backend identifier of SetOwnedItemQuantity.
const SetOwnedItemQuantityEndpointID = "set_owned_item_quantity"

// SetOwnedItemQuantityDefinition describes the public mutation contract.
var SetOwnedItemQuantityDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetOwnedItemQuantity",
	ID:                         SetOwnedItemQuantityEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument z capability stack",
	SupportedResourceVariables: []string{"characterID", "ownedItemID", "quantity", "expectedRevision"},
	Description:                "Sets the quantity of an existing item instance while respecting stack and container limits.",
})
