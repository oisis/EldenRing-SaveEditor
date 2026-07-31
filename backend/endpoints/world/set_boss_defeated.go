/*
Endpoint: SetBossDefeated
EndpointID: set_boss_defeated
Purpose: Ustawia stan pokonania bossa wyłącznie według potwierdzonego kontraktu tego zasobu.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: Boss z grant.endpoint=set_boss_defeated.
Input variables: characterID, bossResourceID, defeated, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetBossDefeatedEndpointID is the stable backend identifier of SetBossDefeated.
const SetBossDefeatedEndpointID = "set_boss_defeated"

// SetBossDefeatedDefinition describes the public mutation contract.
var SetBossDefeatedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetBossDefeated",
	ID:                         SetBossDefeatedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "Boss z grant.endpoint=set_boss_defeated",
	SupportedResourceVariables: []string{"characterID", "bossResourceID", "defeated", "expectedRevision"},
	Description:                "Ustawia stan pokonania bossa wyłącznie według potwierdzonego kontraktu tego zasobu.",
})
