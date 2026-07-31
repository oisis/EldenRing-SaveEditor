/*
Endpoint: CloseSave
EndpointID: close_save
Purpose: Zamyka bieżącą sesję save po jawnej obsłudze niezapisanych zmian.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, expectedRevision, unsavedChangesPolicy.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package savesession

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// CloseSaveEndpointID is the stable backend identifier of CloseSave.
const CloseSaveEndpointID = "close_save"

// CloseSaveDefinition describes the public mutation contract.
var CloseSaveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CloseSave",
	ID:                         CloseSaveEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "expectedRevision", "unsavedChangesPolicy"},
	Description:                "Zamyka bieżącą sesję save po jawnej obsłudze niezapisanych zmian.",
})
