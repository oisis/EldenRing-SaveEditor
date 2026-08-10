/*
Endpoint: GetTutorials
EndpointID: get_tutorials
Purpose: Returns tutorials and their unlock state.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: Tutorial.
Input variables: characterID, availabilityFilter.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetTutorialsEndpointID is the stable backend identifier of GetTutorials.
const GetTutorialsEndpointID = "get_tutorials"

// GetTutorialsDefinition describes the public getter contract.
var GetTutorialsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetTutorials",
	ID:                         GetTutorialsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "Tutorial",
	SupportedResourceVariables: []string{"characterID", "availabilityFilter"},
	Description:                "Returns tutorials and their unlock state.",
})
