/*
Endpoint: LoadSave
EndpointID: load_save
Purpose: Wczytuje save ze wskazanego źródła, identyfikuje platformę i format, waliduje strukturę oraz tworzy nową sesję bez modyfikowania pliku wejściowego.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: source, expectedPlatform.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package savesession

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// LoadSaveEndpointID is the stable backend identifier of LoadSave.
const LoadSaveEndpointID = "load_save"

// LoadSaveDefinition describes the public mutation contract.
var LoadSaveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "LoadSave",
	ID:                         LoadSaveEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"source", "expectedPlatform"},
	Description:                "Wczytuje save ze wskazanego źródła, identyfikuje platformę i format, waliduje strukturę oraz tworzy nową sesję bez modyfikowania pliku wejściowego.",
})
