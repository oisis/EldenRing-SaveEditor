/*
Endpoint: GetCatalogInfo
EndpointID: get_catalog_info
Purpose: Zwraca wersję schematu i danych GameCatalog, wersję gry, status walidacji oraz manifest użytych źródeł.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package catalog

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetCatalogInfoEndpointID is the stable backend identifier of GetCatalogInfo.
const GetCatalogInfoEndpointID = "get_catalog_info"

// GetCatalogInfoDefinition describes the public getter contract.
var GetCatalogInfoDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCatalogInfo",
	ID:                         GetCatalogInfoEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Zwraca wersję schematu i danych GameCatalog, wersję gry, status walidacji oraz manifest użytych źródeł.",
})
