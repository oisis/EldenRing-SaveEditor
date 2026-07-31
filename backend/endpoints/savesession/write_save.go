/*
Endpoint: WriteSave
EndpointID: write_save
Purpose: Serializuje, ponownie wczytuje i waliduje wynik, po czym atomowo zapisuje go do jawnie wskazanego celu.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: saveSessionID, expectedRevision, target.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package savesession

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// WriteSaveEndpointID is the stable backend identifier of WriteSave.
const WriteSaveEndpointID = "write_save"

// WriteSaveDefinition describes the public mutation contract.
var WriteSaveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "WriteSave",
	ID:                         WriteSaveEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "expectedRevision", "target"},
	Description:                "Serializuje, ponownie wczytuje i waliduje wynik, po czym atomowo zapisuje go do jawnie wskazanego celu.",
})
