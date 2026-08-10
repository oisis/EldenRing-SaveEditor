/*
Endpoint: GetWhetblades
EndpointID: get_whetblades
Purpose: Returns Whetblades and their unlock state.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument: Whetblade.
Input variables: characterID, availabilityFilter.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetWhetbladesEndpointID is the stable backend identifier of GetWhetblades.
const GetWhetbladesEndpointID = "get_whetblades"

// GetWhetbladesDefinition describes the public getter contract.
var GetWhetbladesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetWhetblades",
	ID:                         GetWhetbladesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Whetblade",
	SupportedResourceVariables: []string{"characterID", "availabilityFilter"},
	Description:                "Returns Whetblades and their unlock state.",
})
