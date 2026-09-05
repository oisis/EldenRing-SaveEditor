/*
Endpoint: GetDiagnosticMode
EndpointID: get_diagnostic_mode
Purpose: Reports whether extended diagnostics are enabled and whether local diagnostic logging works.
How it works: The runtime handler reads the single diagnostic service the composition root owns and returns its state. It reads no save, opens no session and mutates nothing.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none; diagnostic mode is instance runtime state and is not save state.
Implementation status: implemented
*/
package diagnostics

import (
	"errors"

	diagnosticsservice "github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// GetDiagnosticModeEndpointID is the stable backend identifier of GetDiagnosticMode.
const GetDiagnosticModeEndpointID = "get_diagnostic_mode"

// GetDiagnosticModeDefinition describes the public getter contract.
var GetDiagnosticModeDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetDiagnosticMode",
	ID:                         GetDiagnosticModeEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Reports whether extended diagnostics are enabled and whether local diagnostic logging works.",
})

// GetDiagnosticMode reports the current diagnostic state.
func GetDiagnosticMode(
	service *diagnosticsservice.Service,
) (DiagnosticModeResult, error) {
	if service == nil {
		return DiagnosticModeResult{}, errors.New("the diagnostic service is not available")
	}
	return service.State(), nil
}
