/*
Endpoint: GetInventory
EndpointID: get_inventory
Purpose: Zwraca pełny, uporządkowany widok Inventory wraz z tożsamością instancji, ilością, wariantem, stanem wyposażenia i danymi katalogowymi.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument.
Input variables: characterID, family, containerSection, page, pageSize.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetInventoryEndpointID is the stable backend identifier of GetInventory.
const GetInventoryEndpointID = "get_inventory"

// GetInventoryDefinition describes the public getter contract.
var GetInventoryDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetInventory",
	ID:                         GetInventoryEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"characterID", "family", "containerSection", "page", "pageSize"},
	Description:                "Zwraca pełny, uporządkowany widok Inventory wraz z tożsamością instancji, ilością, wariantem, stanem wyposażenia i danymi katalogowymi.",
})
