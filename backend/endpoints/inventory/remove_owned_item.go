/*
Endpoint: RemoveOwnedItem
EndpointID: remove_owned_item
Purpose: Removes a specific item instance after validating references and effects on Equipment and supporting structures.
How it works: The runtime handler passes the complete request to SaveEngine, which resolves the opaque identity, validates the expected revision and the complete removal plan under its own lock and performs one atomic mutation. The endpoint opens no file, parses no save data of its own, reads no GameCatalog and calls no other endpoint.
Supported resource types: ItemDocument.
Input variables: saveSessionID, characterID, ownedItemID, expectedRevision.
GameCatalog variables read: none; the removal needs no catalog fact, and SaveEngine resolves the record's own save-side game ID for the receipt.
Save variables processed: the twelve bytes of the one physical record the identity was minted for and, where that section's count is confirmed to move, the non-empty count of the section, inside the session's private snapshot; for an InventoryHeld common record the 22 Equipment, 10 Quick Item and 6 Pouch {GaItem handle, 0x180 + physical row} reference pairs of the slot are read as a fail-closed guard and never written; SaveEngine validates the complete plan and finishes with full success or rollback.
Implementation status: implemented
*/
package inventory

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// RemoveOwnedItemEndpointID is the stable backend identifier of RemoveOwnedItem.
const RemoveOwnedItemEndpointID = "remove_owned_item"

// RemoveOwnedItemDefinition describes the public mutation contract.
var RemoveOwnedItemDefinition = contract.MustDefine(contract.Definition{
	Name:                       "RemoveOwnedItem",
	ID:                         RemoveOwnedItemEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "ownedItemID", "expectedRevision"},
	Description:                "Removes a specific item instance after validating references and effects on Equipment and supporting structures.",
})

// RemoveOwnedItemResult is the public name of the receipt SaveEngine owns.
//
// The mutation and its result model belong to SaveEngine, so this is an alias
// rather than a copy: the endpoint adds no field, drops none and renames none,
// and the JSON contract is whatever saveengine.RemoveOwnedItemResult declares.
// See that type for what SaveRevision, GameID and the deliberately stale
// OwnedItemID mean.
type RemoveOwnedItemResult = saveengine.RemoveOwnedItemResult

// RemoveOwnedItem removes the one owned instance ownedItemID was minted for.
//
// Unlike SetOwnedItemQuantity this endpoint owns no decision at all: a removal
// needs no catalog limit, no stack rule and no record mode, so no GameCatalog is
// read and none is required. saveSessionID, ownedItemID and expectedRevision are
// passed through byte for byte; they are never trimmed, normalised or parsed
// here, and the identity is never resolved by game ID or searched for in the
// other container.
//
// The mutation belongs to SaveEngine, which performs it atomically under its own
// lock: it clears the twelve bytes of the addressed record the way that record's
// own section is confirmed to be cleared, lowers the non-empty count of the
// section where that count is confirmed to move, restores every byte it changed
// when a write cannot be verified, and advances the session revision only on
// success. It changes the session's private snapshot; a later, separate
// WriteSave persists that snapshot.
//
// Two removals fail closed there rather than guessing: an InventoryHeld common
// record whose physical row an Equipment, Quick Item or Pouch reference pair
// still names, and a record of the Storage Box key section, which this project
// has no confirmed native write contract for. Neither unequips anything and
// neither writes a byte.
func RemoveOwnedItem(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	expectedRevision string,
) (RemoveOwnedItemResult, error) {
	if engine == nil {
		return RemoveOwnedItemResult{}, errors.New("save engine is not available")
	}
	return engine.RemoveOwnedItem(saveSessionID, characterID, ownedItemID, expectedRevision)
}
