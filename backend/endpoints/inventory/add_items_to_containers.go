/*
Endpoint: AddItemsToContainers
EndpointID: add_items_to_containers
Purpose: Adds several catalog resources to the common section of Inventory, of Storage, or of both, as one atomic mutation of one save revision.
How it works: The runtime handler resolves every requested (kind, key, variantID) through the already loaded GameCatalog, proves the same common-only add contract the single-record endpoints prove, derives the per-container limits from the shared Safety Profile policy, refuses every resource the active profile does not allow and every unconfirmed ban-risk resource, and then delegates one atomic batch mutation to SaveEngine. The endpoint opens no file, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument of family goods or talisman, outside category key_items; a Storage quantity additionally requires depositable goods.
Input variables: safetyProfile, saveSessionID, characterID, items, confirmBanRisk, expectedRevision.
Variant selection: the optional variantID selects the base item document when it is absent and exactly one stored variant of the same (kind, key) pair when it is present; gamecatalog.Catalog.ResourceByKindKeyAndVariant is the single implementation of that rule.
GameCatalog variables read: item.family, item.category, item.subcategory, item.gameID; item.storage.recordMode, item.storage.maxInventory, item.storage.safeModeMaxInventory, item.storage.maxStorage and item.storage.safeModeMaxStorage; item.capabilities.stack; item.goods.isDepositable for goods targeting Storage; item.safety.banRisk and item.safety.cutContent.
Save variables processed: for every requested addition either the four quantity bytes of its first common record or the twelve bytes of the first free common row, the common item count of the affected section, the two trailing allocators NextEquipIndex and NextAcquisitionSortId, and one active GaItemData entry per newly owned item; SaveEngine validates the complete batch against a private candidate image and finishes with full success or no change at all.
Implementation status: implemented
*/
package inventory

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// AddItemsToContainersEndpointID is the stable backend identifier of AddItemsToContainers.
const AddItemsToContainersEndpointID = "add_items_to_containers"

// AddItemsToContainersDefinition describes the public mutation contract.
var AddItemsToContainersDefinition = contract.MustDefine(contract.Definition{
	Name:                   "AddItemsToContainers",
	ID:                     AddItemsToContainersEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "ItemDocument of family goods or talisman, outside category key_items",
	SupportedResourceVariables: []string{
		"safetyProfile", "saveSessionID", "characterID", "items", "confirmBanRisk",
		"expectedRevision",
	},
	Description: "Adds several catalog resources to common Inventory, common Storage or both, as one atomic mutation.",
})

// AddItemsRequestEntry is one requested resource of the batch together with the
// two quantities the shared Add dialog collects. A quantity of zero means the
// entry does not address that container at all; an entry addressing neither is
// rejected rather than silently dropped.
type AddItemsRequestEntry struct {
	Kind              string  `json:"kind"`
	Key               string  `json:"key"`
	VariantID         *uint32 `json:"variantID,omitempty"`
	InventoryQuantity uint32  `json:"inventoryQuantity"`
	StorageQuantity   uint32  `json:"storageQuantity"`
}

// AddItemsToContainersResult is the public name of the receipt SaveEngine owns.
//
// The mutation and its result model belong to SaveEngine, so this is an alias
// rather than a copy: the endpoint adds no field, drops none and renames none.
type AddItemsToContainersResult = saveengine.AddItemsToContainersResult

// AddItemsToContainers adds every requested resource to the containers its
// entry names, as one mutation of one revision.
//
// The batch either applies completely or changes nothing: SaveEngine validates
// every step against a private candidate image of the snapshot and swaps that
// image in only after the last step succeeded. There is no partial result and
// no per-item retry, and one receipt with one operationID describes the whole
// change.
//
// The active Safety Profile decides two things and nothing else: which limit
// applies to each container, and whether a ban-risk or cut-content resource may
// be written at all. Both decisions come from the shared policy module, so a
// call that bypasses the interface is refused by exactly the same rule the
// interface renders. confirmBanRisk is the user's explicit confirmation and can
// never substitute for a profile that forbids the resource.
//
// That confirmation authorizes this one addition and nothing beyond it. The
// ban-risk fact of every written resource travels into SaveEngine with the
// addition, so the committed operation is recorded as a ban-risk operation,
// Review Changes counts it, and Save or Save As demands a second, independent
// confirmation for the session as a whole.
func AddItemsToContainers(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	items []AddItemsRequestEntry,
	confirmBanRisk bool,
	expectedRevision string,
) (AddItemsToContainersResult, error) {
	if engine == nil {
		return AddItemsToContainersResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return AddItemsToContainersResult{}, errors.New("game catalog is not available")
	}
	profile, err := safetyprofile.Parse(safetyProfile)
	if err != nil {
		return AddItemsToContainersResult{}, err
	}
	if len(items) == 0 {
		return AddItemsToContainersResult{}, errors.New("items must not be empty")
	}

	inventoryAdditions := make([]saveengine.ItemAddition, 0, len(items))
	storageAdditions := make([]saveengine.ItemAddition, 0, len(items))
	for index, entry := range items {
		if entry.InventoryQuantity == 0 && entry.StorageQuantity == 0 {
			return AddItemsToContainersResult{}, fmt.Errorf(
				"items[%d]: at least one of inventoryQuantity and storageQuantity must be positive",
				index)
		}
		resolved, err := resolveCommonItemAddition(gameCatalog, entry.Kind, entry.Key, entry.VariantID)
		if err != nil {
			return AddItemsToContainersResult{}, fmt.Errorf("items[%d]: %w", index, err)
		}
		item := resolved.resource.Item
		if err := safetyprofile.AllowMutation(profile, item, confirmBanRisk); err != nil {
			return AddItemsToContainersResult{}, fmt.Errorf(
				"items[%d] resource kind %q key %q: %w", index, entry.Kind, entry.Key, err)
		}

		if entry.InventoryQuantity > 0 {
			limit, known := safetyprofile.InventoryLimit(profile, item)
			if !known || limit == 0 {
				return AddItemsToContainersResult{}, fmt.Errorf(
					"items[%d]: resource kind %q key %q carries no inventory limit",
					index, entry.Kind, entry.Key)
			}
			inventoryAdditions = append(inventoryAdditions, saveengine.ItemAddition{
				GameID:            resolved.gameID,
				Quantity:          entry.InventoryQuantity,
				SeparateInstances: resolved.separateInstances,
				MaxPerRecord:      min(resolved.maxPerStack, limit),
				MaxContainerTotal: limit,
				BanRisk:           safetyprofile.RequiresBanRiskConfirmation(item),
			})
		}
		if entry.StorageQuantity > 0 {
			if err := validateCommonItemAdditionSubcategory(resolved, entry.Kind, entry.Key); err != nil {
				return AddItemsToContainersResult{}, fmt.Errorf("items[%d]: %w", index, err)
			}
			if _, err := storageCommonItemAdditionLimit(resolved, entry.Kind, entry.Key); err != nil {
				return AddItemsToContainersResult{}, fmt.Errorf("items[%d]: %w", index, err)
			}
			limit, known := safetyprofile.StorageLimit(profile, item)
			if !known || limit == 0 {
				return AddItemsToContainersResult{}, fmt.Errorf(
					"items[%d]: resource kind %q key %q carries no storage limit",
					index, entry.Kind, entry.Key)
			}
			storageAdditions = append(storageAdditions, saveengine.ItemAddition{
				GameID:            resolved.gameID,
				Quantity:          entry.StorageQuantity,
				SeparateInstances: resolved.separateInstances,
				MaxPerRecord:      min(resolved.maxPerStack, limit),
				MaxContainerTotal: limit,
				BanRisk:           safetyprofile.RequiresBanRiskConfirmation(item),
			})
		}
	}

	return engine.AddItemsToContainers(
		saveSessionID, characterID, inventoryAdditions, storageAdditions, expectedRevision)
}
