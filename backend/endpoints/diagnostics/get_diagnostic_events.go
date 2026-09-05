/*
Endpoint: GetDiagnosticEvents
EndpointID: get_diagnostic_events
Purpose: Returns a safe portion of the instance-wide diagnostic event stream.
How it works: The runtime handler reads the single diagnostic service the composition root owns and returns a cursor-addressed page of its bounded record buffer. Unlike GetDiagnosticLog it is not bound to a save session, so the console works before any save is opened; it is non-mutating and consumes nothing.
Supported resource types: —.
Input variables: cursor, limit, severity.
GameCatalog variables read: none.
Save variables read: none. The records are produced from a closed event catalogue of build facts, closed identifiers, generated correlation values and numbers; they carry no save bytes, no path, no host and no account.
Implementation status: implemented
*/
package diagnostics

import (
	"errors"

	diagnosticsservice "github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// GetDiagnosticEventsEndpointID is the stable backend identifier of GetDiagnosticEvents.
const GetDiagnosticEventsEndpointID = "get_diagnostic_events"

// GetDiagnosticEventsDefinition describes the public getter contract.
var GetDiagnosticEventsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetDiagnosticEvents",
	ID:                         GetDiagnosticEventsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"cursor", "limit", "severity"},
	Description:                "Returns a safe portion of the instance-wide diagnostic event stream.",
})

// DiagnosticEvent aliases the service record type for public exposure.
type DiagnosticEvent = diagnosticsservice.Record

// GetDiagnosticEventsResult aliases the service page type for public exposure.
type GetDiagnosticEventsResult = diagnosticsservice.Page

// GetDiagnosticEvents returns one page of the instance-wide event stream.
func GetDiagnosticEvents(
	service *diagnosticsservice.Service, cursor string, limit int, severity string,
) (GetDiagnosticEventsResult, error) {
	if service == nil {
		return GetDiagnosticEventsResult{}, errors.New("the diagnostic service is not available")
	}
	return service.Records(cursor, limit, severity)
}
