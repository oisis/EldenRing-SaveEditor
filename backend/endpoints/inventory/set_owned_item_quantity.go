/*
Endpoint: SetOwnedItemQuantity
EndpointID: set_owned_item_quantity
Purpose: Sets the quantity of an existing item instance while respecting stack and container limits.
How it works: The runtime handler reads the addressed record through SaveEngine, resolves its one save-side game ID through the already loaded GameCatalog, derives the two limits from that ItemDocument under the active Safety Profile and delegates one atomic mutation to SaveEngine. The endpoint opens no file, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument z capability stack.
Input variables: safetyProfile, saveSessionID, characterID, ownedItemID, quantity, expectedRevision.
GameCatalog variables read: item.capabilities.stack.known, item.capabilities.stack.enabled and item.capabilities.stack.rules.maxPerStack; item.storage.recordMode; item.storage.maxInventory and item.storage.safeModeMaxInventory for an Inventory record, item.storage.maxStorage and item.storage.safeModeMaxStorage for a Storage record.
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
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetOwnedItemQuantityEndpointID is the stable backend identifier of SetOwnedItemQuantity.
const SetOwnedItemQuantityEndpointID = "set_owned_item_quantity"

// SetOwnedItemQuantityDefinition describes the public mutation contract.
var SetOwnedItemQuantityDefinition = contract.MustDefine(contract.Definition{
	Name:                   "SetOwnedItemQuantity",
	ID:                     SetOwnedItemQuantityEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "ItemDocument z capability stack",
	SupportedResourceVariables: []string{
		"safetyProfile", "saveSessionID", "characterID", "ownedItemID", "quantity",
		"expectedRevision",
	},
	Description: "Sets the quantity of an existing item instance while respecting stack and container limits.",
})

// SetOwnedItemQuantityResult is the public name of the receipt SaveEngine owns.
//
// The mutation and its result model belong to SaveEngine, so this is an alias
// rather than a copy: the endpoint adds no field, drops none and renames none,
// and the JSON contract is whatever saveengine.SetOwnedItemQuantityResult
// declares. See that type for what SaveRevision and the deliberately stale
// OwnedItemID mean.
type SetOwnedItemQuantityResult = saveengine.SetOwnedItemQuantityResult

// SetOwnedItemQuantity sets the stored quantity of the one owned instance
// ownedItemID was minted for.
//
// The endpoint owns exactly one decision SaveEngine cannot make: the two limits.
// It reads the record, resolves its game ID to one ItemDocument and derives
//
//	maxPerRecord      = min(capabilities.stack.rules.maxPerStack, maxContainerTotal)
//	maxContainerTotal = the container limit of the record's own container under
//	                    the active Safety Profile
//
// The container limit comes from backend/safetyprofile and from nowhere else:
// Safe uses storage.safeModeMaxInventory or storage.safeModeMaxStorage where the
// item declares one and the base limit where it does not, while Expanded Limits
// and Chaos always use the base limit. The "-sfv" research fields stay unread.
//
// safetyProfile is supplied by the host, never by the caller of the interface.
// A call that bypasses the interface is therefore refused above the active limit
// by exactly the rule the interface renders.
//
// The rule is deliberately fail-closed: in Storage a single record still never
// exceeds maxPerStack, even when the container limit is larger, because the
// per-stack limit is what one physical row is known to hold. Nothing is
// defaulted, widened or clamped here.
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
// a later, separate WriteSave persists that snapshot.
func SetOwnedItemQuantity(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
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
	profile, err := safetyprofile.Parse(safetyProfile)
	if err != nil {
		return SetOwnedItemQuantityResult{}, err
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
	// Which number each case yields is the profile policy's decision, not this
	// switch's: the two branches differ only in which policy function they ask.
	var maxContainerTotal uint32
	switch owned.Container {
	case "inventory":
		limit, known := safetyprofile.InventoryLimit(profile, resource.Item)
		if !known || limit == 0 {
			return SetOwnedItemQuantityResult{}, fmt.Errorf(
				"owned item %q: item 0x%08X carries no inventory limit", ownedItemID, gameID)
		}
		maxContainerTotal = limit
	case "storage":
		limit, known := safetyprofile.StorageLimit(profile, resource.Item)
		if !known || limit == 0 {
			return SetOwnedItemQuantityResult{}, fmt.Errorf(
				"owned item %q: item 0x%08X carries no storage limit", ownedItemID, gameID)
		}
		maxContainerTotal = limit
	default:
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"owned item %q lives in unknown container %q", ownedItemID, owned.Container)
	}

	return engine.SetOwnedItemQuantity(
		saveSessionID,
		characterID,
		ownedItemID,
		quantity,
		expectedRevision,
		gameID,
		min(stack.Rules.MaxPerStack, maxContainerTotal),
		maxContainerTotal,
	)
}
