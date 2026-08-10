/*
Endpoint: GetQuests
EndpointID: get_quests
Purpose: Returns quests, their current steps, and allowed transitions derived from the catalog.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: Quest.
Input variables: characterID, questKind, questKey.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetQuestsEndpointID is the stable backend identifier of GetQuests.
const GetQuestsEndpointID = "get_quests"

// GetQuestsDefinition describes the public getter contract.
var GetQuestsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetQuests",
	ID:                         GetQuestsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "Quest",
	SupportedResourceVariables: []string{"characterID", "questKind", "questKey"},
	Description:                "Returns quests, their current steps, and allowed transitions derived from the catalog.",
})
