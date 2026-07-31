/*
Endpoint: GetCookbooks
EndpointID: get_cookbooks
Purpose: Zwraca cookbooks i stan ich odblokowania.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument: Cookbook.
Input variables: characterID, availabilityFilter.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetCookbooksEndpointID is the stable backend identifier of GetCookbooks.
const GetCookbooksEndpointID = "get_cookbooks"

// GetCookbooksDefinition describes the public getter contract.
var GetCookbooksDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCookbooks",
	ID:                         GetCookbooksEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Cookbook",
	SupportedResourceVariables: []string{"characterID", "availabilityFilter"},
	Description:                "Zwraca cookbooks i stan ich odblokowania.",
})
