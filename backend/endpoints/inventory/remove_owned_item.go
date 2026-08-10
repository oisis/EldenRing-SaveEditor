/*
Endpoint: RemoveOwnedItem
EndpointID: remove_owned_item
Purpose: Removes a specific item instance after validating references and effects on Equipment and supporting structures.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument.
Input variables: characterID, ownedItemID, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// RemoveOwnedItemEndpointID is the stable backend identifier of RemoveOwnedItem.
const RemoveOwnedItemEndpointID = "remove_owned_item"

// RemoveOwnedItemDefinition describes the public mutation contract.
var RemoveOwnedItemDefinition = contract.MustDefine(contract.Definition{
	Name:                       "RemoveOwnedItem",
	ID:                         RemoveOwnedItemEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"characterID", "ownedItemID", "expectedRevision"},
	Description:                "Removes a specific item instance after validating references and effects on Equipment and supporting structures.",
})
