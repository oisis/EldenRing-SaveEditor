/*
Endpoint: GetDiagnosticLog
EndpointID: get_diagnostic_log
Purpose: Zwraca bezpieczny fragment strukturalnego logu diagnostycznego bieżącej sesji.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: saveSessionID, cursor, limit, severity, scope.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package diagnostics

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetDiagnosticLogEndpointID is the stable backend identifier of GetDiagnosticLog.
const GetDiagnosticLogEndpointID = "get_diagnostic_log"

// GetDiagnosticLogDefinition describes the public getter contract.
var GetDiagnosticLogDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetDiagnosticLog",
	ID:                         GetDiagnosticLogEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "cursor", "limit", "severity", "scope"},
	Description:                "Zwraca bezpieczny fragment strukturalnego logu diagnostycznego bieżącej sesji.",
})
