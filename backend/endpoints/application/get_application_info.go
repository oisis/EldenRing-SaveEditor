/*
Endpoint: GetApplicationInfo
EndpointID: get_application_info
Purpose: Zwraca wersję aplikacji, wersje obsługiwanych schematów oraz podstawowe informacje o możliwościach backendu.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package application

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetApplicationInfoEndpointID is the stable backend identifier of GetApplicationInfo.
const GetApplicationInfoEndpointID = "get_application_info"

// GetApplicationInfoDefinition describes the public getter contract.
var GetApplicationInfoDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetApplicationInfo",
	ID:                         GetApplicationInfoEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Zwraca wersję aplikacji, wersje obsługiwanych schematów oraz podstawowe informacje o możliwościach backendu.",
})
