/*
Endpoint: GetColosseums
EndpointID: get_colosseums
Purpose: Zwraca kolosea i stan ich odblokowania.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: Colosseum.
Input variables: characterID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetColosseumsEndpointID is the stable backend identifier of GetColosseums.
const GetColosseumsEndpointID = "get_colosseums"

// GetColosseumsDefinition describes the public getter contract.
var GetColosseumsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetColosseums",
	ID:                         GetColosseumsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "Colosseum",
	SupportedResourceVariables: []string{"characterID"},
	Description:                "Zwraca kolosea i stan ich odblokowania.",
})
