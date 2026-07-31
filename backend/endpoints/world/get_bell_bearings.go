/*
Endpoint: GetBellBearings
EndpointID: get_bell_bearings
Purpose: Zwraca Bell Bearings i stan ich odblokowania.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument: BellBearing.
Input variables: characterID, availabilityFilter.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetBellBearingsEndpointID is the stable backend identifier of GetBellBearings.
const GetBellBearingsEndpointID = "get_bell_bearings"

// GetBellBearingsDefinition describes the public getter contract.
var GetBellBearingsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBellBearings",
	ID:                         GetBellBearingsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: BellBearing",
	SupportedResourceVariables: []string{"characterID", "availabilityFilter"},
	Description:                "Zwraca Bell Bearings i stan ich odblokowania.",
})
