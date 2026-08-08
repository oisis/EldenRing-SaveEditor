/*
Endpoint: GetBosses
EndpointID: get_bosses
Purpose: Zwraca bossów i informację o ich pokonaniu.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: Boss.
Input variables: characterID, regionKind, regionKey.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetBossesEndpointID is the stable backend identifier of GetBosses.
const GetBossesEndpointID = "get_bosses"

// GetBossesDefinition describes the public getter contract.
var GetBossesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBosses",
	ID:                         GetBossesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "Boss",
	SupportedResourceVariables: []string{"characterID", "regionKind", "regionKey"},
	Description:                "Zwraca bossów i informację o ich pokonaniu.",
})
