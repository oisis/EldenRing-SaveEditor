/*
Endpoint: SetGraceVisited
EndpointID: set_grace_visited
Purpose: Sets the visited state of a Site of Grace together with its required confirmed dependencies.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: Grace z grant.endpoint=set_grace_visited.
Input variables: characterID, graceKind, graceKey, visited, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetGraceVisitedEndpointID is the stable backend identifier of SetGraceVisited.
const SetGraceVisitedEndpointID = "set_grace_visited"

// SetGraceVisitedDefinition describes the public mutation contract.
var SetGraceVisitedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetGraceVisited",
	ID:                         SetGraceVisitedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "Grace z grant.endpoint=set_grace_visited",
	SupportedResourceVariables: []string{"characterID", "graceKind", "graceKey", "visited", "expectedRevision"},
	Description:                "Sets the visited state of a Site of Grace together with its required confirmed dependencies.",
})
