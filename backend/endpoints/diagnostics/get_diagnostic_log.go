/*
Endpoint: GetDiagnosticLog
EndpointID: get_diagnostic_log
Purpose: Returns a safe portion of the current session's structured diagnostic log.
How it works: The runtime handler delegates to SaveEngine under the engine lock and returns a safe, typed result without modifying save or session state.
Supported resource types: —.
Input variables: saveSessionID, cursor, limit, severity, scope.
GameCatalog variables read: none required by the current contract.
Save variables read: the diagnostic journal of the session registered under saveSessionID; the getter remains non-mutating.
Implementation status: implemented
*/
package diagnostics

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetDiagnosticLogEndpointID is the stable backend identifier of GetDiagnosticLog.
const GetDiagnosticLogEndpointID = "get_diagnostic_log"

// GetDiagnosticLogDefinition describes the public getter contract.
var GetDiagnosticLogDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetDiagnosticLog",
	ID:                         GetDiagnosticLogEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "cursor", "limit", "severity", "scope"},
	Description:                "Returns a safe portion of the current session's structured diagnostic log.",
})

// DiagnosticRecord aliases the saveengine diagnostic record type for public exposure.
type DiagnosticRecord = saveengine.DiagnosticRecord

// GetDiagnosticLogResult aliases the saveengine result type for public exposure.
type GetDiagnosticLogResult = saveengine.DiagnosticLogResult

// GetDiagnosticLog returns a safe, filtered portion of the current session's diagnostic log.
func GetDiagnosticLog(
	engine *saveengine.Engine,
	saveSessionID string,
	cursor string,
	limit int,
	severity string,
	scope string,
) (GetDiagnosticLogResult, error) {
	if engine == nil {
		return GetDiagnosticLogResult{}, errors.New("saveengine.Engine is required")
	}
	return engine.GetDiagnosticLog(saveSessionID, cursor, limit, severity, scope)
}
