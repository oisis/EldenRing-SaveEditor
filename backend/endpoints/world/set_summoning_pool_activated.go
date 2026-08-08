/*
Endpoint: SetSummoningPoolActivated
EndpointID: set_summoning_pool_activated
Purpose: Ustawia stan aktywacji Summoning Pool.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: SummoningPool z grant.endpoint=set_summoning_pool_activated.
Input variables: characterID, summoningPoolKind, summoningPoolKey, activated, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetSummoningPoolActivatedEndpointID is the stable backend identifier of SetSummoningPoolActivated.
const SetSummoningPoolActivatedEndpointID = "set_summoning_pool_activated"

// SetSummoningPoolActivatedDefinition describes the public mutation contract.
var SetSummoningPoolActivatedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetSummoningPoolActivated",
	ID:                         SetSummoningPoolActivatedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "SummoningPool z grant.endpoint=set_summoning_pool_activated",
	SupportedResourceVariables: []string{"characterID", "summoningPoolKind", "summoningPoolKey", "activated", "expectedRevision"},
	Description:                "Ustawia stan aktywacji Summoning Pool.",
})
