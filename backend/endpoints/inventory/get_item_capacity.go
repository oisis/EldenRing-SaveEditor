/*
Endpoint: GetItemCapacity
EndpointID: get_item_capacity
Purpose: Reports whether one planned addition fits the selected common item container and the physical-record and GaItemData cost it would consume. The getter reserves and mutates nothing.
How it works: The handler resolves the requested resource through GameCatalog, proves the common-only add contract and destination-specific limits, then asks SaveEngine for one read-only preflight against the private session snapshot.
Supported resource types: ItemDocument of family goods or talisman, outside category key_items and the unresolved Flasks subcategory; Storage additionally requires depositable goods.
Input variables: saveSessionID, characterID, destination, kind, key, variantID, quantity.
Variant selection: the optional variantID selects the base item document when it is absent and exactly one stored variant of the same (kind, key) pair when it is present; gamecatalog.Catalog.ResourceByKindKeyAndVariant is the single implementation of that rule.
GameCatalog variables read: item.family, item.gameID, item.category, item.subcategory, item.storage.recordMode, item.storage.maxInventory or maxStorage, item.capabilities.stack, and item.goods.isDepositable for goods targeting Storage.
Save variables read: activity and revision; both common and key sections of Inventory and Storage; the GaItem table needed to resolve their handles; the destination common count and acquisition allocators; and the active GaItemData count and IDs. Nothing is written or reserved.
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

// GetItemCapacityEndpointID is the stable backend identifier of GetItemCapacity.
const GetItemCapacityEndpointID = "get_item_capacity"

// GetItemCapacityDefinition describes the public getter contract.
var GetItemCapacityDefinition = contract.MustDefine(contract.Definition{
	Name:                   "GetItemCapacity",
	ID:                     GetItemCapacityEndpointID,
	Kind:                   contract.Getter,
	SupportedResourceTypes: "ItemDocument of family goods or talisman supported by common-only addition",
	SupportedResourceVariables: []string{
		"saveSessionID", "characterID", "destination", "kind", "key", "variantID", "quantity",
	},
	Description: "Reports the current common-container, allocator and GaItemData capacity for one" +
		" planned item addition without reserving or mutating anything.",
})

// GetItemCapacityResult adds the public catalog identity to the read-only
// SaveEngine preflight. The embedded capacity remains the single definition of
// every save-derived field.
type GetItemCapacityResult struct {
	saveengine.ItemCapacity
	Kind schema.ResourceKind `json:"kind"`
	Key  string              `json:"key"`
}

// GetItemCapacity returns a snapshot-only preflight for adding quantity of one
// catalog resource to common Inventory or common Storage.
//
// The result is informational rather than a reservation. A later mutation must
// still supply and verify expectedRevision, because any successful mutation may
// invalidate these numbers immediately after this getter returns.
func GetItemCapacity(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	destination string,
	kind string,
	key string,
	variantID *uint32,
	quantity uint32,
) (GetItemCapacityResult, error) {
	if engine == nil {
		return GetItemCapacityResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetItemCapacityResult{}, errors.New("game catalog is not available")
	}
	switch destination {
	case saveengine.ItemCapacityDestinationInventory, saveengine.ItemCapacityDestinationStorage:
	default:
		return GetItemCapacityResult{}, fmt.Errorf(
			"destination must be %q or %q; got %q",
			saveengine.ItemCapacityDestinationInventory,
			saveengine.ItemCapacityDestinationStorage,
			destination)
	}

	resolved, err := resolveCommonItemAddition(gameCatalog, kind, key, variantID)
	if err != nil {
		return GetItemCapacityResult{}, err
	}
	if err := validateCommonItemAdditionSubcategory(resolved, kind, key); err != nil {
		return GetItemCapacityResult{}, err
	}
	item := resolved.resource.Item

	var maxContainerTotal, maxPerRecord uint32
	switch destination {
	case saveengine.ItemCapacityDestinationInventory:
		if !item.Storage.MaxInventory.Known || item.Storage.MaxInventory.Value == 0 {
			return GetItemCapacityResult{}, fmt.Errorf(
				"resource kind %q key %q carries no inventory limit", kind, key)
		}
		maxContainerTotal = item.Storage.MaxInventory.Value
		maxPerRecord = min(resolved.maxPerStack, maxContainerTotal)
	case saveengine.ItemCapacityDestinationStorage:
		maxContainerTotal, err = storageCommonItemAdditionLimit(resolved, kind, key)
		if err != nil {
			return GetItemCapacityResult{}, err
		}
		maxPerRecord = maxContainerTotal
	}

	capacity, err := engine.GetItemCapacity(
		saveSessionID,
		characterID,
		destination,
		resolved.gameID,
		quantity,
		resolved.separateInstances,
		maxPerRecord,
		maxContainerTotal,
	)
	if err != nil {
		return GetItemCapacityResult{}, err
	}
	return GetItemCapacityResult{
		ItemCapacity: capacity,
		Kind:         resolved.resource.Kind,
		Key:          resolved.resource.Key,
	}, nil
}
