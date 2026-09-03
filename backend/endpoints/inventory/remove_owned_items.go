/*
Endpoint: RemoveOwnedItems
EndpointID: remove_owned_items
Purpose: Atomically removes several owned item instances of one character as one save revision.
How it works: The runtime handler forwards the exact identities to SaveEngine, which resolves each of them, plans every removal against a private candidate image of the snapshot and swaps that image in only after the last removal succeeded. The endpoint opens no file, parses no save data of its own, resolves no GameCatalog document and calls no other endpoint.
Supported resource types: any owned instance addressable by an OwnedItemID in a section with a confirmed removal contract.
Input variables: saveSessionID, characterID, ownedItemIDs, expectedRevision.
GameCatalog variables read: none; a removal is addressed by identity and needs no catalog document.
Save variables processed: for every removal the twelve bytes of the addressed record and, where the section is confirmed to move it, the non-empty count of that section; SaveEngine validates the complete batch and finishes with full success or no change at all.
Implementation status: implemented
*/
package inventory

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// RemoveOwnedItemsEndpointID is the stable backend identifier of RemoveOwnedItems.
const RemoveOwnedItemsEndpointID = "remove_owned_items"

// RemoveOwnedItemsDefinition describes the public mutation contract.
var RemoveOwnedItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                   "RemoveOwnedItems",
	ID:                     RemoveOwnedItemsEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "any owned instance addressable by an OwnedItemID in a section with a confirmed removal contract",
	SupportedResourceVariables: []string{
		"saveSessionID", "characterID", "ownedItemIDs", "expectedRevision",
	},
	Description: "Atomically removes several owned item instances of one character as one save revision.",
})

// RemoveOwnedItemsResult is the public name of the receipt SaveEngine owns. The
// endpoint adds no field, drops none and renames none.
type RemoveOwnedItemsResult = saveengine.RemoveOwnedItemsResult

// RemoveOwnedItems removes every named record of one character as one mutation
// of one revision.
//
// The batch either applies completely or changes nothing, and one receipt with
// one operationID describes the whole change. A record an Equipment, Quick Item
// or Pouch slot references rejects the whole batch: the shared removal planner
// refuses it, and no earlier step of the batch is kept. An empty list and a
// repeated ownedItemID are both rejected before the session is touched.
//
// saveSessionID, every ownedItemID and expectedRevision are passed through byte
// for byte; they are never trimmed, normalised or parsed here.
func RemoveOwnedItems(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	ownedItemIDs []string,
	expectedRevision string,
) (RemoveOwnedItemsResult, error) {
	if engine == nil {
		return RemoveOwnedItemsResult{}, errors.New("save engine is not available")
	}
	return engine.RemoveOwnedItems(saveSessionID, characterID, ownedItemIDs, expectedRevision)
}
