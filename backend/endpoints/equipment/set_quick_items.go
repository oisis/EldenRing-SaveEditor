/*
Endpoint: SetQuickItems
EndpointID: set_quick_items
Purpose: Atomowo ustawia zawartość slotów Quick Items.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument z capability equipment dopuszczającą slot quick_item.
Input variables: characterID, slotAssignments, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetQuickItemsEndpointID is the stable backend identifier of SetQuickItems.
const SetQuickItemsEndpointID = "set_quick_items"

// SetQuickItemsDefinition describes the public mutation contract.
var SetQuickItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetQuickItems",
	ID:                         SetQuickItemsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument z capability equipment dopuszczającą slot quick_item",
	SupportedResourceVariables: []string{"characterID", "slotAssignments", "expectedRevision"},
	Description:                "Atomowo ustawia zawartość slotów Quick Items.",
})
