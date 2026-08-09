/*
Endpoint: GetUndoState
EndpointID: get_undo_state
Purpose: Zwraca informację, czy dla postaci istnieje bezpieczny punkt cofnięcia oraz jakiej operacji dotyczy.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package character

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetUndoStateEndpointID is the stable backend identifier of GetUndoState.
const GetUndoStateEndpointID = "get_undo_state"

// GetUndoStateDefinition describes the public getter contract.
var GetUndoStateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetUndoState",
	ID:                         GetUndoStateEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Zwraca informację, czy dla postaci istnieje bezpieczny punkt cofnięcia oraz jakiej operacji dotyczy.",
})
