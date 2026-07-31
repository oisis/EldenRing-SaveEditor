/*
Endpoint: GetQuickItems
EndpointID: get_quick_items
Purpose: Zwraca stan slotów Quick Items.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument z capability equipment dopuszczającą slot quick_item.
Input variables: characterID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetQuickItemsEndpointID is the stable backend identifier of GetQuickItems.
const GetQuickItemsEndpointID = "get_quick_items"

// GetQuickItemsDefinition describes the public getter contract.
var GetQuickItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetQuickItems",
	ID:                         GetQuickItemsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument z capability equipment dopuszczającą slot quick_item",
	SupportedResourceVariables: []string{"characterID"},
	Description:                "Zwraca stan slotów Quick Items.",
})
