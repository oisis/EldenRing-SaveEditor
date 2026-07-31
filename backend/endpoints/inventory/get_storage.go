/*
Endpoint: GetStorage
EndpointID: get_storage
Purpose: Zwraca pełny, uporządkowany widok Storage według tego samego kontraktu instancji co Inventory.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument.
Input variables: characterID, family, containerSection, page, pageSize.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetStorageEndpointID is the stable backend identifier of GetStorage.
const GetStorageEndpointID = "get_storage"

// GetStorageDefinition describes the public getter contract.
var GetStorageDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetStorage",
	ID:                         GetStorageEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"characterID", "family", "containerSection", "page", "pageSize"},
	Description:                "Zwraca pełny, uporządkowany widok Storage według tego samego kontraktu instancji co Inventory.",
})
