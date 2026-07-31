/*
Endpoint: SetDiagnosticMode
EndpointID: set_diagnostic_mode
Purpose: Włącza albo wyłącza rozszerzoną diagnostykę bez zmiany zawartości save.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: enabled.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package diagnostics

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetDiagnosticModeEndpointID is the stable backend identifier of SetDiagnosticMode.
const SetDiagnosticModeEndpointID = "set_diagnostic_mode"

// SetDiagnosticModeDefinition describes the public mutation contract.
var SetDiagnosticModeDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetDiagnosticMode",
	ID:                         SetDiagnosticModeEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"enabled"},
	Description:                "Włącza albo wyłącza rozszerzoną diagnostykę bez zmiany zawartości save.",
})
