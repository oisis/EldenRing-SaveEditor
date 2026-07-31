/*
Endpoint: GetRepairPlan
EndpointID: get_repair_plan
Purpose: Buduje niemutujący, związany z aktualną wersją save plan wybranych napraw.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: GameResource references.
Input variables: saveSessionID, saveRevision, issueIDs.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package diagnostics

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetRepairPlanEndpointID is the stable backend identifier of GetRepairPlan.
const GetRepairPlanEndpointID = "get_repair_plan"

// GetRepairPlanDefinition describes the public getter contract.
var GetRepairPlanDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetRepairPlan",
	ID:                         GetRepairPlanEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "saveRevision", "issueIDs"},
	Description:                "Buduje niemutujący, związany z aktualną wersją save plan wybranych napraw.",
})
