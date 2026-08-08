/*
Endpoint: SetCookbookUnlocked
EndpointID: set_cookbook_unlocked
Purpose: Ustawia stan odblokowania cookbook i wszystkie należące do niego potwierdzone zależności.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Cookbook z grant.endpoint=set_cookbook_unlocked.
Input variables: characterID, cookbookKind, cookbookKey, unlocked, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetCookbookUnlockedEndpointID is the stable backend identifier of SetCookbookUnlocked.
const SetCookbookUnlockedEndpointID = "set_cookbook_unlocked"

// SetCookbookUnlockedDefinition describes the public mutation contract.
var SetCookbookUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetCookbookUnlocked",
	ID:                         SetCookbookUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Cookbook z grant.endpoint=set_cookbook_unlocked",
	SupportedResourceVariables: []string{"characterID", "cookbookKind", "cookbookKey", "unlocked", "expectedRevision"},
	Description:                "Ustawia stan odblokowania cookbook i wszystkie należące do niego potwierdzone zależności.",
})
