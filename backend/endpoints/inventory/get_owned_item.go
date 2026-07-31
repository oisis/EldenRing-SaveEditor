/*
Endpoint: GetOwnedItem
EndpointID: get_owned_item
Purpose: Zwraca szczegóły jednej posiadanej instancji itemu wskazanej przez stabilne OwnedItemID.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument.
Input variables: characterID, ownedItemID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetOwnedItemEndpointID is the stable backend identifier of GetOwnedItem.
const GetOwnedItemEndpointID = "get_owned_item"

// GetOwnedItemDefinition describes the public getter contract.
var GetOwnedItemDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetOwnedItem",
	ID:                         GetOwnedItemEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"characterID", "ownedItemID"},
	Description:                "Zwraca szczegóły jednej posiadanej instancji itemu wskazanej przez stabilne OwnedItemID.",
})
