/*
Endpoint: SetDiagnosticMode
EndpointID: set_diagnostic_mode
Purpose: Turns the extended diagnostics of this running instance on or off.
How it works: The runtime handler delegates to the single diagnostic service the composition root owns and returns the state now in effect. It is a runtime application setting: it needs no saveSessionID and no expectedRevision, it advances no saveRevision, it changes no dirty flag, operation history, Undo/Redo stack or recovery journal, and it produces neither a MutationReceipt nor a session.changed event.
Supported resource types: —.
Input variables: enabled.
GameCatalog variables read: none.
Save variables processed: none. Diagnostic mode is instance runtime state; it never enters a save, a snapshot or a recovery journal, and it is not persisted between launches.
Implementation status: implemented
*/
package diagnostics

import (
	"errors"

	diagnosticsservice "github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// SetDiagnosticModeEndpointID is the stable backend identifier of SetDiagnosticMode.
const SetDiagnosticModeEndpointID = "set_diagnostic_mode"

// SetDiagnosticModeDefinition describes the public mutation contract. It is a
// mutation of the application's runtime diagnostics, never of save data.
var SetDiagnosticModeDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetDiagnosticMode",
	ID:                         SetDiagnosticModeEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"enabled"},
	Description:                "Turns the extended diagnostics of this running instance on or off.",
})

// DiagnosticModeResult is the state of the diagnostics, shared by the setter
// and the getter so the two can never describe it differently.
type DiagnosticModeResult = diagnosticsservice.State

// SetDiagnosticMode turns extended diagnostics on or off and reports the state
// now in effect.
func SetDiagnosticMode(
	service *diagnosticsservice.Service, enabled bool,
) (DiagnosticModeResult, error) {
	if service == nil {
		return DiagnosticModeResult{}, errors.New("the diagnostic service is not available")
	}
	return service.SetDebugMode(enabled), nil
}
