/*
Endpoint: SetOwnedItemQuantity
EndpointID: set_owned_item_quantity
Purpose: Sets the quantity of an existing item instance while respecting stack and container limits.
How it works: The runtime handler reads the addressed record through SaveEngine, resolves its one save-side game ID through the already loaded GameCatalog, derives the two limits from that ItemDocument and delegates one atomic mutation to SaveEngine. The endpoint opens no file, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument z capability stack.
Input variables: saveSessionID, characterID, ownedItemID, quantity, expectedRevision.
GameCatalog variables read: item.capabilities.stack.known, item.capabilities.stack.enabled and item.capabilities.stack.rules.maxPerStack; item.storage.recordMode; item.storage.maxInventory for an Inventory record and item.storage.maxStorage for a Storage record.
Save variables processed: the four quantity bytes of the one physical record the identity was minted for, inside the session's private snapshot; SaveEngine validates the complete plan and finishes with full success or rollback.
Implementation status: implemented
*/
package inventory

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetOwnedItemQuantityEndpointID is the stable backend identifier of SetOwnedItemQuantity.
const SetOwnedItemQuantityEndpointID = "set_owned_item_quantity"

// SetOwnedItemQuantityDefinition describes the public mutation contract.
var SetOwnedItemQuantityDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetOwnedItemQuantity",
	ID:                         SetOwnedItemQuantityEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument z capability stack",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "ownedItemID", "quantity", "expectedRevision"},
	Description:                "Sets the quantity of an existing item instance while respecting stack and container limits.",
})

// SetOwnedItemQuantityResult reports the one committed quantity change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. OwnedItemID is echoed back exactly as supplied and is
// already stale: the commit retired every identity of the previous revision, so
// it identifies the performed operation rather than a record the caller may
// address again. Reading the container back under the new revision mints fresh
// identities.
type SetOwnedItemQuantityResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	OwnedItemID   string `json:"ownedItemID"`
	CharacterID   int    `json:"characterID"`
	Quantity      uint32 `json:"quantity"`
}

// SetOwnedItemQuantity sets the stored quantity of the one owned instance
// ownedItemID was minted for.
//
// The endpoint owns exactly one decision SaveEngine cannot make: the two limits.
// It reads the record, resolves its game ID to one ItemDocument and derives
//
//	maxPerRecord      = min(capabilities.stack.rules.maxPerStack, maxContainerTotal)
//	maxContainerTotal = storage.maxInventory for an Inventory record,
//	                    storage.maxStorage for a Storage record
//
// The rule is deliberately fail-closed: in Storage a single record still never
// exceeds maxPerStack, even when maxStorage is larger, because the per-stack
// limit is what one physical row is known to hold. Nothing is defaulted,
// widened or clamped here, and no mode is accepted, so the Safe Mode fields and
// the "-sfv" fields are never read.
//
// Unknown catalog data rejects the whole request: an unknown or disabled stack
// capability, missing stack rules, a recordMode that is unknown or
// separate_instances, and an unknown or zero limit of the record's own
// container are all hard errors. No placeholder limit is invented.
//
// saveSessionID, ownedItemID and expectedRevision are passed through byte for
// byte; they are never trimmed, normalised or parsed here. quantity is passed
// through unchanged and is never clamped to a limit: a value above a limit is
// rejected by SaveEngine.
//
// The mutation itself belongs to SaveEngine, which performs it atomically under
// its own lock, verifies the write, rolls back a failed one and advances the
// session revision only on success. It changes the session's private snapshot;
// no file is written, because there is no WriteSave yet.
func SetOwnedItemQuantity(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	quantity uint32,
	expectedRevision string,
) (SetOwnedItemQuantityResult, error) {
	if engine == nil {
		return SetOwnedItemQuantityResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetOwnedItemQuantityResult{}, errors.New("game catalog is not available")
	}

	owned, err := engine.GetOwnedItem(saveSessionID, characterID, ownedItemID)
	if err != nil {
		return SetOwnedItemQuantityResult{}, err
	}

	gameIDs, err := engine.ResolveGaItemIDs(saveSessionID, characterID, []uint32{owned.GaItemHandle})
	if err != nil {
		return SetOwnedItemQuantityResult{}, err
	}
	gameID := gameIDs[0]
	resource, exists := gameCatalog.ItemByGameID(gameID)
	if !exists || resource.Kind != schema.ResourceKindItem || resource.Item == nil || resource.Key == "" {
		return SetOwnedItemQuantityResult{}, fmt.Errorf("owned item %q: game ID 0x%08X is not a known item",
			ownedItemID, gameID)
	}

	stack := resource.Item.Capabilities.Stack
	if !stack.Known {
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X has an unknown stack capability", ownedItemID, gameID)
	}
	if !stack.Enabled {
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X does not stack", ownedItemID, gameID)
	}
	if stack.Rules == nil || stack.Rules.MaxPerStack == 0 {
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X carries no stack limit", ownedItemID, gameID)
	}

	storage := resource.Item.Storage
	if !storage.RecordMode.Known || storage.RecordMode.Value != schema.RecordModeQuantityStack {
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X does not store a quantity in one record", ownedItemID, gameID)
	}

	// ponytail: the container is the record's own physical container, so the two
	// literal cases stay literal instead of becoming a lookup table of one rule.
	var maxContainerTotal uint32
	switch owned.Container {
	case "inventory":
		if !storage.MaxInventory.Known || storage.MaxInventory.Value == 0 {
			return SetOwnedItemQuantityResult{}, fmt.Errorf(
				"owned item %q: item 0x%08X carries no inventory limit", ownedItemID, gameID)
		}
		maxContainerTotal = storage.MaxInventory.Value
	case "storage":
		if !storage.MaxStorage.Known || storage.MaxStorage.Value == 0 {
			return SetOwnedItemQuantityResult{}, fmt.Errorf(
				"owned item %q: item 0x%08X carries no storage limit", ownedItemID, gameID)
		}
		maxContainerTotal = storage.MaxStorage.Value
	default:
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"owned item %q lives in unknown container %q", ownedItemID, owned.Container)
	}

	committed, err := engine.SetOwnedItemQuantity(
		saveSessionID,
		characterID,
		ownedItemID,
		quantity,
		expectedRevision,
		gameID,
		min(stack.Rules.MaxPerStack, maxContainerTotal),
		maxContainerTotal,
	)
	if err != nil {
		return SetOwnedItemQuantityResult{}, err
	}

	return SetOwnedItemQuantityResult{
		SaveSessionID: committed.SaveSessionID,
		SaveRevision:  committed.SaveRevision,
		OwnedItemID:   committed.OwnedItemID,
		CharacterID:   committed.CharacterID,
		Quantity:      committed.Quantity,
	}, nil
}
