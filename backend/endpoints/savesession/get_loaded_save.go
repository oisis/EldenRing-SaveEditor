/*
Endpoint: GetLoadedSave
EndpointID: get_loaded_save
Purpose: Zwraca tożsamość załadowanego save, platformę, wersję formatu, stan zmian i bezpieczne metadane sesji.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: saveSessionID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package savesession

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetLoadedSaveEndpointID is the stable backend identifier of GetLoadedSave.
const GetLoadedSaveEndpointID = "get_loaded_save"

// GetLoadedSaveDefinition describes the public getter contract.
var GetLoadedSaveDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetLoadedSave",
	ID:                         GetLoadedSaveEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID"},
	Description:                "Zwraca tożsamość załadowanego save, platformę, wersję formatu, stan zmian i bezpieczne metadane sesji.",
})
