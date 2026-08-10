/*
Endpoint: SetGestureUnlocked
EndpointID: set_gesture_unlocked
Purpose: Unlocks or locks a gesture and changes only its associated GestureGameData state.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Gesture z grant.endpoint=set_gesture_unlocked.
Input variables: characterID, gestureKind, gestureKey, unlocked, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package world

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetGestureUnlockedEndpointID is the stable backend identifier of SetGestureUnlocked.
const SetGestureUnlockedEndpointID = "set_gesture_unlocked"

// SetGestureUnlockedDefinition describes the public mutation contract.
var SetGestureUnlockedDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetGestureUnlocked",
	ID:                         SetGestureUnlockedEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Gesture z grant.endpoint=set_gesture_unlocked",
	SupportedResourceVariables: []string{"characterID", "gestureKind", "gestureKey", "unlocked", "expectedRevision"},
	Description:                "Unlocks or locks a gesture and changes only its associated GestureGameData state.",
})
