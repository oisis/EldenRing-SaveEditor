/*
Endpoint: SetWhetbladeUnlocked
EndpointID: set_whetblade_unlocked
Purpose: Ustawia stan odblokowania Whetblade.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Whetblade z grant.endpoint=set_whetblade_unlocked.
Input variables: characterID, whetbladeKind, whetbladeKey, unlocked, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetWhetbladeUnlockedEndpointID is the stable backend identifier of SetWhetbladeUnlocked.
const SetWhetbladeUnlockedEndpointID = "set_whetblade_unlocked"

// SetWhetbladeUnlockedDefinition describes the public mutation contract.
var SetWhetbladeUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetWhetbladeUnlocked",
	ID:                         SetWhetbladeUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Whetblade z grant.endpoint=set_whetblade_unlocked",
	SupportedResourceVariables: []string{"characterID", "whetbladeKind", "whetbladeKey", "unlocked", "expectedRevision"},
	Description:                "Ustawia stan odblokowania Whetblade.",
})
