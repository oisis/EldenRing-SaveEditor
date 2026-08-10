/*
Endpoint: GetItemCapacity
EndpointID: get_item_capacity
Purpose: Returns the current capacity of the relevant containers and supporting structures and the cost of the planned item addition. The getter reserves and mutates nothing.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument.
Input variables: characterID, destination, kind, key, variantID, quantity.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetItemCapacityEndpointID is the stable backend identifier of GetItemCapacity.
const GetItemCapacityEndpointID = "get_item_capacity"

// GetItemCapacityDefinition describes the public getter contract.
var GetItemCapacityDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetItemCapacity",
	ID:                         GetItemCapacityEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"characterID", "destination", "kind", "key", "variantID", "quantity"},
	Description:                "Returns the current capacity of the relevant containers and supporting structures and the cost of the planned item addition. The getter reserves and mutates nothing.",
})
