/*
Endpoint: GetSummoningPools
EndpointID: get_summoning_pools
Purpose: Zwraca Summoning Pools i stan ich aktywacji.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: SummoningPool.
Input variables: characterID, regionKind, regionKey.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetSummoningPoolsEndpointID is the stable backend identifier of GetSummoningPools.
const GetSummoningPoolsEndpointID = "get_summoning_pools"

// GetSummoningPoolsDefinition describes the public getter contract.
var GetSummoningPoolsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetSummoningPools",
	ID:                         GetSummoningPoolsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "SummoningPool",
	SupportedResourceVariables: []string{"characterID", "regionKind", "regionKey"},
	Description:                "Zwraca Summoning Pools i stan ich aktywacji.",
})
