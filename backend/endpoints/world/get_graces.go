/*
Endpoint: GetGraces
EndpointID: get_graces
Purpose: Zwraca Sites of Grace i informację, czy zostały odwiedzone.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: Grace.
Input variables: characterID, regionResourceID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetGracesEndpointID is the stable backend identifier of GetGraces.
const GetGracesEndpointID = "get_graces"

// GetGracesDefinition describes the public getter contract.
var GetGracesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetGraces",
	ID:                         GetGracesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "Grace",
	SupportedResourceVariables: []string{"characterID", "regionResourceID"},
	Description:                "Zwraca Sites of Grace i informację, czy zostały odwiedzone.",
})
