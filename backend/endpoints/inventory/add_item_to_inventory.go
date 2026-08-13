/*
Endpoint: AddItemToInventory
EndpointID: add_item_to_inventory
Purpose: Adds the specified resource or variant to the common section of Inventory after validating the family, the routing, the limits and the complete mutation plan.
How it works: The runtime handler resolves the requested (kind, key, variantID) through the already loaded GameCatalog, proves that the resolved document is one of the two families whose save-side handle is derived from the game ID alone, refuses a key-routed resource, derives the record mode and the two limits from that ItemDocument and delegates one atomic mutation to SaveEngine. The endpoint opens no file, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument of family goods or talisman, outside category key_items.
Input variables: saveSessionID, characterID, kind, key, variantID, quantity, expectedRevision.
Variant selection: the optional variantID selects the base item document when it is absent and exactly one stored variant of the same (kind, key) pair when it is present; gamecatalog.Catalog.ResourceByKindKeyAndVariant is the single implementation of that rule.
GameCatalog variables read: item.family, item.category, item.gameID; item.storage.recordMode and item.storage.maxInventory; item.capabilities.stack.known, item.capabilities.stack.enabled and item.capabilities.stack.rules.maxPerStack for a quantity stack.
Save variables processed: for a top-up the four quantity bytes of the first common record of the item; for a new record the twelve bytes of the first free common row, the common item count in front of the section, the two trailing allocators NextEquipIndex and NextAcquisitionSortId, and one active GaItemData entry when the character owns no physical record of the item yet; SaveEngine validates the complete plan and finishes with full success or rollback.
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

// AddItemToInventoryEndpointID is the stable backend identifier of AddItemToInventory.
const AddItemToInventoryEndpointID = "add_item_to_inventory"

// addItemKeyCategory is the catalog category of every resource the game routes
// into the key section of InventoryHeld. This endpoint writes the common section
// only, so it refuses the whole category rather than the four families
// SaveForge 1.x routed there: the category is a stated fact of every goods
// document, and rejecting it needs no hardcoded ID, no schema field and no
// regeneration. The rule is deliberately wider than the legacy routing — a map
// or a bell bearing is a key_items resource the game keeps in the common section
// — and it costs reach in exchange for never writing a key-routed item into the
// wrong section.
const addItemKeyCategory = "key_items"

// The two families this endpoint accepts, and the game-ID prefix each of them
// carries. Only these two have a save-side handle derived from the game ID
// alone; every other family needs a record in the variable-length GaItem table,
// which the mutation never allocates.
const (
	addItemGoodsPrefix    uint32 = 0x40000000
	addItemTalismanPrefix uint32 = 0x20000000
	addItemFamilyPrefix   uint32 = 0xF0000000
)

// AddItemToInventoryDefinition describes the public mutation contract.
var AddItemToInventoryDefinition = contract.MustDefine(contract.Definition{
	Name:                       "AddItemToInventory",
	ID:                         AddItemToInventoryEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument of family goods or talisman, outside category key_items",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "kind", "key", "variantID", "quantity", "expectedRevision"},
	Description:                "Adds the specified resource or variant to the common section of Inventory after validating the family, the routing, the limits and the complete mutation plan.",
})

// AddItemToInventoryResult is the public name of the receipt SaveEngine owns.
//
// The mutation and its result model belong to SaveEngine, so this is an alias
// rather than a copy: the endpoint adds no field, drops none and renames none,
// and the JSON contract is whatever saveengine.AddItemToInventoryResult
// declares. See that type for what SaveRevision, Added, Quantity and
// CreatedRecord mean, and why the receipt carries no OwnedItemID.
type AddItemToInventoryResult = saveengine.AddItemToInventoryResult

// AddItemToInventory adds quantity of the requested resource or variant to the
// common section of the Inventory of one character.
//
// quantity is a delta and is always at least 1: it is the amount added, never a
// target total. An item that stores a quantity in one record is topped up on its
// first common record when the character already holds one, and receives a new
// record otherwise; an item that stores every copy in its own record always
// receives a new record and accepts quantity 1 only.
//
// The endpoint owns exactly the decisions SaveEngine cannot make, and every one
// of them is fail-closed:
//
//   - The family has to be known and has to be goods or talisman. Every other
//     family is rejected before a handle is derived, including the spells and
//     spirit ashes that share the goods game-ID prefix.
//
//   - The game ID has to be known and its prefix has to agree with the family. A
//     disagreement is a hard error and is never silently corrected.
//
//   - The category has to be known and must not be key_items, so a resource the
//     game routes into the key section is never written into the common one.
//
//   - The record mode has to be known and decides the shape of the add.
//
//   - The two limits are
//
//     maxPerRecord      = min(capabilities.stack.rules.maxPerStack, maxInventory)
//     for a quantity stack, 1 for separate instances
//     maxContainerTotal = storage.maxInventory
//
//     Nothing is defaulted, widened or clamped here, and no mode is accepted, so
//     the Safe Mode fields and the "-sfv" fields are never read.
//
// saveSessionID, kind, key and expectedRevision are passed through byte for
// byte; they are never trimmed, normalised or parsed here.
//
// The mutation itself belongs to SaveEngine, which performs it atomically under
// its own lock, verifies every write, rolls back a failed plan and advances the
// session revision only on success. It changes the session's private snapshot; a
// later, separate WriteSave persists that snapshot.
func AddItemToInventory(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	kind string,
	key string,
	variantID *uint32,
	quantity uint32,
	expectedRevision string,
) (AddItemToInventoryResult, error) {
	if engine == nil {
		return AddItemToInventoryResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return AddItemToInventoryResult{}, errors.New("game catalog is not available")
	}

	resource, err := gameCatalog.ResourceByKindKeyAndVariant(schema.ResourceKind(kind), key, variantID)
	if err != nil {
		return AddItemToInventoryResult{}, err
	}
	if resource.Item == nil {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q has no item document", kind, key)
	}
	item := resource.Item

	if !item.Family.Known {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q has an unknown family", kind, key)
	}
	var familyPrefix uint32
	switch item.Family.Value {
	case schema.ItemFamilyGoods:
		familyPrefix = addItemGoodsPrefix
	case schema.ItemFamilyTalisman:
		familyPrefix = addItemTalismanPrefix
	default:
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q is of family %q; this endpoint adds only %q and %q",
			kind, key, item.Family.Value, schema.ItemFamilyGoods, schema.ItemFamilyTalisman)
	}

	if !item.GameID.Known {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q has an unknown game ID", kind, key)
	}
	gameID := item.GameID.Value
	// The prefix is an independent consistency gate rather than a second family
	// test: a document whose two facts disagree is rejected, never reinterpreted
	// into whichever of them looks more plausible.
	if gameID&addItemFamilyPrefix != familyPrefix {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q declares family %q and game ID 0x%08X, which disagree",
			kind, key, item.Family.Value, gameID)
	}

	if !item.Category.Known {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q has an unknown category", kind, key)
	}
	if item.Category.Value == addItemKeyCategory {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q is a %q resource, which the game keeps in the key section;"+
				" this endpoint writes the common section only", kind, key, addItemKeyCategory)
	}

	storage := item.Storage
	if !storage.MaxInventory.Known || storage.MaxInventory.Value == 0 {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q carries no inventory limit", kind, key)
	}
	maxContainerTotal := storage.MaxInventory.Value

	if !storage.RecordMode.Known {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q has an unknown record mode", kind, key)
	}
	var separateInstances bool
	var maxPerRecord uint32
	switch storage.RecordMode.Value {
	case schema.RecordModeQuantityStack:
		stack := item.Capabilities.Stack
		if !stack.Known {
			return AddItemToInventoryResult{}, fmt.Errorf(
				"resource kind %q key %q has an unknown stack capability", kind, key)
		}
		if !stack.Enabled {
			return AddItemToInventoryResult{}, fmt.Errorf(
				"resource kind %q key %q stores a quantity but does not stack", kind, key)
		}
		if stack.Rules == nil || stack.Rules.MaxPerStack == 0 {
			return AddItemToInventoryResult{}, fmt.Errorf(
				"resource kind %q key %q carries no stack limit", kind, key)
		}
		// Fail-closed exactly as SetOwnedItemQuantity derives it: one physical row
		// never holds more than the per-stack limit, whatever the container allows.
		maxPerRecord = min(stack.Rules.MaxPerStack, maxContainerTotal)
	case schema.RecordModeSeparateInstances:
		separateInstances = true
		maxPerRecord = 1
	default:
		return AddItemToInventoryResult{}, fmt.Errorf(
			"resource kind %q key %q declares the unsupported record mode %q",
			kind, key, storage.RecordMode.Value)
	}

	return engine.AddItemToInventory(
		saveSessionID,
		characterID,
		gameID,
		quantity,
		expectedRevision,
		separateInstances,
		maxPerRecord,
		maxContainerTotal,
	)
}
