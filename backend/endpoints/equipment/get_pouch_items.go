/*
Endpoint: GetPouchItems
EndpointID: get_pouch_items
Purpose: Zwraca stan slotów Pouch.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument z capability equipment dopuszczającą slot pouch.
Input variables: characterID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetPouchItemsEndpointID is the stable backend identifier of GetPouchItems.
const GetPouchItemsEndpointID = "get_pouch_items"

// GetPouchItemsDefinition describes the public getter contract.
var GetPouchItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetPouchItems",
	ID:                         GetPouchItemsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument z capability equipment dopuszczającą slot pouch",
	SupportedResourceVariables: []string{"characterID"},
	Description:                "Zwraca stan slotów Pouch.",
})
