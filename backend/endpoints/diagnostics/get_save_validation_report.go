/*
Endpoint: GetSaveValidationReport
EndpointID: get_save_validation_report
Purpose: Runs non-mutating save validation and returns detected problems without proposing unconfirmed repairs.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, scope.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package diagnostics

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetSaveValidationReportEndpointID is the stable backend identifier of GetSaveValidationReport.
const GetSaveValidationReportEndpointID = "get_save_validation_report"

// GetSaveValidationReportDefinition describes the public getter contract.
var GetSaveValidationReportDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetSaveValidationReport",
	ID:                         GetSaveValidationReportEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "scope"},
	Description:                "Runs non-mutating save validation and returns detected problems without proposing unconfirmed repairs.",
})
