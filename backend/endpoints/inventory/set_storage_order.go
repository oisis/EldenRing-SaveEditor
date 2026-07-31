/*
Endpoint: SetStorageOrder
EndpointID: set_storage_order
Purpose: Ustawia pełną kolejność obsługiwanych instancji Storage bez zmiany ich semantycznej zawartości.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument.
Input variables: characterID, orderedOwnedItemIDs, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetStorageOrderEndpointID is the stable backend identifier of SetStorageOrder.
const SetStorageOrderEndpointID = "set_storage_order"

// SetStorageOrderDefinition describes the public mutation contract.
var SetStorageOrderDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetStorageOrder",
	ID:                         SetStorageOrderEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"characterID", "orderedOwnedItemIDs", "expectedRevision"},
	Description:                "Ustawia pełną kolejność obsługiwanych instancji Storage bez zmiany ich semantycznej zawartości.",
})
