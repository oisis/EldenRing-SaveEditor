/*
Endpoint: SetTutorialUnlocked
EndpointID: set_tutorial_unlocked
Purpose: Ustawia stan odblokowania tutorialu.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: Tutorial z grant.endpoint=set_tutorial_unlocked.
Input variables: characterID, tutorialResourceID, unlocked, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetTutorialUnlockedEndpointID is the stable backend identifier of SetTutorialUnlocked.
const SetTutorialUnlockedEndpointID = "set_tutorial_unlocked"

// SetTutorialUnlockedDefinition describes the public mutation contract.
var SetTutorialUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetTutorialUnlocked",
	ID:                         SetTutorialUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "Tutorial z grant.endpoint=set_tutorial_unlocked",
	SupportedResourceVariables: []string{"characterID", "tutorialResourceID", "unlocked", "expectedRevision"},
	Description:                "Ustawia stan odblokowania tutorialu.",
})
