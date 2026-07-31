/*
Endpoint: ApplyRepairs
EndpointID: apply_repairs
Purpose: Wykonuje dokładnie plan zwrócony przez GetRepairPlan, związany z konkretną sesją i rewizją save.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, planToken, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package diagnostics

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// ApplyRepairsEndpointID is the stable backend identifier of ApplyRepairs.
const ApplyRepairsEndpointID = "apply_repairs"

// ApplyRepairsDefinition describes the public mutation contract.
var ApplyRepairsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyRepairs",
	ID:                         ApplyRepairsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "planToken", "expectedRevision"},
	Description:                "Wykonuje dokładnie plan zwrócony przez GetRepairPlan, związany z konkretną sesją i rewizją save.",
})
