/*
Endpoint: SetSaveAccountID
EndpointID: set_save_account_id
Purpose: Sets the save owner identifier according to platform-specific rules.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, accountID, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package savesession

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetSaveAccountIDEndpointID is the stable backend identifier of SetSaveAccountID.
const SetSaveAccountIDEndpointID = "set_save_account_id"

// SetSaveAccountIDDefinition describes the public mutation contract.
var SetSaveAccountIDDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetSaveAccountID",
	ID:                         SetSaveAccountIDEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "accountID", "expectedRevision"},
	Description:                "Sets the save owner identifier according to platform-specific rules.",
})
