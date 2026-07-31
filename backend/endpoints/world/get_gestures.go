/*
Endpoint: GetGestures
EndpointID: get_gestures
Purpose: Zwraca gesty i informację, czy każdy z nich jest odblokowany.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument: Gesture.
Input variables: characterID, availabilityFilter.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetGesturesEndpointID is the stable backend identifier of GetGestures.
const GetGesturesEndpointID = "get_gestures"

// GetGesturesDefinition describes the public getter contract.
var GetGesturesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetGestures",
	ID:                         GetGesturesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Gesture",
	SupportedResourceVariables: []string{"characterID", "availabilityFilter"},
	Description:                "Zwraca gesty i informację, czy każdy z nich jest odblokowany.",
})
