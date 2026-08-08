/*
Endpoint: SetQuestStep
EndpointID: set_quest_step
Purpose: Przenosi quest do jawnie wskazanego, dozwolonego kroku.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: Quest z grant.endpoint=set_quest_step.
Input variables: characterID, questKind, questKey, stepKind, stepKey, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetQuestStepEndpointID is the stable backend identifier of SetQuestStep.
const SetQuestStepEndpointID = "set_quest_step"

// SetQuestStepDefinition describes the public mutation contract.
var SetQuestStepDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetQuestStep",
	ID:                         SetQuestStepEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "Quest z grant.endpoint=set_quest_step",
	SupportedResourceVariables: []string{"characterID", "questKind", "questKey", "stepKind", "stepKey", "expectedRevision"},
	Description:                "Przenosi quest do jawnie wskazanego, dozwolonego kroku.",
})
