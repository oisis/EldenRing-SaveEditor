/*
Endpoint: AddItemToStorage
EndpointID: add_item_to_storage
Purpose: Adds the specified resource or variant to Storage after validating addToStorage, capacity, relations, and the complete mutation plan.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument z addToStorage.enabled=true.
Input variables: characterID, kind, key, variantID, quantity, expectedRevision.
Variant selection: the optional variantID selects the base item document when it is absent and exactly one stored variant of the same (kind, key) pair when it is present; gamecatalog.Catalog.ResourceByKindKeyAndVariant is the single implementation of that rule.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// AddItemToStorageEndpointID is the stable backend identifier of AddItemToStorage.
const AddItemToStorageEndpointID = "add_item_to_storage"

// AddItemToStorageDefinition describes the public mutation contract.
var AddItemToStorageDefinition = contract.MustDefine(contract.Definition{
	Name:                       "AddItemToStorage",
	ID:                         AddItemToStorageEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument z addToStorage.enabled=true",
	SupportedResourceVariables: []string{"characterID", "kind", "key", "variantID", "quantity", "expectedRevision"},
	Description:                "Adds the specified resource or variant to Storage after validating addToStorage, capacity, relations, and the complete mutation plan.",
})
